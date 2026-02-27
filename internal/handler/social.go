package handler

import (
	"net/http"
	"time"

	"github.com/attaboy/platform/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SocialHandler handles social interaction endpoints.
type SocialHandler struct {
	pool *pgxpool.Pool
}

// NewSocialHandler creates a new SocialHandler.
func NewSocialHandler(pool *pgxpool.Pool) *SocialHandler {
	return &SocialHandler{pool: pool}
}

// CreatePost handles POST /social/posts.
func (h *SocialHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	var input struct {
		Content    string `json:"content"`
		Type       string `json:"type"`
		TargetType string `json:"target_type,omitempty"`
		TargetID   string `json:"target_id,omitempty"`
	}
	if err := DecodeJSON(r, &input); err != nil {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	if input.Type == "" {
		input.Type = "text"
	}

	var targetID *uuid.UUID
	if input.TargetID != "" {
		parsed, err := uuid.Parse(input.TargetID)
		if err != nil {
			RespondError(w, domain.ErrValidation("invalid target_id"))
			return
		}
		targetID = &parsed
	}

	var postID uuid.UUID
	err = h.pool.QueryRow(r.Context(), `
		INSERT INTO social_posts (player_id, content, type, target_type, target_id)
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		playerID, input.Content, input.Type, input.TargetType, targetID).Scan(&postID)
	if err != nil {
		RespondError(w, domain.ErrInternal("create social post", err))
		return
	}

	RespondJSON(w, http.StatusCreated, map[string]string{"id": postID.String()})
}

// ListPosts handles GET /social/posts (public feed).
func (h *SocialHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), `
		SELECT sp.id, sp.player_id, sp.content, sp.type, sp.target_type, sp.target_id, sp.created_at
		FROM social_posts sp
		ORDER BY sp.created_at DESC LIMIT 50`)
	if err != nil {
		RespondError(w, domain.ErrInternal("list social posts", err))
		return
	}
	defer rows.Close()

	type post struct {
		ID         uuid.UUID  `json:"id"`
		PlayerID   uuid.UUID  `json:"player_id"`
		Content    string     `json:"content"`
		Type       string     `json:"type"`
		TargetType *string    `json:"target_type,omitempty"`
		TargetID   *uuid.UUID `json:"target_id,omitempty"`
		CreatedAt  time.Time  `json:"created_at"`
	}

	var posts []post
	for rows.Next() {
		var p post
		if err := rows.Scan(&p.ID, &p.PlayerID, &p.Content, &p.Type, &p.TargetType, &p.TargetID, &p.CreatedAt); err != nil {
			RespondError(w, domain.ErrInternal("scan social post", err))
			return
		}
		posts = append(posts, p)
	}

	RespondJSON(w, http.StatusOK, posts)
}

// DeletePost handles DELETE /social/posts/{id}.
func (h *SocialHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	playerID, err := playerIDFromContext(r)
	if err != nil {
		RespondError(w, err)
		return
	}

	postID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, domain.ErrValidation("invalid post id"))
		return
	}

	result, err := h.pool.Exec(r.Context(), `
		DELETE FROM social_posts WHERE id = $1 AND player_id = $2`,
		postID, playerID)
	if err != nil {
		RespondError(w, domain.ErrInternal("delete social post", err))
		return
	}

	if result.RowsAffected() == 0 {
		RespondError(w, domain.ErrNotFound("social post", postID.String()))
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
