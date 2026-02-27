package projection

import (
	"context"
	"fmt"
	"time"
)

// BalanceProjection represents a cached player balance.
type BalanceProjection struct {
	PlayerID        string `json:"player_id"`
	Balance         int64  `json:"balance"`
	BonusBalance    int64  `json:"bonus_balance"`
	ReservedBalance int64  `json:"reserved_balance"`
	UpdatedAt       string `json:"updated_at"`
}

const balanceTTL = 5 * time.Minute

// UpdateBalance caches a player's balance projection.
func UpdateBalance(ctx context.Context, store Store, p BalanceProjection) error {
	p.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	key := fmt.Sprintf("projection:balance:%s", p.PlayerID)
	return SetJSON(ctx, store, key, p, balanceTTL)
}

// GetBalance retrieves a cached player balance projection.
func GetBalance(ctx context.Context, store Store, playerID string) (*BalanceProjection, error) {
	key := fmt.Sprintf("projection:balance:%s", playerID)
	var p BalanceProjection
	if err := GetJSON(ctx, store, key, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// InvalidateBalance removes a player's cached balance.
func InvalidateBalance(ctx context.Context, store Store, playerID string) error {
	key := fmt.Sprintf("projection:balance:%s", playerID)
	return store.Delete(ctx, key)
}
