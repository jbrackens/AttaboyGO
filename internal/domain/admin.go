package domain

import (
	"time"

	"github.com/google/uuid"
)

// AdminUser represents an admin_users row.
type AdminUser struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name"`
	Role         string    `json:"role"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PlayerNote represents an admin note on a player.
type PlayerNote struct {
	ID          uuid.UUID `json:"id"`
	PlayerID    uuid.UUID `json:"player_id"`
	AdminUserID uuid.UUID `json:"admin_user_id"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
}
