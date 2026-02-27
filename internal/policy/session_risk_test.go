package policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateSessionRisk_LowRisk(t *testing.T) {
	result := EvaluateSessionRisk(SessionRiskSignals{
		RequestVelocity: 10,
		AuthFailures:    0,
	})
	assert.Equal(t, RiskLow, result.Level)
	assert.Equal(t, 0, result.Score)
	assert.Empty(t, result.Flags)
}

func TestEvaluateSessionRisk_MediumRisk(t *testing.T) {
	result := EvaluateSessionRisk(SessionRiskSignals{
		RequestVelocity: 60,
		AuthFailures:    3,
		GeoAnomaly:      false,
	})
	assert.Equal(t, RiskMedium, result.Level)
	assert.Contains(t, result.Flags, "elevated_velocity")
	assert.Contains(t, result.Flags, "auth_failures_moderate")
}

func TestEvaluateSessionRisk_HighRisk(t *testing.T) {
	result := EvaluateSessionRisk(SessionRiskSignals{
		RequestVelocity:  120,
		AuthFailures:     6,
		GeoAnomaly:       true,
		MultipleAccounts: true,
	})
	assert.Equal(t, RiskHigh, result.Level)
	assert.True(t, result.Score >= 60)
}

func TestEvaluateSessionRisk_GeoAnomalyAddsScore(t *testing.T) {
	result := EvaluateSessionRisk(SessionRiskSignals{GeoAnomaly: true})
	assert.Equal(t, 25, result.Score)
	assert.Contains(t, result.Flags, "geo_anomaly")
}

func TestEvaluateSessionRisk_DeviceChangedAddsScore(t *testing.T) {
	result := EvaluateSessionRisk(SessionRiskSignals{DeviceChanged: true})
	assert.Equal(t, 15, result.Score)
	assert.Contains(t, result.Flags, "device_changed")
}
