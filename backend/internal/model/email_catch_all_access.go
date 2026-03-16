package model

import "time"

// EmailCatchAllAccess stores the mutable runtime state that governs whether one
// approved catch-all mailbox can still receive mail.
type EmailCatchAllAccess struct {
	UserID                   int64      `json:"user_id"`
	SubscriptionExpiresAt    *time.Time `json:"subscription_expires_at,omitempty"`
	RemainingCount           int64      `json:"remaining_count"`
	TemporaryRewardCount     int64      `json:"temporary_reward_count"`
	TemporaryRewardExpiresAt *time.Time `json:"temporary_reward_expires_at,omitempty"`
	DailyLimitOverride       *int64     `json:"daily_limit_override,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

// EmailCatchAllDailyUsage stores one user's already-consumed catch-all traffic
// for one canonical Shanghai-local day.
type EmailCatchAllDailyUsage struct {
	UserID    int64     `json:"user_id"`
	UsageDate string    `json:"usage_date"`
	UsedCount int64     `json:"used_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// EmailCatchAllConsumeResult returns the post-consumption state after the
// relay reserves one or more catch-all deliveries.
type EmailCatchAllConsumeResult struct {
	Access                           EmailCatchAllAccess     `json:"access"`
	DailyUsage                       EmailCatchAllDailyUsage `json:"daily_usage"`
	EffectiveDailyLimit              int64                   `json:"effective_daily_limit"`
	ConsumedMode                     string                  `json:"consumed_mode"`
	ConsumedPermanentCount           int64                   `json:"consumed_permanent_count"`
	ConsumedTemporaryRewardCount     int64                   `json:"consumed_temporary_reward_count"`
	ConsumedTemporaryRewardExpiresAt *time.Time              `json:"consumed_temporary_reward_expires_at,omitempty"`
}

// ActiveTemporaryRewardCount returns the still-usable PoW reward balance at
// the provided time. Expired temporary rewards are treated as zero.
func (item EmailCatchAllAccess) ActiveTemporaryRewardCount(now time.Time) int64 {
	if item.TemporaryRewardCount <= 0 || item.TemporaryRewardExpiresAt == nil {
		return 0
	}
	if !item.TemporaryRewardExpiresAt.After(now.UTC()) {
		return 0
	}
	return item.TemporaryRewardCount
}

// ActiveTemporaryRewardExpiry returns the temporary reward expiry timestamp
// only when the temporary reward balance is still usable.
func (item EmailCatchAllAccess) ActiveTemporaryRewardExpiry(now time.Time) *time.Time {
	if item.ActiveTemporaryRewardCount(now) <= 0 {
		return nil
	}
	return item.TemporaryRewardExpiresAt
}

// NormalizeTemporaryReward removes expired temporary-reward state from one
// access snapshot so higher layers never accidentally treat stale rewards as
// usable balance.
func (item EmailCatchAllAccess) NormalizeTemporaryReward(now time.Time) EmailCatchAllAccess {
	if item.ActiveTemporaryRewardCount(now) > 0 {
		return item
	}
	item.TemporaryRewardCount = 0
	item.TemporaryRewardExpiresAt = nil
	return item
}

// TotalRemainingCount returns the currently usable catch-all count pool,
// combining permanent purchased balance with any still-active temporary reward.
func (item EmailCatchAllAccess) TotalRemainingCount(now time.Time) int64 {
	normalized := item.NormalizeTemporaryReward(now)
	return normalized.RemainingCount + normalized.TemporaryRewardCount
}
