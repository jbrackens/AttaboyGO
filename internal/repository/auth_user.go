package repository

import (
	"context"
	"errors"

	"github.com/attaboy/platform/internal/domain"
	"github.com/jackc/pgx/v5"
)

// PgAuthUserRepository implements AuthUserRepository using pgx.
type PgAuthUserRepository struct{}

// NewPgAuthUserRepository creates a new PgAuthUserRepository.
func NewPgAuthUserRepository() *PgAuthUserRepository {
	return &PgAuthUserRepository{}
}

// FindByEmail returns an auth user by email, or nil if not found.
func (r *PgAuthUserRepository) FindByEmail(ctx context.Context, db DBTX, email string) (*domain.AuthUser, error) {
	row := db.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at, updated_at
		 FROM auth_users WHERE email = $1`, email)

	u := &domain.AuthUser{}
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// Create inserts a new auth user.
func (r *PgAuthUserRepository) Create(ctx context.Context, db DBTX, user *domain.AuthUser) error {
	_, err := db.Exec(ctx,
		`INSERT INTO auth_users (id, email, password_hash) VALUES ($1, $2, $3)`,
		user.ID, user.Email, user.PasswordHash)
	return err
}

// UpdatePasswordHash updates the password hash for the given email.
func (r *PgAuthUserRepository) UpdatePasswordHash(ctx context.Context, db DBTX, email, hash string) error {
	tag, err := db.Exec(ctx,
		`UPDATE auth_users SET password_hash = $1, updated_at = now() WHERE email = $2`,
		hash, email)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound("user", email)
	}
	return nil
}
