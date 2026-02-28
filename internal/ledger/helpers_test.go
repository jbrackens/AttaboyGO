package ledger

import (
	"encoding/json"
	"testing"

	"github.com/attaboy/platform/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- strPtr Tests ---

func TestStrPtr(t *testing.T) {
	t.Run("non-empty string", func(t *testing.T) {
		p := strPtr("hello")
		require.NotNil(t, p)
		assert.Equal(t, "hello", *p)
	})

	t.Run("empty string returns nil", func(t *testing.T) {
		p := strPtr("")
		assert.Nil(t, p)
	})
}

// --- ensureJSON Tests ---

func TestEnsureJSON(t *testing.T) {
	t.Run("nil returns empty object", func(t *testing.T) {
		result := ensureJSON(nil)
		assert.Equal(t, json.RawMessage(`{}`), result)
	})

	t.Run("non-nil passthrough", func(t *testing.T) {
		data := json.RawMessage(`{"key":"value"}`)
		result := ensureJSON(data)
		assert.Equal(t, data, result)
	})
}

// --- mergeMeta Tests ---

func TestMergeMeta(t *testing.T) {
	t.Run("nil base with extras", func(t *testing.T) {
		result := mergeMeta(nil, map[string]interface{}{"realBet": 100, "bonusBet": 50})
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(result, &m))
		assert.Equal(t, float64(100), m["realBet"])
		assert.Equal(t, float64(50), m["bonusBet"])
	})

	t.Run("existing base with extras", func(t *testing.T) {
		base := json.RawMessage(`{"gameId":"g1"}`)
		result := mergeMeta(base, map[string]interface{}{"realBet": 200})
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(result, &m))
		assert.Equal(t, "g1", m["gameId"])
		assert.Equal(t, float64(200), m["realBet"])
	})

	t.Run("extras overwrite base", func(t *testing.T) {
		base := json.RawMessage(`{"realBet":100}`)
		result := mergeMeta(base, map[string]interface{}{"realBet": 200})
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(result, &m))
		assert.Equal(t, float64(200), m["realBet"])
	})

	t.Run("empty extras", func(t *testing.T) {
		base := json.RawMessage(`{"key":"val"}`)
		result := mergeMeta(base, map[string]interface{}{})
		var m map[string]interface{}
		require.NoError(t, json.Unmarshal(result, &m))
		assert.Equal(t, "val", m["key"])
	})
}

// --- extractBetSplit Tests ---

func TestExtractBetSplit(t *testing.T) {
	t.Run("valid split metadata", func(t *testing.T) {
		tx := &domain.Transaction{
			Amount:   15000,
			Metadata: json.RawMessage(`{"realBet":10000,"bonusBet":5000}`),
		}
		real, bonus := extractBetSplit(tx)
		assert.Equal(t, int64(10000), real)
		assert.Equal(t, int64(5000), bonus)
	})

	t.Run("no metadata falls back to amount", func(t *testing.T) {
		tx := &domain.Transaction{
			Amount:   15000,
			Metadata: nil,
		}
		real, bonus := extractBetSplit(tx)
		assert.Equal(t, int64(15000), real)
		assert.Equal(t, int64(0), bonus)
	})

	t.Run("empty metadata falls back to amount", func(t *testing.T) {
		tx := &domain.Transaction{
			Amount:   8000,
			Metadata: json.RawMessage(`{}`),
		}
		real, bonus := extractBetSplit(tx)
		assert.Equal(t, int64(8000), real)
		assert.Equal(t, int64(0), bonus)
	})

	t.Run("invalid JSON falls back to amount", func(t *testing.T) {
		tx := &domain.Transaction{
			Amount:   5000,
			Metadata: json.RawMessage(`invalid`),
		}
		real, bonus := extractBetSplit(tx)
		assert.Equal(t, int64(5000), real)
		assert.Equal(t, int64(0), bonus)
	})

	t.Run("only realBet in metadata", func(t *testing.T) {
		tx := &domain.Transaction{
			Amount:   7000,
			Metadata: json.RawMessage(`{"realBet":7000}`),
		}
		real, bonus := extractBetSplit(tx)
		assert.Equal(t, int64(7000), real)
		assert.Equal(t, int64(0), bonus)
	})
}

// --- extractWinSplit Tests ---

func TestExtractWinSplit(t *testing.T) {
	t.Run("valid split metadata", func(t *testing.T) {
		tx := &domain.Transaction{
			Amount:   20000,
			Metadata: json.RawMessage(`{"realWin":15000,"bonusWin":5000}`),
		}
		real, bonus := extractWinSplit(tx)
		assert.Equal(t, int64(15000), real)
		assert.Equal(t, int64(5000), bonus)
	})

	t.Run("no metadata falls back to amount", func(t *testing.T) {
		tx := &domain.Transaction{
			Amount:   20000,
			Metadata: nil,
		}
		real, bonus := extractWinSplit(tx)
		assert.Equal(t, int64(20000), real)
		assert.Equal(t, int64(0), bonus)
	})

	t.Run("all bonus win", func(t *testing.T) {
		tx := &domain.Transaction{
			Amount:   10000,
			Metadata: json.RawMessage(`{"realWin":0,"bonusWin":10000}`),
		}
		real, bonus := extractWinSplit(tx)
		// realWin=0, bonusWin=10000, sum=10000 > 0 so no fallback
		assert.Equal(t, int64(0), real)
		assert.Equal(t, int64(10000), bonus)
	})
}

// --- jsonUnmarshal Tests ---

func TestJsonUnmarshal(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		var m map[string]interface{}
		err := jsonUnmarshal(json.RawMessage(`{"key":"val"}`), &m)
		require.NoError(t, err)
		assert.Equal(t, "val", m["key"])
	})

	t.Run("nil data returns error", func(t *testing.T) {
		var m map[string]interface{}
		err := jsonUnmarshal(nil, &m)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil json data")
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		var m map[string]interface{}
		err := jsonUnmarshal(json.RawMessage(`{invalid`), &m)
		require.Error(t, err)
	})
}
