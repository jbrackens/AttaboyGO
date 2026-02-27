package admin

import (
	"net/http"
	"time"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ModerationHandler handles admin content moderation.
type ModerationHandler struct {
	pool *pgxpool.Pool
}

// NewModerationHandler creates a new ModerationHandler.
func NewModerationHandler(pool *pgxpool.Pool) *ModerationHandler {
	return &ModerationHandler{pool: pool}
}

// ListFlaggedPosts handles GET /admin/moderation/posts.
func (h *ModerationHandler) ListFlaggedPosts(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), `
		SELECT sp.id, sp.player_id, sp.content, sp.type, sp.created_at
		FROM social_posts sp
		ORDER BY sp.created_at DESC LIMIT 50`)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("list posts for moderation", err))
		return
	}
	defer rows.Close()

	type moderatedPost struct {
		ID        uuid.UUID `json:"id"`
		PlayerID  uuid.UUID `json:"player_id"`
		Content   string    `json:"content"`
		Type      string    `json:"type"`
		CreatedAt time.Time `json:"created_at"`
	}

	var posts []moderatedPost
	for rows.Next() {
		var p moderatedPost
		if err := rows.Scan(&p.ID, &p.PlayerID, &p.Content, &p.Type, &p.CreatedAt); err != nil {
			handler.RespondError(w, domain.ErrInternal("scan moderated post", err))
			return
		}
		posts = append(posts, p)
	}

	handler.RespondJSON(w, http.StatusOK, posts)
}

// DeletePost handles DELETE /admin/moderation/posts/{id}.
func (h *ModerationHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	postID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		handler.RespondError(w, domain.ErrValidation("invalid post id"))
		return
	}

	result, err := h.pool.Exec(r.Context(), `DELETE FROM social_posts WHERE id = $1`, postID)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("delete post", err))
		return
	}

	if result.RowsAffected() == 0 {
		handler.RespondError(w, domain.ErrNotFound("social post", postID.String()))
		return
	}

	handler.RespondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ListPluginDispatches handles GET /admin/moderation/dispatches.
func (h *ModerationHandler) ListPluginDispatches(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, plugin_id, scope, status, error, created_at
		FROM plugin_dispatches
		ORDER BY created_at DESC LIMIT 50`)
	if err != nil {
		handler.RespondError(w, domain.ErrInternal("list dispatches", err))
		return
	}
	defer rows.Close()

	type dispatch struct {
		ID        uuid.UUID `json:"id"`
		PluginID  string    `json:"plugin_id"`
		Scope     string    `json:"scope"`
		Status    string    `json:"status"`
		Error     *string   `json:"error,omitempty"`
		CreatedAt time.Time `json:"created_at"`
	}

	var dispatches []dispatch
	for rows.Next() {
		var d dispatch
		if err := rows.Scan(&d.ID, &d.PluginID, &d.Scope, &d.Status, &d.Error, &d.CreatedAt); err != nil {
			handler.RespondError(w, domain.ErrInternal("scan dispatch", err))
			return
		}
		dispatches = append(dispatches, d)
	}

	handler.RespondJSON(w, http.StatusOK, dispatches)
}
