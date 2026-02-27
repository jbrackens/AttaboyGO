package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DualWriteBridge enables parallel writes to V1 and V2 during migration cutover.
// Feature-flagged: when disabled, writes only go to V2.
type DualWriteBridge struct {
	pool    *pgxpool.Pool
	enabled bool
	logger  *slog.Logger
}

// NewDualWriteBridge creates a new dual-write bridge.
func NewDualWriteBridge(pool *pgxpool.Pool, enabled bool, logger *slog.Logger) *DualWriteBridge {
	return &DualWriteBridge{pool: pool, enabled: enabled, logger: logger}
}

// DeterministicUUID generates a UUID from a V1 player ID using SHA256.
// This ensures the same V1 ID always maps to the same V2 UUID across systems.
func DeterministicUUID(namespace, v1ID string) uuid.UUID {
	h := sha256.New()
	h.Write([]byte(namespace))
	h.Write([]byte(":"))
	h.Write([]byte(v1ID))
	digest := h.Sum(nil)

	// Use first 16 bytes as UUID, set version 5 (SHA-based)
	var id uuid.UUID
	copy(id[:], digest[:16])
	id[6] = (id[6] & 0x0f) | 0x50 // version 5
	id[8] = (id[8] & 0x3f) | 0x80 // variant RFC4122
	return id
}

// DeterministicUUIDHex returns the hex string of the deterministic UUID.
func DeterministicUUIDHex(namespace, v1ID string) string {
	return DeterministicUUID(namespace, v1ID).String()
}

// SHA256Hex returns the full SHA256 hex digest of namespace:v1ID.
func SHA256Hex(namespace, v1ID string) string {
	h := sha256.New()
	h.Write([]byte(namespace))
	h.Write([]byte(":"))
	h.Write([]byte(v1ID))
	return hex.EncodeToString(h.Sum(nil))
}

// MigratePlayer migrates a V1 player to V2, creating a deterministic UUID.
func (b *DualWriteBridge) MigratePlayer(ctx context.Context, v1PlayerID string, email string) (uuid.UUID, error) {
	v2ID := DeterministicUUID("player", v1PlayerID)

	_, err := b.pool.Exec(ctx, `
		INSERT INTO v2_players (id, balance, bonus_balance, reserved_balance, currency, status)
		VALUES ($1, 0, 0, 0, 'EUR', 'active')
		ON CONFLICT (id) DO NOTHING`, v2ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert v2 player: %w", err)
	}

	b.logger.Info("migrated player",
		"v1_id", v1PlayerID,
		"v2_id", v2ID,
		"email", email)

	return v2ID, nil
}

// IsEnabled returns whether dual-write is active.
func (b *DualWriteBridge) IsEnabled() bool {
	return b.enabled
}
