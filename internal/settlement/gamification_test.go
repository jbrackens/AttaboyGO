package settlement

import (
	"testing"
	"time"

	"github.com/attaboy/platform/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestComputeEngagementScore(t *testing.T) {
	// Formula: videoMinutes*2 + socialInteractions*3 + predictionActions*5
	score := ComputeEngagementScore(domain.EngagementSignals{
		VideoMinutes:       10,
		SocialInteractions: 5,
		PredictionActions:  3,
	})
	assert.Equal(t, 10*2+5*3+3*5, score) // 20+15+15 = 50
}

func TestComputeEngagementScore_Zero(t *testing.T) {
	score := ComputeEngagementScore(domain.EngagementSignals{})
	assert.Equal(t, 0, score)
}

func TestCheckQuestEligibility_NoPreviousReward(t *testing.T) {
	assert.True(t, CheckQuestEligibility(nil, 60))
}

func TestCheckQuestEligibility_CooldownActive(t *testing.T) {
	recent := time.Now().Add(-30 * time.Minute) // 30 min ago
	assert.False(t, CheckQuestEligibility(&recent, 60))
}

func TestCheckQuestEligibility_CooldownExpired(t *testing.T) {
	old := time.Now().Add(-90 * time.Minute) // 90 min ago
	assert.True(t, CheckQuestEligibility(&old, 60))
}

func TestCheckRewardBudget_WithinBudget(t *testing.T) {
	assert.True(t, CheckRewardBudget(100_000, 50_000, 250_000))
}

func TestCheckRewardBudget_ExceedsBudget(t *testing.T) {
	assert.False(t, CheckRewardBudget(200_000, 60_000, 250_000))
}

func TestCheckRewardBudget_ExactlyAtBudget(t *testing.T) {
	assert.True(t, CheckRewardBudget(200_000, 50_000, 250_000))
}
