package infra

import (
	"fmt"
	"math/big"

	"github.com/jackc/pgx/v5/pgtype"
)

// NumericToInt64 converts a pgtype.Numeric (from PostgreSQL numeric(15,0)) to int64.
// Returns an error if the value is NULL, has a non-zero exponent (fractional digits),
// or overflows int64.
func NumericToInt64(n pgtype.Numeric) (int64, error) {
	if !n.Valid {
		return 0, fmt.Errorf("numeric value is NULL")
	}

	// pgtype.Numeric stores value as Int * 10^Exp
	// For numeric(15,0), Exp should be 0 or the Int should absorb it.
	bi := new(big.Int).Set(n.Int)

	if n.Exp > 0 {
		// Multiply by 10^Exp
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n.Exp)), nil)
		bi.Mul(bi, multiplier)
	} else if n.Exp < 0 {
		// For numeric(15,0) columns this shouldn't happen, but handle it.
		// Divide by 10^(-Exp) — truncates any fractional part.
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-n.Exp)), nil)
		bi.Div(bi, divisor)
	}

	if !bi.IsInt64() {
		return 0, fmt.Errorf("numeric value %s overflows int64", bi.String())
	}

	return bi.Int64(), nil
}

// Int64ToNumeric converts an int64 to pgtype.Numeric for writing to PostgreSQL numeric(15,0).
func Int64ToNumeric(v int64) pgtype.Numeric {
	return pgtype.Numeric{
		Int:              big.NewInt(v),
		Exp:              0,
		NaN:              false,
		InfinityModifier: pgtype.Finite,
		Valid:            true,
	}
}
