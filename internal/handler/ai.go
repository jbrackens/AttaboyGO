package handler

import (
	"net/http"
	"time"

	"github.com/attaboy/platform/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AIHandler handles AI conversation endpoints.
type AIHandler struct {
	pool *pgxpool.Pool
}

// NewAIHandler creates a new AIHandler.
func NewAIHandler(pool *pgxpool.Pool) *AIHandler {
	return &AIHandler{pool: pool}
}

// CreateConversation handles POST /ai/conversations.
func (h *AIHandler) CreateConversation(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	var convID uuid.UUID
	err = h.pool.QueryRow(r.Context(), `
		INSERT INTO ai_conversations (player_id) VALUES ($1) RETURNING id`,
		playerID).Scan(&convID)
	if err != nil {
		RespondError(w, domain.ErrInternal("create conversation", err))
		return
	}

	RespondJSON(w, http.StatusCreated, map[string]string{"id": convID.String()})
}

// ListConversations handles GET /ai/conversations.
func (h *AIHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT id, created_at, updated_at
		FROM ai_conversations
		WHERE player_id = $1
		ORDER BY updated_at DESC LIMIT 20`, playerID)
	if err != nil {
		RespondError(w, domain.ErrInternal("list conversations", err))
		return
	}
	defer rows.Close()

	type conversation struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	var conversations []conversation
	for rows.Next() {
		var c conversation
		if err := rows.Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			RespondError(w, domain.ErrInternal("scan conversation", err))
			return
		}
		conversations = append(conversations, c)
	}

	RespondJSON(w, http.StatusOK, conversations)
}

// SendMessage handles POST /ai/conversations/{id}/messages.
func (h *AIHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	convID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, domain.ErrValidation("invalid conversation id"))
		return
	}

	// Verify ownership
	var ownerID uuid.UUID
	err = h.pool.QueryRow(r.Context(), `SELECT player_id FROM ai_conversations WHERE id = $1`, convID).Scan(&ownerID)
	if err != nil || ownerID != playerID {
		RespondError(w, domain.ErrNotFound("conversation", convID.String()))
		return
	}

	var input struct {
		Content string `json:"content"`
	}
	if err := DecodeJSON(r, &input); err != nil {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	// Insert user message
	var msgID uuid.UUID
	err = h.pool.QueryRow(r.Context(), `
		INSERT INTO ai_messages (conversation_id, role, content)
		VALUES ($1, 'user', $2) RETURNING id`,
		convID, input.Content).Scan(&msgID)
	if err != nil {
		RespondError(w, domain.ErrInternal("save message", err))
		return
	}

	// Update conversation timestamp
	_, _ = h.pool.Exec(r.Context(), `UPDATE ai_conversations SET updated_at = now() WHERE id = $1`, convID)

	RespondJSON(w, http.StatusCreated, map[string]string{
		"id":   msgID.String(),
		"role": "user",
	})
}

// GetMessages handles GET /ai/conversations/{id}/messages.
func (h *AIHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	convID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, domain.ErrValidation("invalid conversation id"))
		return
	}

	// Verify ownership
	var ownerID uuid.UUID
	err = h.pool.QueryRow(r.Context(), `SELECT player_id FROM ai_conversations WHERE id = $1`, convID).Scan(&ownerID)
	if err != nil || ownerID != playerID {
		RespondError(w, domain.ErrNotFound("conversation", convID.String()))
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT id, role, content, blocked, created_at
		FROM ai_messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC`, convID)
	if err != nil {
		RespondError(w, domain.ErrInternal("list messages", err))
		return
	}
	defer rows.Close()

	type message struct {
		ID        uuid.UUID `json:"id"`
		Role      string    `json:"role"`
		Content   string    `json:"content"`
		Blocked   bool      `json:"blocked"`
		CreatedAt time.Time `json:"created_at"`
	}

	var messages []message
	for rows.Next() {
		var m message
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &m.Blocked, &m.CreatedAt); err != nil {
			RespondError(w, domain.ErrInternal("scan message", err))
			return
		}
		messages = append(messages, m)
	}

	RespondJSON(w, http.StatusOK, messages)
}
