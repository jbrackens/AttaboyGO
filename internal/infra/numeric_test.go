package infra

import (
	"math"
	"math/big"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNumericToInt64_Zero(t *testing.T) {
	n := Int64ToNumeric(0)
	v, err := NumericToInt64(n)
	require.NoError(t, err)
	assert.Equal(t, int64(0), v)
}

func TestNumericToInt64_Positive(t *testing.T) {
	n := Int64ToNumeric(100000)
	v, err := NumericToInt64(n)
	require.NoError(t, err)
	assert.Equal(t, int64(100000), v)
}

func TestNumericToInt64_Negative(t *testing.T) {
	n := Int64ToNumeric(-50000)
	v, err := NumericToInt64(n)
	require.NoError(t, err)
	assert.Equal(t, int64(-50000), v)
}

func TestNumericToInt64_MaxBalance(t *testing.T) {
	// numeric(15,0) max is 999_999_999_999_999
	maxVal := int64(999_999_999_999_999)
	n := Int64ToNumeric(maxVal)
	v, err := NumericToInt64(n)
	require.NoError(t, err)
	assert.Equal(t, maxVal, v)
}

func TestNumericToInt64_NullReturnsError(t *testing.T) {
	n := pgtype.Numeric{Valid: false}
	_, err := NumericToInt64(n)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NULL")
}

func TestNumericToInt64_WithPositiveExponent(t *testing.T) {
	// 500 * 10^2 = 50000
	n := pgtype.Numeric{
		Int:   big.NewInt(500),
		Exp:   2,
		Valid: true,
	}
	v, err := NumericToInt64(n)
	require.NoError(t, err)
	assert.Equal(t, int64(50000), v)
}

func TestNumericToInt64_WithNegativeExponent(t *testing.T) {
	// 50099 * 10^-2 = 500 (truncated)
	n := pgtype.Numeric{
		Int:   big.NewInt(50099),
		Exp:   -2,
		Valid: true,
	}
	v, err := NumericToInt64(n)
	require.NoError(t, err)
	assert.Equal(t, int64(500), v)
}

func TestNumericToInt64_Overflow(t *testing.T) {
	overflow := new(big.Int).SetInt64(math.MaxInt64)
	overflow.Add(overflow, big.NewInt(1))
	n := pgtype.Numeric{
		Int:   overflow,
		Exp:   0,
		Valid: true,
	}
	_, err := NumericToInt64(n)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "overflows")
}

func TestInt64ToNumeric_Roundtrip(t *testing.T) {
	values := []int64{0, 1, -1, 100000, -100000, 999_999_999_999_999, math.MaxInt64, math.MinInt64}
	for _, v := range values {
		n := Int64ToNumeric(v)
		result, err := NumericToInt64(n)
		require.NoError(t, err, "value: %d", v)
		assert.Equal(t, v, result, "value: %d", v)
	}
}
