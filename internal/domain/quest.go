package domain

import (
	"time"

	"github.com/google/uuid"
)

// EngagementSignals holds the raw activity counts for gamification scoring.
type EngagementSignals struct {
	VideoMinutes       int `json:"video_minutes"`
	SocialInteractions int `json:"social_interactions"`
	PredictionActions  int `json:"prediction_actions"`
}

// ComputeScore calculates the weighted engagement score.
// Formula: videoMinutes*2 + socialInteractions*3 + predictionActions*5
func (s EngagementSignals) ComputeScore() int {
	return s.VideoMinutes*2 + s.SocialInteractions*3 + s.PredictionActions*5
}

// GamificationConfig holds configurable parameters for the 3-gate system.
type GamificationConfig struct {
	MinEngagementScore int   `json:"min_engagement_score"` // default 50
	CooldownMinutes    int   `json:"cooldown_minutes"`     // default 60
	DailyBudgetCents   int64 `json:"daily_budget_cents"`   // default 250_000 (€2,500)
}

// DefaultGamificationConfig returns the default configuration.
func DefaultGamificationConfig() GamificationConfig {
	return GamificationConfig{
		MinEngagementScore: 50,
		CooldownMinutes:    60,
		DailyBudgetCents:   250_000,
	}
}

// Quest represents a gamification quest.
type Quest struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	RewardType  string     `json:"reward_type"` // bonus_credit, free_spin, etc.
	RewardValue int64      `json:"reward_value"`
	Active      bool       `json:"active"`
	StartsAt    *time.Time `json:"starts_at,omitempty"`
	EndsAt      *time.Time `json:"ends_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// QuestProgress tracks a player's progress on a quest.
type QuestProgress struct {
	ID         uuid.UUID  `json:"id"`
	PlayerID   uuid.UUID  `json:"player_id"`
	QuestID    uuid.UUID  `json:"quest_id"`
	Progress   int        `json:"progress"`
	Target     int        `json:"target"`
	Completed  bool       `json:"completed"`
	ClaimedAt  *time.Time `json:"claimed_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Reward represents a reward credited through the gamification system.
type Reward struct {
	ID        uuid.UUID `json:"id"`
	PlayerID  uuid.UUID `json:"player_id"`
	QuestID   uuid.UUID `json:"quest_id"`
	Type      string    `json:"type"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}
