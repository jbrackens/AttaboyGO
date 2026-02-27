package settlement

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/attaboy/platform/internal/domain"
	"github.com/attaboy/platform/internal/ledger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GamificationSettlement handles the 3-gate reward evaluation and crediting.
type GamificationSettlement struct {
	engine *ledger.Engine
	config domain.GamificationConfig
}

// NewGamificationSettlement creates a gamification settlement handler.
func NewGamificationSettlement(engine *ledger.Engine, config domain.GamificationConfig) *GamificationSettlement {
	return &GamificationSettlement{engine: engine, config: config}
}

// GateResult captures which gates passed/failed.
type GateResult struct {
	EngagementScore   int  `json:"engagement_score"`
	ScorePassed       bool `json:"score_passed"`
	EligibilityPassed bool `json:"eligibility_passed"`
	BudgetPassed      bool `json:"budget_passed"`
	AllPassed         bool `json:"all_passed"`
}

// EvaluateAndCreditReward runs the 3-gate evaluation and credits the reward if all pass.
//
// Gate 1: Engagement score >= minimum (default 50)
// Gate 2: Eligibility (cooldown period since last reward)
// Gate 3: Budget (daily reward budget not exceeded)
func (s *GamificationSettlement) EvaluateAndCreditReward(
	ctx context.Context,
	tx pgx.Tx,
	playerID uuid.UUID,
	signals domain.EngagementSignals,
	rewardAmount int64,
	lastRewardAt *time.Time,
	dailyRewardsSpent int64,
) (*GateResult, *domain.CommandResult, error) {
	gate := &GateResult{}

	// Gate 1: Engagement score
	gate.EngagementScore = signals.ComputeScore()
	gate.ScorePassed = gate.EngagementScore >= s.config.MinEngagementScore

	// Gate 2: Eligibility (cooldown check)
	gate.EligibilityPassed = true
	if lastRewardAt != nil {
		cooldownEnd := lastRewardAt.Add(time.Duration(s.config.CooldownMinutes) * time.Minute)
		gate.EligibilityPassed = time.Now().After(cooldownEnd)
	}

	// Gate 3: Budget check
	gate.BudgetPassed = (dailyRewardsSpent + rewardAmount) <= s.config.DailyBudgetCents

	gate.AllPassed = gate.ScorePassed && gate.EligibilityPassed && gate.BudgetPassed

	if !gate.AllPassed {
		return gate, nil, nil
	}

	// All gates passed — credit as bonus
	meta, _ := json.Marshal(map[string]interface{}{
		"reward_type":      "gamification",
		"engagement_score": gate.EngagementScore,
	})
	result, err := s.engine.ExecuteBonusCredit(ctx, tx, domain.BonusCreditParams{
		PlayerID:              playerID,
		Amount:                rewardAmount,
		ExternalTransactionID: fmt.Sprintf("reward-%s-%d", playerID, time.Now().UnixNano()),
		Metadata:              meta,
	})
	if err != nil {
		return gate, nil, fmt.Errorf("credit reward: %w", err)
	}

	return gate, result, nil
}

// ComputeEngagementScore calculates the weighted score from signals.
func ComputeEngagementScore(signals domain.EngagementSignals) int {
	return signals.ComputeScore()
}

// CheckQuestEligibility checks if a player can claim a quest reward (cooldown gate).
func CheckQuestEligibility(lastRewardAt *time.Time, cooldownMinutes int) bool {
	if lastRewardAt == nil {
		return true
	}
	cooldownEnd := lastRewardAt.Add(time.Duration(cooldownMinutes) * time.Minute)
	return time.Now().After(cooldownEnd)
}

// CheckRewardBudget checks if the daily budget allows this reward.
func CheckRewardBudget(dailySpent, rewardAmount, dailyBudget int64) bool {
	return (dailySpent + rewardAmount) <= dailyBudget
}
