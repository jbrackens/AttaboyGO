package projection

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStore_SetAndGet(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	err := store.Set(ctx, "k1", []byte("hello"), 0)
	require.NoError(t, err)

	val, err := store.Get(ctx, "k1")
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), val)
}

func TestInMemoryStore_KeyNotFound(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	_, err := store.Get(ctx, "missing")
	assert.Error(t, err)
}

func TestInMemoryStore_Delete(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	_ = store.Set(ctx, "k1", []byte("data"), 0)
	_ = store.Delete(ctx, "k1")

	_, err := store.Get(ctx, "k1")
	assert.Error(t, err)
}

func TestInMemoryStore_TTLExpiry(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	_ = store.Set(ctx, "k1", []byte("data"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	_, err := store.Get(ctx, "k1")
	assert.Error(t, err)
}

func TestBalanceProjection_RoundTrip(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	p := BalanceProjection{
		PlayerID:        "abc-123",
		Balance:         100000,
		BonusBalance:    50000,
		ReservedBalance: 0,
	}

	err := UpdateBalance(ctx, store, p)
	require.NoError(t, err)

	got, err := GetBalance(ctx, store, "abc-123")
	require.NoError(t, err)
	assert.Equal(t, int64(100000), got.Balance)
	assert.Equal(t, int64(50000), got.BonusBalance)
	assert.NotEmpty(t, got.UpdatedAt)
}

func TestBalanceProjection_Invalidate(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	_ = UpdateBalance(ctx, store, BalanceProjection{PlayerID: "abc-123", Balance: 100})
	_ = InvalidateBalance(ctx, store, "abc-123")

	_, err := GetBalance(ctx, store, "abc-123")
	assert.Error(t, err)
}
