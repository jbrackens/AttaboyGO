package policy

// RgLimitPolicy defines responsible gaming limits for a player.
type RgLimitPolicy struct {
	SingleTransactionMax int64 `json:"single_transaction_max"` // cents
	DailyDepositMax      int64 `json:"daily_deposit_max"`      // cents
	DailyLossMax         int64 `json:"daily_loss_max"`         // cents
}

// DefaultRgLimits returns the default RG limits (€1k single, €2k daily deposit, €1.5k daily loss).
func DefaultRgLimits() RgLimitPolicy {
	return RgLimitPolicy{
		SingleTransactionMax: 100_000, // €1,000
		DailyDepositMax:      200_000, // €2,000
		DailyLossMax:         150_000, // €1,500
	}
}

// RgEvaluation holds the result of an RG limits check.
type RgEvaluation struct {
	Allowed       bool   `json:"allowed"`
	BreachedLimit string `json:"breached_limit,omitempty"`
	LimitValue    int64  `json:"limit_value,omitempty"`
	RequestedAmt  int64  `json:"requested_amount,omitempty"`
}

// EvaluateRgLimits checks a transaction amount against the player's RG limits.
// dailyDeposits and dailyLosses are the running totals for the current day.
func EvaluateRgLimits(policy RgLimitPolicy, amount int64, txType string, dailyDeposits, dailyLosses int64) RgEvaluation {
	// Single transaction limit
	if policy.SingleTransactionMax > 0 && amount > policy.SingleTransactionMax {
		return RgEvaluation{
			Allowed:       false,
			BreachedLimit: "single_transaction",
			LimitValue:    policy.SingleTransactionMax,
			RequestedAmt:  amount,
		}
	}

	// Daily deposit limit
	if txType == "wallet_deposit" && policy.DailyDepositMax > 0 {
		if dailyDeposits+amount > policy.DailyDepositMax {
			return RgEvaluation{
				Allowed:       false,
				BreachedLimit: "daily_deposit",
				LimitValue:    policy.DailyDepositMax,
				RequestedAmt:  dailyDeposits + amount,
			}
		}
	}

	// Daily loss limit (applies to bets)
	if txType == "bet" && policy.DailyLossMax > 0 {
		if dailyLosses+amount > policy.DailyLossMax {
			return RgEvaluation{
				Allowed:       false,
				BreachedLimit: "daily_loss",
				LimitValue:    policy.DailyLossMax,
				RequestedAmt:  dailyLosses + amount,
			}
		}
	}

	return RgEvaluation{Allowed: true}
}
