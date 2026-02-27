package provider

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSPRNGFallback_GeneratesInRange(t *testing.T) {
	// No API key → falls back to CSPRNG
	client := NewRandomOrgClient("", slog.New(slog.NewTextHandler(os.Stderr, nil)))

	nums, err := client.RandomIntegers(context.Background(), 10, 1, 100)
	require.NoError(t, err)
	assert.Len(t, nums, 10)

	for _, n := range nums {
		assert.GreaterOrEqual(t, n, 1)
		assert.LessOrEqual(t, n, 100)
	}
}

func TestCSPRNGFallback_SingleValue(t *testing.T) {
	client := NewRandomOrgClient("", slog.New(slog.NewTextHandler(os.Stderr, nil)))

	nums, err := client.RandomIntegers(context.Background(), 1, 0, 1)
	require.NoError(t, err)
	assert.Len(t, nums, 1)
	assert.Contains(t, []int{0, 1}, nums[0])
}

func TestCSPRNGFallback_MinEqualsMax(t *testing.T) {
	client := NewRandomOrgClient("", slog.New(slog.NewTextHandler(os.Stderr, nil)))

	nums, err := client.RandomIntegers(context.Background(), 5, 42, 42)
	require.NoError(t, err)
	assert.Len(t, nums, 5)
	for _, n := range nums {
		assert.Equal(t, 42, n)
	}
}

func TestCSPRNGIntegers_InvalidRange(t *testing.T) {
	_, err := csprngIntegers(1, 100, 50)
	assert.Error(t, err)
}
