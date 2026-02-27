package policy

// RiskLevel classifies session risk.
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// SessionRiskSignals holds the raw inputs for risk evaluation.
type SessionRiskSignals struct {
	RequestVelocity  int  `json:"request_velocity"`   // requests in last 5min
	AuthFailures     int  `json:"auth_failures"`       // failed logins in last hour
	GeoAnomaly       bool `json:"geo_anomaly"`         // IP location differs from profile
	DeviceChanged    bool `json:"device_changed"`      // new device fingerprint
	MultipleAccounts bool `json:"multiple_accounts"`   // IP shared with other accounts
}

// SessionRiskResult holds the evaluated risk.
type SessionRiskResult struct {
	Level  RiskLevel `json:"level"`
	Score  int       `json:"score"`
	Flags  []string  `json:"flags,omitempty"`
}

// EvaluateSessionRisk computes a risk score from session signals.
func EvaluateSessionRisk(signals SessionRiskSignals) SessionRiskResult {
	var score int
	var flags []string

	if signals.RequestVelocity > 100 {
		score += 30
		flags = append(flags, "high_velocity")
	} else if signals.RequestVelocity > 50 {
		score += 15
		flags = append(flags, "elevated_velocity")
	}

	if signals.AuthFailures > 5 {
		score += 40
		flags = append(flags, "auth_failures")
	} else if signals.AuthFailures > 2 {
		score += 20
		flags = append(flags, "auth_failures_moderate")
	}

	if signals.GeoAnomaly {
		score += 25
		flags = append(flags, "geo_anomaly")
	}

	if signals.DeviceChanged {
		score += 15
		flags = append(flags, "device_changed")
	}

	if signals.MultipleAccounts {
		score += 20
		flags = append(flags, "multiple_accounts")
	}

	level := RiskLow
	if score >= 60 {
		level = RiskHigh
	} else if score >= 30 {
		level = RiskMedium
	}

	return SessionRiskResult{Level: level, Score: score, Flags: flags}
}
