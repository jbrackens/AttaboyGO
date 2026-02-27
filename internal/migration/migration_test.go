package migration

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testLogger = slog.New(slog.NewTextHandler(os.Stderr, nil))

func TestDeterministicUUID_Consistency(t *testing.T) {
	// Same input always produces same UUID
	id1 := DeterministicUUID("player", "v1-player-123")
	id2 := DeterministicUUID("player", "v1-player-123")
	assert.Equal(t, id1, id2)
}

func TestDeterministicUUID_DifferentInputs(t *testing.T) {
	id1 := DeterministicUUID("player", "v1-player-123")
	id2 := DeterministicUUID("player", "v1-player-456")
	assert.NotEqual(t, id1, id2)
}

func TestDeterministicUUID_DifferentNamespaces(t *testing.T) {
	id1 := DeterministicUUID("player", "123")
	id2 := DeterministicUUID("transaction", "123")
	assert.NotEqual(t, id1, id2)
}

func TestDeterministicUUID_ValidVersion(t *testing.T) {
	id := DeterministicUUID("player", "test-id")
	// Version should be 5 (SHA-based)
	version := id[6] >> 4
	assert.Equal(t, byte(5), version)
}

func TestDeterministicUUID_ValidVariant(t *testing.T) {
	id := DeterministicUUID("player", "test-id")
	// Variant should be RFC4122 (10xx xxxx)
	variant := id[8] >> 6
	assert.Equal(t, byte(2), variant)
}

func TestDeterministicUUIDHex(t *testing.T) {
	hex := DeterministicUUIDHex("player", "test-123")
	assert.Len(t, hex, 36) // UUID format: 8-4-4-4-12
	assert.Contains(t, hex, "-")
}

func TestSHA256Hex(t *testing.T) {
	hex := SHA256Hex("player", "test-123")
	assert.Len(t, hex, 64) // SHA256 = 32 bytes = 64 hex chars
}

func TestSHA256Hex_Consistent(t *testing.T) {
	h1 := SHA256Hex("player", "test-123")
	h2 := SHA256Hex("player", "test-123")
	assert.Equal(t, h1, h2)
}

func TestTransactionMapper_MapTransaction(t *testing.T) {
	mapper := &TransactionMapper{logger: testLogger}

	v1Tx := V1Transaction{
		ID:       "old-tx-001",
		PlayerID: "old-player-001",
		Type:     "deposit",
		Amount:   10000,
	}

	v2TxID, err := mapper.MapTransaction(v1Tx)
	require.NoError(t, err)

	// Verify deterministic: same input produces same output
	v2TxID2, _ := mapper.MapTransaction(v1Tx)
	assert.Equal(t, v2TxID, v2TxID2)
}
