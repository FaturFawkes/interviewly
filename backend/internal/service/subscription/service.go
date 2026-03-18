package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/interview_app/backend/config"
	"github.com/interview_app/backend/internal/infrastructure/cache"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	planFree    = "free"
	planStarter = "starter"
	planPro     = "pro"
	planElite   = "elite"

	softTrialDurationHours     = 48
	softTrialBonusVoiceMinutes = 30
	softTrialTriggerSessions   = 2
)

type PlanQuota struct {
	PlanID            string
	TotalVoiceMinutes int
	TotalSessions     int
}

type SubscriptionState struct {
	ID                string     `json:"id"`
	UserID            string     `json:"user_id"`
	PlanID            string     `json:"plan_id"`
	Status            string     `json:"status"`
	TotalVoiceMinutes int        `json:"total_voice_minutes"`
	UsedVoiceMinutes  int        `json:"used_voice_minutes"`
	TotalSessions     int        `json:"total_sessions"`
	UsedSessions      int        `json:"used_sessions"`
	PeriodStart       time.Time  `json:"period_start"`
	PeriodEnd         time.Time  `json:"period_end"`
	TrialStartedAt    *time.Time `json:"trial_started_at,omitempty"`
	TrialEndsAt       *time.Time `json:"trial_ends_at,omitempty"`
	TrialUsedAt       *time.Time `json:"trial_used_at,omitempty"`
	TrialPlanID       string     `json:"trial_plan_id,omitempty"`
	TrialVoiceBonus   int        `json:"trial_voice_bonus_minutes"`
	TrialVoiceUsed    int        `json:"trial_consumed_voice_minutes"`
}

type SubscriptionStatus struct {
	PlanID                  string     `json:"plan_id"`
	IsFreeTier              bool       `json:"is_free_tier"`
	TotalVoiceMinutes       int        `json:"total_voice_minutes"`
	UsedVoiceMinutes        int        `json:"used_voice_minutes"`
	RemainingVoiceMinutes   int        `json:"remaining_voice_minutes"`
	TotalSessions           int        `json:"total_sessions"`
	UsedSessions            int        `json:"used_sessions"`
	RemainingSessions       int        `json:"remaining_sessions"`
	TrialAvailable          bool       `json:"trial_available"`
	TrialActive             bool       `json:"trial_active"`
	TrialDurationHours      int        `json:"trial_duration_hours"`
	TrialBonusVoiceMinutes  int        `json:"trial_bonus_voice_minutes"`
	TrialEndsAt             *time.Time `json:"trial_ends_at,omitempty"`
	TriggerRequiredSessions int        `json:"trigger_required_sessions"`
	TriggerProgressSessions int        `json:"trigger_progress_sessions"`
	UpsellMessages          []string   `json:"upsell_messages"`
	AntiAbuseRules          []string   `json:"anti_abuse_rules"`
}

type VoiceQuotaCheck struct {
	CanStart                bool   `json:"can_start"`
	Message                 string `json:"message,omitempty"`
	TotalVoiceMinutes       int    `json:"total_voice_minutes"`
	UsedVoiceMinutes        int    `json:"used_voice_minutes"`
	RemainingVoiceMinutes   int    `json:"remaining_voice_minutes"`
	AllowedCallSeconds      int    `json:"allowed_call_seconds"`
	WarningThresholdReached bool   `json:"warning_threshold_reached"`
}

type Service struct {
	pool  *pgxpool.Pool
	cache *cache.RedisCache

	mu            sync.Mutex
	memorySubs    map[string]*SubscriptionState
	planCatalog   map[string]PlanQuota
	stateCacheTTL time.Duration
}

func NewService(cfg *config.Config, pool *pgxpool.Pool, redisCache *cache.RedisCache) *Service {
	_ = cfg
	return &Service{
		pool:       pool,
		cache:      redisCache,
		memorySubs: map[string]*SubscriptionState{},
		planCatalog: map[string]PlanQuota{
			planFree: {
				PlanID:            planFree,
				TotalVoiceMinutes: 10,
				TotalSessions:     5,
			},
			planStarter: {
				PlanID:            planStarter,
				TotalVoiceMinutes: 30,
				TotalSessions:     30,
			},
			planPro: {
				PlanID:            planPro,
				TotalVoiceMinutes: 120,
				TotalSessions:     -1,
			},
			planElite: {
				PlanID:            planElite,
				TotalVoiceMinutes: 300,
				TotalSessions:     -1,
			},
		},
		stateCacheTTL: 60 * time.Second,
	}
}

func (s *Service) PlanQuota(planID string) PlanQuota {
	plan, ok := s.planCatalog[normalizePlanID(planID)]
	if !ok {
		return s.planCatalog[planStarter]
	}
	return plan
}

func (s *Service) EnsureActiveSubscription(userID string) (*SubscriptionState, error) {
	trimmed := strings.TrimSpace(userID)
	if trimmed == "" {
		return nil, errors.New("user id is required")
	}

	if cached, ok := s.getCachedState(trimmed); ok {
		return cached, nil
	}

	if s.pool == nil {
		s.mu.Lock()
		defer s.mu.Unlock()

		state, exists := s.memorySubs[trimmed]
		now := time.Now().UTC()
		if !exists || !now.Before(state.PeriodEnd) {
			fresh := s.newState(trimmed, planFree, now)
			s.memorySubs[trimmed] = fresh
			s.cacheState(trimmed, fresh)
			return fresh, nil
		}

		s.maybeActivateSoftTrialInMemory(state, now)

		s.cacheState(trimmed, state)
		return cloneState(state), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	state, err := s.getCurrentStateFromDB(ctx, trimmed, now)
	if err != nil {
		return nil, err
	}

	if state != nil {
		if activatedState, activateErr := s.tryActivateSoftTrialDB(ctx, state, now); activateErr == nil {
			state = activatedState
		}
		s.cacheState(trimmed, state)
		return state, nil
	}

	fresh := s.newState(trimmed, planFree, now)
	if err := s.insertStateDB(ctx, fresh); err != nil {
		return nil, err
	}

	s.cacheState(trimmed, fresh)
	return fresh, nil
}

func (s *Service) CanStartSession(userID string) (*SubscriptionState, error) {
	state, err := s.EnsureActiveSubscription(userID)
	if err != nil {
		return nil, err
	}

	if s.sessionLimitReached(state, time.Now().UTC()) {
		return nil, errors.New("monthly interview session limit reached for current plan")
	}

	return state, nil
}

func (s *Service) ConsumeSession(userID, sessionID string) (*SubscriptionState, error) {
	trimmedUserID := strings.TrimSpace(userID)
	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedUserID == "" {
		return nil, errors.New("user id is required")
	}
	if trimmedSessionID == "" {
		return nil, errors.New("session id is required")
	}

	state, err := s.EnsureActiveSubscription(trimmedUserID)
	if err != nil {
		return nil, err
	}

	if s.sessionLimitReached(state, time.Now().UTC()) {
		return nil, errors.New("monthly interview session limit reached for current plan")
	}

	if s.pool == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		current := s.memorySubs[trimmedUserID]
		if current == nil {
			return nil, errors.New("subscription not found")
		}
		if current.TotalSessions >= 0 {
			current.UsedSessions++
		}
		s.cacheState(trimmedUserID, current)
		return cloneState(current), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var inserted bool
	err = tx.QueryRow(
		ctx,
		`INSERT INTO app_usage_tracking
			(id, user_id, session_id, usage_type, consumed_minutes, consumed_sessions, period_start, period_end, metadata)
		 VALUES
			($1, $2, $3::uuid, 'session_count', 0, 1, $4, $5, $6::jsonb)
		 ON CONFLICT (user_id, session_id, usage_type, period_start) WHERE session_id IS NOT NULL
		 DO NOTHING
		 RETURNING true`,
		uuid.NewString(),
		trimmedUserID,
		trimmedSessionID,
		state.PeriodStart,
		state.PeriodEnd,
		`{"source":"session_start"}`,
	).Scan(&inserted)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if inserted {
		_, err = tx.Exec(
			ctx,
			`UPDATE app_subscriptions
				SET used_sessions = CASE WHEN total_sessions_limit < 0 THEN used_sessions ELSE used_sessions + 1 END,
					updated_at = NOW()
			 WHERE id = $1`,
			state.ID,
		)
		if err != nil {
			return nil, err
		}
	}

	updated, err := s.getCurrentStateFromTx(ctx, tx, trimmedUserID, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	s.cacheState(trimmedUserID, updated)
	return updated, nil
}

func (s *Service) CheckVoiceQuota(userID string) (*VoiceQuotaCheck, error) {
	state, err := s.EnsureActiveSubscription(userID)
	if err != nil {
		return nil, err
	}

	totalVoiceMinutes, usedVoiceMinutes, remainingMinutes := s.effectiveVoiceUsage(state, time.Now().UTC())

	warningThresholdReached := false
	if totalVoiceMinutes > 0 {
		remainingPercent := (remainingMinutes * 100) / totalVoiceMinutes
		warningThresholdReached = remainingPercent > 0 && remainingPercent <= 10
	}

	result := &VoiceQuotaCheck{
		CanStart:                remainingMinutes > 0,
		TotalVoiceMinutes:       totalVoiceMinutes,
		UsedVoiceMinutes:        usedVoiceMinutes,
		RemainingVoiceMinutes:   remainingMinutes,
		AllowedCallSeconds:      remainingMinutes * 60,
		WarningThresholdReached: warningThresholdReached,
	}

	if !result.CanStart {
		result.Message = "voice quota exceeded for current monthly plan"
	}

	if warningThresholdReached {
		result.Message = "voice quota is below 10%"
	}

	return result, nil
}

func (s *Service) CommitVoiceUsage(userID, sessionID string, elapsedSeconds int) (*SubscriptionState, error) {
	trimmedUserID := strings.TrimSpace(userID)
	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedUserID == "" {
		return nil, errors.New("user id is required")
	}
	if trimmedSessionID == "" {
		return nil, errors.New("session id is required")
	}
	if elapsedSeconds <= 0 {
		return s.EnsureActiveSubscription(trimmedUserID)
	}

	consumedMinutes := int((elapsedSeconds + 59) / 60)
	state, err := s.EnsureActiveSubscription(trimmedUserID)
	if err != nil {
		return nil, err
	}

	if s.pool == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		current := s.memorySubs[trimmedUserID]
		if current == nil {
			return nil, errors.New("subscription not found")
		}
		total, used, remaining := s.effectiveVoiceUsage(current, time.Now().UTC())
		_ = total
		_ = used
		if consumedMinutes > remaining {
			consumedMinutes = remaining
		}
		consumeBase, consumeTrial := s.allocateVoiceConsumption(current, consumedMinutes, time.Now().UTC())
		current.UsedVoiceMinutes += consumeBase
		current.TrialVoiceUsed += consumeTrial
		s.cacheState(trimmedUserID, current)
		return cloneState(current), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var inserted bool
	err = tx.QueryRow(
		ctx,
		`INSERT INTO app_usage_tracking
			(id, user_id, session_id, usage_type, consumed_minutes, consumed_sessions, period_start, period_end, metadata)
		 VALUES
			($1, $2, $3::uuid, 'voice_minutes', $4, 0, $5, $6, $7::jsonb)
		 ON CONFLICT (user_id, session_id, usage_type, period_start) WHERE session_id IS NOT NULL
		 DO NOTHING
		 RETURNING true`,
		uuid.NewString(),
		trimmedUserID,
		trimmedSessionID,
		consumedMinutes,
		state.PeriodStart,
		state.PeriodEnd,
		fmt.Sprintf(`{"elapsed_seconds":%d}`, elapsedSeconds),
	).Scan(&inserted)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if inserted {
		consumeBase, consumeTrial := s.allocateVoiceConsumption(state, consumedMinutes, time.Now().UTC())
		_, err = tx.Exec(
			ctx,
			`UPDATE app_subscriptions
				SET used_voice_minutes = LEAST(total_voice_minutes, used_voice_minutes + $2),
					trial_consumed_voice_minutes = LEAST(trial_voice_bonus_minutes, trial_consumed_voice_minutes + $3),
					updated_at = NOW()
			 WHERE id = $1`,
			state.ID,
			consumeBase,
			consumeTrial,
		)
		if err != nil {
			return nil, err
		}
	}

	updated, err := s.getCurrentStateFromTx(ctx, tx, trimmedUserID, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	s.cacheState(trimmedUserID, updated)
	return updated, nil
}

func (s *Service) newState(userID, planID string, now time.Time) *SubscriptionState {
	quota := s.PlanQuota(planID)

	var periodStart time.Time
	var periodEnd time.Time
	if quota.PlanID == planFree {
		periodStart = time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
		periodEnd = periodStart.AddDate(0, 0, 7)
	} else {
		periodStart = time.Date(now.UTC().Year(), now.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
		periodEnd = periodStart.AddDate(0, 1, 0)
	}

	return &SubscriptionState{
		ID:                uuid.NewString(),
		UserID:            userID,
		PlanID:            quota.PlanID,
		Status:            "active",
		TotalVoiceMinutes: quota.TotalVoiceMinutes,
		UsedVoiceMinutes:  0,
		TotalSessions:     quota.TotalSessions,
		UsedSessions:      0,
		PeriodStart:       periodStart,
		PeriodEnd:         periodEnd,
		TrialVoiceBonus:   0,
		TrialVoiceUsed:    0,
	}
}

func (s *Service) getCurrentStateFromDB(ctx context.Context, userID string, now time.Time) (*SubscriptionState, error) {
	row := s.pool.QueryRow(
		ctx,
		`SELECT id, user_id, plan_id, status, total_voice_minutes, used_voice_minutes, total_sessions_limit, used_sessions, period_start, period_end,
				trial_started_at, trial_ends_at, trial_used_at, COALESCE(trial_plan_id, ''), trial_voice_bonus_minutes, trial_consumed_voice_minutes
		   FROM app_subscriptions
		  WHERE user_id = $1
		    AND status = 'active'
		    AND period_end > $2
		  ORDER BY period_end DESC
		  LIMIT 1`,
		userID,
		now,
	)

	state := &SubscriptionState{}
	err := row.Scan(
		&state.ID,
		&state.UserID,
		&state.PlanID,
		&state.Status,
		&state.TotalVoiceMinutes,
		&state.UsedVoiceMinutes,
		&state.TotalSessions,
		&state.UsedSessions,
		&state.PeriodStart,
		&state.PeriodEnd,
		&state.TrialStartedAt,
		&state.TrialEndsAt,
		&state.TrialUsedAt,
		&state.TrialPlanID,
		&state.TrialVoiceBonus,
		&state.TrialVoiceUsed,
	)
	if err == nil {
		return state, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return nil, err
}

func (s *Service) getCurrentStateFromTx(ctx context.Context, tx pgx.Tx, userID string, now time.Time) (*SubscriptionState, error) {
	row := tx.QueryRow(
		ctx,
		`SELECT id, user_id, plan_id, status, total_voice_minutes, used_voice_minutes, total_sessions_limit, used_sessions, period_start, period_end,
				trial_started_at, trial_ends_at, trial_used_at, COALESCE(trial_plan_id, ''), trial_voice_bonus_minutes, trial_consumed_voice_minutes
		   FROM app_subscriptions
		  WHERE user_id = $1
		    AND status = 'active'
		    AND period_end > $2
		  ORDER BY period_end DESC
		  LIMIT 1`,
		userID,
		now,
	)

	state := &SubscriptionState{}
	err := row.Scan(
		&state.ID,
		&state.UserID,
		&state.PlanID,
		&state.Status,
		&state.TotalVoiceMinutes,
		&state.UsedVoiceMinutes,
		&state.TotalSessions,
		&state.UsedSessions,
		&state.PeriodStart,
		&state.PeriodEnd,
		&state.TrialStartedAt,
		&state.TrialEndsAt,
		&state.TrialUsedAt,
		&state.TrialPlanID,
		&state.TrialVoiceBonus,
		&state.TrialVoiceUsed,
	)
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (s *Service) insertStateDB(ctx context.Context, state *SubscriptionState) error {
	var trialPlanID interface{}
	trimmedTrialPlanID := strings.TrimSpace(state.TrialPlanID)
	if trimmedTrialPlanID != "" {
		normalizedTrialPlanID := normalizePlanID(trimmedTrialPlanID)
		if normalizedTrialPlanID == planPro || normalizedTrialPlanID == planElite {
			trialPlanID = normalizedTrialPlanID
		}
	}

	_, err := s.pool.Exec(
		ctx,
		`INSERT INTO app_subscriptions
			(id, user_id, plan_id, status, total_voice_minutes, used_voice_minutes, total_sessions_limit, used_sessions, period_start, period_end,
			 trial_started_at, trial_ends_at, trial_used_at, trial_plan_id, trial_voice_bonus_minutes, trial_consumed_voice_minutes,
			 created_at, updated_at)
		 VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NOW(), NOW())`,
		state.ID,
		state.UserID,
		state.PlanID,
		state.Status,
		state.TotalVoiceMinutes,
		state.UsedVoiceMinutes,
		state.TotalSessions,
		state.UsedSessions,
		state.PeriodStart,
		state.PeriodEnd,
		state.TrialStartedAt,
		state.TrialEndsAt,
		state.TrialUsedAt,
		trialPlanID,
		state.TrialVoiceBonus,
		state.TrialVoiceUsed,
	)
	return err
}

func (s *Service) GetSubscriptionStatus(userID string) (*SubscriptionStatus, error) {
	state, err := s.EnsureActiveSubscription(userID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	totalVoice, usedVoice, remainingVoice := s.effectiveVoiceUsage(state, now)
	totalSessions := s.effectiveSessionLimit(state, now)
	remainingSessions := -1
	if totalSessions >= 0 {
		remainingSessions = totalSessions - state.UsedSessions
		if remainingSessions < 0 {
			remainingSessions = 0
		}
	}

	trialActive := s.isTrialActive(state, now)
	trialAvailable := normalizePlanID(state.PlanID) == planFree && state.TrialUsedAt == nil && !trialActive

	status := &SubscriptionStatus{
		PlanID:                  state.PlanID,
		IsFreeTier:              normalizePlanID(state.PlanID) == planFree,
		TotalVoiceMinutes:       totalVoice,
		UsedVoiceMinutes:        usedVoice,
		RemainingVoiceMinutes:   remainingVoice,
		TotalSessions:           totalSessions,
		UsedSessions:            state.UsedSessions,
		RemainingSessions:       remainingSessions,
		TrialAvailable:          trialAvailable,
		TrialActive:             trialActive,
		TrialDurationHours:      softTrialDurationHours,
		TrialBonusVoiceMinutes:  softTrialBonusVoiceMinutes,
		TrialEndsAt:             state.TrialEndsAt,
		TriggerRequiredSessions: softTrialTriggerSessions,
		TriggerProgressSessions: minInt(state.UsedSessions, softTrialTriggerSessions),
		UpsellMessages: []string{
			"Upgrade untuk simulasi interview real (voice)",
			"Sisa voice kamu tinggal sedikit",
			"Naikkan skor interview kamu dengan Pro",
		},
		AntiAbuseRules: []string{
			"1 trial per user",
			"Login wajib (Google/email)",
			"Limit device/IP",
		},
	}

	return status, nil
}

func (s *Service) getCachedState(userID string) (*SubscriptionState, bool) {
	if s.cache == nil {
		return nil, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()

	payload, err := s.cache.Get(ctx, s.cacheKey(userID))
	if err != nil || strings.TrimSpace(payload) == "" {
		return nil, false
	}

	var state SubscriptionState
	if err := json.Unmarshal([]byte(payload), &state); err != nil {
		return nil, false
	}

	if time.Now().UTC().After(state.PeriodEnd) {
		return nil, false
	}

	return &state, true
}

func (s *Service) cacheState(userID string, state *SubscriptionState) {
	if s.cache == nil || state == nil {
		return
	}

	payload, err := json.Marshal(state)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()

	_ = s.cache.Set(ctx, s.cacheKey(userID), payload, s.stateCacheTTL)
}

func (s *Service) cacheKey(userID string) string {
	return "subscription:state:" + userID
}

func normalizePlanID(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case planFree:
		return planFree
	case planPro:
		return planPro
	case planElite:
		return planElite
	case planStarter:
		return planStarter
	default:
		return planFree
	}
}

func (s *Service) maybeActivateSoftTrialInMemory(state *SubscriptionState, now time.Time) {
	if state == nil || !s.shouldActivateSoftTrial(state, now) {
		return
	}

	trialStart := now.UTC()
	trialEnd := trialStart.Add(softTrialDurationHours * time.Hour)
	state.TrialStartedAt = &trialStart
	state.TrialEndsAt = &trialEnd
	state.TrialUsedAt = &trialStart
	state.TrialPlanID = planPro
	state.TrialVoiceBonus = softTrialBonusVoiceMinutes
	state.TrialVoiceUsed = 0
}

func (s *Service) tryActivateSoftTrialDB(ctx context.Context, state *SubscriptionState, now time.Time) (*SubscriptionState, error) {
	if state == nil || !s.shouldActivateSoftTrial(state, now) {
		return state, nil
	}

	trialStart := now.UTC()
	trialEnd := trialStart.Add(softTrialDurationHours * time.Hour)

	_, err := s.pool.Exec(
		ctx,
		`UPDATE app_subscriptions
			SET trial_started_at = $2,
				trial_ends_at = $3,
				trial_used_at = COALESCE(trial_used_at, $2),
				trial_plan_id = $4,
				trial_voice_bonus_minutes = $5,
				trial_consumed_voice_minutes = 0,
				updated_at = NOW()
		  WHERE id = $1
			AND trial_used_at IS NULL`,
		state.ID,
		trialStart,
		trialEnd,
		planPro,
		softTrialBonusVoiceMinutes,
	)
	if err != nil {
		return nil, err
	}

	updated, err := s.getCurrentStateFromDB(ctx, state.UserID, now)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return state, nil
	}

	return updated, nil
}

func (s *Service) shouldActivateSoftTrial(state *SubscriptionState, now time.Time) bool {
	if state == nil {
		return false
	}
	if normalizePlanID(state.PlanID) != planFree {
		return false
	}
	if state.TrialUsedAt != nil {
		return false
	}
	if s.isTrialActive(state, now) {
		return false
	}
	return state.UsedSessions >= softTrialTriggerSessions
}

func (s *Service) isTrialActive(state *SubscriptionState, now time.Time) bool {
	if state == nil || state.TrialEndsAt == nil {
		return false
	}
	return now.Before(*state.TrialEndsAt)
}

func (s *Service) effectiveSessionLimit(state *SubscriptionState, now time.Time) int {
	if s.isTrialActive(state, now) {
		trialPlan := state.TrialPlanID
		if strings.TrimSpace(trialPlan) == "" {
			trialPlan = planPro
		}
		return s.PlanQuota(trialPlan).TotalSessions
	}
	return state.TotalSessions
}

func (s *Service) sessionLimitReached(state *SubscriptionState, now time.Time) bool {
	limit := s.effectiveSessionLimit(state, now)
	if limit < 0 {
		return false
	}
	return state.UsedSessions >= limit
}

func (s *Service) effectiveVoiceUsage(state *SubscriptionState, now time.Time) (int, int, int) {
	total := state.TotalVoiceMinutes
	used := state.UsedVoiceMinutes

	if s.isTrialActive(state, now) {
		total += state.TrialVoiceBonus
		used += state.TrialVoiceUsed
	}

	remaining := total - used
	if remaining < 0 {
		remaining = 0
	}

	return total, used, remaining
}

func (s *Service) allocateVoiceConsumption(state *SubscriptionState, consumedMinutes int, now time.Time) (int, int) {
	if consumedMinutes <= 0 {
		return 0, 0
	}

	remainingBase := state.TotalVoiceMinutes - state.UsedVoiceMinutes
	if remainingBase < 0 {
		remainingBase = 0
	}
	consumeBase := minInt(consumedMinutes, remainingBase)
	remainingAfterBase := consumedMinutes - consumeBase

	consumeTrial := 0
	if remainingAfterBase > 0 && s.isTrialActive(state, now) {
		remainingTrial := state.TrialVoiceBonus - state.TrialVoiceUsed
		if remainingTrial < 0 {
			remainingTrial = 0
		}
		consumeTrial = minInt(remainingAfterBase, remainingTrial)
	}

	return consumeBase, consumeTrial
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func cloneState(state *SubscriptionState) *SubscriptionState {
	if state == nil {
		return nil
	}
	copyValue := *state
	return &copyValue
}
