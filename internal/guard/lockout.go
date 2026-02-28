package guard

import (
	"context"
	"time"

	"github.com/attaboy/platform/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	MaxAttempts   = 5
	LockoutWindow = 15 * time.Minute
)

// RecordAttempt inserts a login attempt row.
func RecordAttempt(ctx context.Context, pool *pgxpool.Pool, email, realm, ip string, success bool) {
	_, _ = pool.Exec(ctx, `
		INSERT INTO login_attempts (email, realm, ip_address, success)
		VALUES ($1, $2, $3, $4)`,
		email, realm, ip, success)
}

// CheckLocked returns ErrAccountLocked if the account has >= MaxAttempts failed
// logins within the lockout window.
func CheckLocked(ctx context.Context, pool *pgxpool.Pool, email, realm string) error {
	var count int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM login_attempts
		WHERE email = $1 AND realm = $2 AND success = false
		  AND created_at > $3`,
		email, realm, time.Now().Add(-LockoutWindow)).Scan(&count)
	if err != nil {
		return nil // fail open on DB error — don't block login
	}
	if count >= MaxAttempts {
		return domain.ErrAccountLocked("too many failed login attempts, try again later")
	}
	return nil
}
