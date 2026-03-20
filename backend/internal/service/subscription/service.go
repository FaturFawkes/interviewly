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

	defaultWarningThresholdPercent = 10
	defaultFUPDelaySeconds         = 5
)

type PlanQuota struct {
	PlanID            string
	TotalVoiceMinutes int
	TotalSessions     int
	TotalTextRequests int
	TotalJDParses     int
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
	TotalTextRequests int        `json:"total_text_requests"`
	UsedTextRequests  int        `json:"used_text_requests"`
	TotalJDParses     int        `json:"total_jd_limit"`
	UsedJDParses      int        `json:"used_jd_parses"`
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
	TotalTextRequests       int        `json:"total_text_requests"`
	UsedTextRequests        int        `json:"used_text_requests"`
	RemainingTextRequests   int        `json:"remaining_text_requests"`
	TextFUPExceeded         bool       `json:"text_fup_exceeded"`
	ShouldSlowdownResponse  bool       `json:"should_slowdown_response"`
	SuggestedDowngradeModel string     `json:"suggested_downgrade_model,omitempty"`
	TotalJDLimit            int        `json:"total_jd_limit"`
	UsedJDParses            int        `json:"used_jd_parses"`
	RemainingJDParses       int        `json:"remaining_jd_parses"`
	TotalVoiceTopupMinutes  int        `json:"total_voice_topup_minutes"`
	UsedVoiceTopupMinutes   int        `json:"used_voice_topup_minutes"`
	RemainingVoiceTopup     int        `json:"remaining_voice_topup_minutes"`
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

type TextUsageDecision struct {
	TotalTextRequests       int    `json:"total_text_requests"`
	UsedTextRequests        int    `json:"used_text_requests"`
	RemainingTextRequests   int    `json:"remaining_text_requests"`
	FUPExceeded             bool   `json:"fup_exceeded"`
	WarningThresholdReached bool   `json:"warning_threshold_reached"`
	ShouldDelayResponse     bool   `json:"should_delay_response"`
	SuggestedModel          string `json:"suggested_model,omitempty"`
	UpgradeMessage          string `json:"upgrade_message,omitempty"`
}

type JDUsageDecision struct {
	CanParse          bool   `json:"can_parse"`
	TotalJDLimit      int    `json:"total_jd_limit"`
	UsedJDParses      int    `json:"used_jd_parses"`
	RemainingJDParses int    `json:"remaining_jd_parses"`
	Message           string `json:"message,omitempty"`
}

type Service struct {
	pool  *pgxpool.Pool
	cache *cache.RedisCache

	mu             sync.Mutex
	memorySubs     map[string]*SubscriptionState
	planCatalog    map[string]PlanQuota
	stateCacheTTL  time.Duration
	warningPercent int
	fupDelay       time.Duration
	fupModel       string
}

func NewService(cfg *config.Config, pool *pgxpool.Pool, redisCache *cache.RedisCache) *Service {
	warningPercent := defaultWarningThresholdPercent
	fupDelaySeconds := defaultFUPDelaySeconds
	fupModel := ""

	if cfg != nil {
		if cfg.SubscriptionWarningThresholdPercent > 0 {
			warningPercent = cfg.SubscriptionWarningThresholdPercent
		}
		if cfg.SubscriptionFUPDelaySeconds > 0 {
			fupDelaySeconds = cfg.SubscriptionFUPDelaySeconds
		}
		fupModel = strings.TrimSpace(cfg.AIModelFUPDowngrade)
	}

	return &Service{
		pool:       pool,
		cache:      redisCache,
		memorySubs: map[string]*SubscriptionState{},
		planCatalog: map[string]PlanQuota{
			planFree: {
				PlanID:            planFree,
				TotalVoiceMinutes: 10,
				TotalSessions:     5,
				TotalTextRequests: 600,
				TotalJDParses:     1,
			},
			planStarter: {
				PlanID:            planStarter,
				TotalVoiceMinutes: 30,
				TotalSessions:     -1,
				TotalTextRequests: 1200,
				TotalJDParses:     5,
			},
			planPro: {
				PlanID:            planPro,
				TotalVoiceMinutes: 120,
				TotalSessions:     -1,
				TotalTextRequests: 4000,
				TotalJDParses:     30,
			},
			planElite: {
				PlanID:            planElite,
				TotalVoiceMinutes: 300,
				TotalSessions:     -1,
				TotalTextRequests: 10000,
				TotalJDParses:     50,
			},
		},
		stateCacheTTL:  60 * time.Second,
		warningPercent: warningPercent,
		fupDelay:       time.Duration(fupDelaySeconds) * time.Second,
		fupModel:       fupModel,
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

// ApplyPaidPlan activates a paid plan for the current billing period and resets period counters.
func (s *Service) ApplyPaidPlan(userID, planID string) (*SubscriptionState, error) {
	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return nil, errors.New("user id is required")
	}

	normalizedPlanID := normalizePlanID(planID)
	if normalizedPlanID == planFree {
		return nil, errors.New("paid plan id is required")
	}

	now := time.Now().UTC()
	state := s.newState(trimmedUserID, normalizedPlanID, now)
	state.Status = "active"

	if s.pool == nil {
		s.mu.Lock()
		defer s.mu.Unlock()

		s.memorySubs[trimmedUserID] = state
		s.cacheState(trimmedUserID, state)
		return cloneState(state), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(
		ctx,
		`UPDATE app_subscriptions
			SET status = 'canceled',
				updated_at = NOW()
		  WHERE user_id = $1
			AND status = 'active'`,
		trimmedUserID,
	)
	if err != nil {
		return nil, err
	}

	updatedState, err := s.upsertPaidStateFromTx(ctx, tx, state)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	s.cacheState(trimmedUserID, updatedState)
	return updatedState, nil
}

func (s *Service) ConsumeTextRequest(userID, source string) (*TextUsageDecision, error) {
	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return nil, errors.New("user id is required")
	}

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
		current.UsedTextRequests++
		s.cacheState(trimmedUserID, current)
		return s.buildTextUsageDecision(current, time.Now().UTC()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(
		ctx,
		`UPDATE app_subscriptions
			SET used_text_requests = used_text_requests + 1,
				updated_at = NOW()
		 WHERE id = $1`,
		state.ID,
	)
	if err != nil {
		return nil, err
	}

	_, _ = tx.Exec(
		ctx,
		`INSERT INTO app_usage_tracking
			(id, user_id, session_id, usage_type, consumed_minutes, consumed_sessions, period_start, period_end, metadata)
		 VALUES
			($1, $2, NULL, 'text_request', 0, 1, $3, $4, $5::jsonb)`,
		uuid.NewString(),
		trimmedUserID,
		state.PeriodStart,
		state.PeriodEnd,
		fmt.Sprintf(`{"source":%q}`, strings.TrimSpace(source)),
	)

	updated, err := s.getCurrentStateFromTx(ctx, tx, trimmedUserID, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	s.cacheState(trimmedUserID, updated)
	return s.buildTextUsageDecision(updated, time.Now().UTC()), nil
}

func (s *Service) ConsumeJDParse(userID, source string) (*JDUsageDecision, error) {
	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return nil, errors.New("user id is required")
	}

	state, err := s.EnsureActiveSubscription(trimmedUserID)
	if err != nil {
		return nil, err
	}

	if s.jdLimitReached(state, time.Now().UTC()) {
		decision := s.buildJDUsageDecision(state, time.Now().UTC())
		decision.CanParse = false
		decision.Message = "job description parsing limit reached for current plan"
		return decision, errors.New(decision.Message)
	}

	if s.pool == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		current := s.memorySubs[trimmedUserID]
		if current == nil {
			return nil, errors.New("subscription not found")
		}
		if s.jdLimitReached(current, time.Now().UTC()) {
			decision := s.buildJDUsageDecision(current, time.Now().UTC())
			decision.CanParse = false
			decision.Message = "job description parsing limit reached for current plan"
			return decision, errors.New(decision.Message)
		}
		current.UsedJDParses++
		s.cacheState(trimmedUserID, current)
		return s.buildJDUsageDecision(current, time.Now().UTC()), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	effectiveLimit := s.effectiveJDLimit(state, time.Now().UTC())

	tag, err := tx.Exec(
		ctx,
		`UPDATE app_subscriptions
			SET used_jd_parses = used_jd_parses + 1,
				updated_at = NOW()
		 WHERE id = $1
		   AND ($2 < 0 OR used_jd_parses < $2)`,
		state.ID,
		effectiveLimit,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		decision := s.buildJDUsageDecision(state, time.Now().UTC())
		decision.CanParse = false
		decision.Message = "job description parsing limit reached for current plan"
		return decision, errors.New(decision.Message)
	}

	_, _ = tx.Exec(
		ctx,
		`INSERT INTO app_usage_tracking
			(id, user_id, session_id, usage_type, consumed_minutes, consumed_sessions, period_start, period_end, metadata)
		 VALUES
			($1, $2, NULL, 'jd_parse', 0, 1, $3, $4, $5::jsonb)`,
		uuid.NewString(),
		trimmedUserID,
		state.PeriodStart,
		state.PeriodEnd,
		fmt.Sprintf(`{"source":%q}`, strings.TrimSpace(source)),
	)

	updated, err := s.getCurrentStateFromTx(ctx, tx, trimmedUserID, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	s.cacheState(trimmedUserID, updated)
	return s.buildJDUsageDecision(updated, time.Now().UTC()), nil
}

func (s *Service) CheckVoiceQuota(userID string) (*VoiceQuotaCheck, error) {
	state, err := s.EnsureActiveSubscription(userID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	totalVoiceMinutes, usedVoiceMinutes, remainingMinutes := s.effectiveVoiceUsage(state, now)
	walletTotal, walletUsed, walletRemaining, walletErr := s.voiceWalletUsage(strings.TrimSpace(userID))
	if walletErr != nil {
		return nil, walletErr
	}

	totalVoiceMinutes += walletTotal
	usedVoiceMinutes += walletUsed
	remainingMinutes += walletRemaining

	warningThresholdReached := false
	if totalVoiceMinutes > 0 {
		remainingPercent := (remainingMinutes * 100) / totalVoiceMinutes
		warningThresholdReached = remainingPercent > 0 && remainingPercent <= s.warningPercent
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
		result.Message = "voice quota exceeded for current plan"
	}

	if warningThresholdReached {
		result.Message = fmt.Sprintf("voice quota is below %d%%", s.warningPercent)
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

	walletRemaining, err := s.voiceWalletRemainingFromTx(ctx, tx, trimmedUserID)
	if err != nil {
		return nil, err
	}

	_, _, remainingPlanAndTrial := s.effectiveVoiceUsage(state, time.Now().UTC())
	availableTotal := remainingPlanAndTrial + walletRemaining
	if consumedMinutes > availableTotal {
		consumedMinutes = availableTotal
	}
	if consumedMinutes <= 0 {
		updated, currentErr := s.getCurrentStateFromTx(ctx, tx, trimmedUserID, time.Now().UTC())
		if currentErr != nil {
			return nil, currentErr
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		s.cacheState(trimmedUserID, updated)
		return updated, nil
	}

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
		remainingAfterPlan := consumedMinutes - consumeBase - consumeTrial
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

		if remainingAfterPlan > 0 {
			if _, consumeErr := s.consumeVoiceWalletFromTx(ctx, tx, trimmedUserID, remainingAfterPlan); consumeErr != nil {
				return nil, consumeErr
			}
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
		TotalTextRequests: quota.TotalTextRequests,
		UsedTextRequests:  0,
		TotalJDParses:     quota.TotalJDParses,
		UsedJDParses:      0,
		PeriodStart:       periodStart,
		PeriodEnd:         periodEnd,
		TrialVoiceBonus:   0,
		TrialVoiceUsed:    0,
	}
}

func (s *Service) getCurrentStateFromDB(ctx context.Context, userID string, now time.Time) (*SubscriptionState, error) {
	row := s.pool.QueryRow(
		ctx,
		`SELECT id, user_id, plan_id, status, total_voice_minutes, used_voice_minutes, total_sessions_limit, used_sessions,
				COALESCE(total_text_requests, 0), COALESCE(used_text_requests, 0), COALESCE(total_jd_limit, 0), COALESCE(used_jd_parses, 0), period_start, period_end,
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
		&state.TotalTextRequests,
		&state.UsedTextRequests,
		&state.TotalJDParses,
		&state.UsedJDParses,
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
		`SELECT id, user_id, plan_id, status, total_voice_minutes, used_voice_minutes, total_sessions_limit, used_sessions,
				COALESCE(total_text_requests, 0), COALESCE(used_text_requests, 0), COALESCE(total_jd_limit, 0), COALESCE(used_jd_parses, 0), period_start, period_end,
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
		&state.TotalTextRequests,
		&state.UsedTextRequests,
		&state.TotalJDParses,
		&state.UsedJDParses,
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
			 total_text_requests, used_text_requests, total_jd_limit, used_jd_parses,
			 trial_started_at, trial_ends_at, trial_used_at, trial_plan_id, trial_voice_bonus_minutes, trial_consumed_voice_minutes,
			 created_at, updated_at)
		 VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, NOW(), NOW())`,
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
		state.TotalTextRequests,
		state.UsedTextRequests,
		state.TotalJDParses,
		state.UsedJDParses,
		state.TrialStartedAt,
		state.TrialEndsAt,
		state.TrialUsedAt,
		trialPlanID,
		state.TrialVoiceBonus,
		state.TrialVoiceUsed,
	)
	return err
}

func (s *Service) upsertPaidStateFromTx(ctx context.Context, tx pgx.Tx, state *SubscriptionState) (*SubscriptionState, error) {
	if state == nil {
		return nil, errors.New("subscription state is required")
	}

	row := tx.QueryRow(
		ctx,
		`INSERT INTO app_subscriptions
			(id, user_id, plan_id, status, total_voice_minutes, used_voice_minutes, total_sessions_limit, used_sessions,
			 period_start, period_end, total_text_requests, used_text_requests, total_jd_limit, used_jd_parses,
			 trial_started_at, trial_ends_at, trial_used_at, trial_plan_id, trial_voice_bonus_minutes, trial_consumed_voice_minutes,
			 created_at, updated_at)
		 VALUES
			($1, $2, $3, 'active', $4, 0, $5, 0, $6, $7, $8, 0, $9, 0,
			 NULL, NULL, NULL, NULL, 0, 0, NOW(), NOW())
		 ON CONFLICT (user_id, period_start, period_end)
		 DO UPDATE
			SET plan_id = EXCLUDED.plan_id,
				status = 'active',
				total_voice_minutes = EXCLUDED.total_voice_minutes,
				used_voice_minutes = 0,
				total_sessions_limit = EXCLUDED.total_sessions_limit,
				used_sessions = 0,
				total_text_requests = EXCLUDED.total_text_requests,
				used_text_requests = 0,
				total_jd_limit = EXCLUDED.total_jd_limit,
				used_jd_parses = 0,
				trial_started_at = NULL,
				trial_ends_at = NULL,
				trial_plan_id = NULL,
				trial_voice_bonus_minutes = 0,
				trial_consumed_voice_minutes = 0,
				updated_at = NOW()
		 RETURNING id, user_id, plan_id, status, total_voice_minutes, used_voice_minutes, total_sessions_limit, used_sessions,
				   total_text_requests, used_text_requests, total_jd_limit, used_jd_parses, period_start, period_end,
				   trial_started_at, trial_ends_at, trial_used_at, COALESCE(trial_plan_id, ''), trial_voice_bonus_minutes, trial_consumed_voice_minutes`,
		state.ID,
		state.UserID,
		state.PlanID,
		state.TotalVoiceMinutes,
		state.TotalSessions,
		state.PeriodStart,
		state.PeriodEnd,
		state.TotalTextRequests,
		state.TotalJDParses,
	)

	updated := &SubscriptionState{}
	err := row.Scan(
		&updated.ID,
		&updated.UserID,
		&updated.PlanID,
		&updated.Status,
		&updated.TotalVoiceMinutes,
		&updated.UsedVoiceMinutes,
		&updated.TotalSessions,
		&updated.UsedSessions,
		&updated.TotalTextRequests,
		&updated.UsedTextRequests,
		&updated.TotalJDParses,
		&updated.UsedJDParses,
		&updated.PeriodStart,
		&updated.PeriodEnd,
		&updated.TrialStartedAt,
		&updated.TrialEndsAt,
		&updated.TrialUsedAt,
		&updated.TrialPlanID,
		&updated.TrialVoiceBonus,
		&updated.TrialVoiceUsed,
	)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) GetSubscriptionStatus(userID string) (*SubscriptionStatus, error) {
	state, err := s.EnsureActiveSubscription(userID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	totalVoice, usedVoice, remainingVoice := s.effectiveVoiceUsage(state, now)
	walletTotal, walletUsed, walletRemaining, walletErr := s.voiceWalletUsage(strings.TrimSpace(userID))
	if walletErr != nil {
		return nil, walletErr
	}
	totalVoice += walletTotal
	usedVoice += walletUsed
	remainingVoice += walletRemaining

	totalSessions := s.effectiveSessionLimit(state, now)
	remainingSessions := -1
	if totalSessions >= 0 {
		remainingSessions = totalSessions - state.UsedSessions
		if remainingSessions < 0 {
			remainingSessions = 0
		}
	}

	totalTextRequests := s.effectiveTextLimit(state, now)
	remainingTextRequests := totalTextRequests - state.UsedTextRequests
	if remainingTextRequests < 0 {
		remainingTextRequests = 0
	}
	textFUPExceeded := totalTextRequests >= 0 && state.UsedTextRequests > totalTextRequests

	totalJDLimit := s.effectiveJDLimit(state, now)
	remainingJDParses := -1
	if totalJDLimit >= 0 {
		remainingJDParses = totalJDLimit - state.UsedJDParses
		if remainingJDParses < 0 {
			remainingJDParses = 0
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
		TotalTextRequests:       totalTextRequests,
		UsedTextRequests:        state.UsedTextRequests,
		RemainingTextRequests:   remainingTextRequests,
		TextFUPExceeded:         textFUPExceeded,
		ShouldSlowdownResponse:  textFUPExceeded,
		SuggestedDowngradeModel: s.fupModel,
		TotalJDLimit:            totalJDLimit,
		UsedJDParses:            state.UsedJDParses,
		RemainingJDParses:       remainingJDParses,
		TotalVoiceTopupMinutes:  walletTotal,
		UsedVoiceTopupMinutes:   walletUsed,
		RemainingVoiceTopup:     walletRemaining,
		TrialAvailable:          trialAvailable,
		TrialActive:             trialActive,
		TrialDurationHours:      softTrialDurationHours,
		TrialBonusVoiceMinutes:  softTrialBonusVoiceMinutes,
		TrialEndsAt:             state.TrialEndsAt,
		TriggerRequiredSessions: softTrialTriggerSessions,
		TriggerProgressSessions: minInt(state.UsedSessions, softTrialTriggerSessions),
		UpsellMessages: []string{
			"Upgrade plan untuk kuota voice dan request AI yang lebih besar",
			"Beli voice top-up ketika menit voice bulanan habis",
			"Aktifkan Pro/Elite untuk deep feedback dan prioritas response",
		},
		AntiAbuseRules: []string{
			"1 trial per user",
			"Login wajib (Google/email)",
			"Rate limit 10 request per menit",
		},
	}

	if !textFUPExceeded {
		status.SuggestedDowngradeModel = ""
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

func (s *Service) effectiveTextLimit(state *SubscriptionState, now time.Time) int {
	if s.isTrialActive(state, now) {
		trialPlan := state.TrialPlanID
		if strings.TrimSpace(trialPlan) == "" {
			trialPlan = planPro
		}
		return s.PlanQuota(trialPlan).TotalTextRequests
	}
	return state.TotalTextRequests
}

func (s *Service) effectiveJDLimit(state *SubscriptionState, now time.Time) int {
	if s.isTrialActive(state, now) {
		trialPlan := state.TrialPlanID
		if strings.TrimSpace(trialPlan) == "" {
			trialPlan = planPro
		}
		return s.PlanQuota(trialPlan).TotalJDParses
	}
	return state.TotalJDParses
}

func (s *Service) sessionLimitReached(state *SubscriptionState, now time.Time) bool {
	limit := s.effectiveSessionLimit(state, now)
	if limit < 0 {
		return false
	}
	return state.UsedSessions >= limit
}

func (s *Service) jdLimitReached(state *SubscriptionState, now time.Time) bool {
	limit := s.effectiveJDLimit(state, now)
	if limit < 0 {
		return false
	}
	return state.UsedJDParses >= limit
}

func (s *Service) buildTextUsageDecision(state *SubscriptionState, now time.Time) *TextUsageDecision {
	totalTextRequests := s.effectiveTextLimit(state, now)
	remaining := totalTextRequests - state.UsedTextRequests
	if remaining < 0 {
		remaining = 0
	}

	fupExceeded := totalTextRequests >= 0 && state.UsedTextRequests > totalTextRequests

	warningThresholdReached := false
	if totalTextRequests > 0 {
		remainingPercent := (remaining * 100) / totalTextRequests
		warningThresholdReached = remainingPercent > 0 && remainingPercent <= s.warningPercent
	}

	decision := &TextUsageDecision{
		TotalTextRequests:       totalTextRequests,
		UsedTextRequests:        state.UsedTextRequests,
		RemainingTextRequests:   remaining,
		FUPExceeded:             fupExceeded,
		WarningThresholdReached: warningThresholdReached,
		ShouldDelayResponse:     fupExceeded,
		SuggestedModel:          "",
		UpgradeMessage:          "",
	}

	if fupExceeded {
		decision.SuggestedModel = s.fupModel
		decision.UpgradeMessage = "Text fair usage exceeded. Upgrade plan for normal speed and higher AI quota."
	}

	if warningThresholdReached && decision.UpgradeMessage == "" {
		decision.UpgradeMessage = "Text fair usage almost exhausted for this period."
	}

	return decision
}

func (s *Service) buildJDUsageDecision(state *SubscriptionState, now time.Time) *JDUsageDecision {
	totalJDLimit := s.effectiveJDLimit(state, now)
	remaining := -1
	if totalJDLimit >= 0 {
		remaining = totalJDLimit - state.UsedJDParses
		if remaining < 0 {
			remaining = 0
		}
	}

	return &JDUsageDecision{
		CanParse:          totalJDLimit < 0 || state.UsedJDParses < totalJDLimit,
		TotalJDLimit:      totalJDLimit,
		UsedJDParses:      state.UsedJDParses,
		RemainingJDParses: remaining,
	}
}

func (s *Service) TextFUPDelay() time.Duration {
	if s.fupDelay <= 0 {
		return defaultFUPDelaySeconds * time.Second
	}
	return s.fupDelay
}

func (s *Service) TextFUPDowngradeModel() string {
	return strings.TrimSpace(s.fupModel)
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

func (s *Service) voiceWalletUsage(userID string) (int, int, int, error) {
	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return 0, 0, 0, nil
	}
	if s.pool == nil {
		return 0, 0, 0, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var total int
	var used int
	err := s.pool.QueryRow(
		ctx,
		`SELECT COALESCE(SUM(purchased_minutes), 0), COALESCE(SUM(consumed_minutes), 0)
		   FROM app_voice_wallet_entries
		  WHERE user_id = $1`,
		trimmedUserID,
	).Scan(&total, &used)
	if err != nil {
		return 0, 0, 0, err
	}

	remaining := total - used
	if remaining < 0 {
		remaining = 0
	}

	return total, used, remaining, nil
}

func (s *Service) voiceWalletRemainingFromTx(ctx context.Context, tx pgx.Tx, userID string) (int, error) {
	var remaining int
	err := tx.QueryRow(
		ctx,
		`SELECT COALESCE(SUM(purchased_minutes - consumed_minutes), 0)
		   FROM app_voice_wallet_entries
		  WHERE user_id = $1`,
		userID,
	).Scan(&remaining)
	if err != nil {
		return 0, err
	}
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

func (s *Service) consumeVoiceWalletFromTx(ctx context.Context, tx pgx.Tx, userID string, neededMinutes int) (int, error) {
	if neededMinutes <= 0 {
		return 0, nil
	}

	rows, err := tx.Query(
		ctx,
		`SELECT id, purchased_minutes, consumed_minutes
		   FROM app_voice_wallet_entries
		  WHERE user_id = $1
		    AND consumed_minutes < purchased_minutes
		  ORDER BY created_at ASC, id ASC
		  FOR UPDATE`,
		userID,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	consumed := 0
	remainingNeeded := neededMinutes

	for rows.Next() {
		if remainingNeeded <= 0 {
			break
		}

		var entryID string
		var purchasedMinutes int
		var consumedMinutes int
		if scanErr := rows.Scan(&entryID, &purchasedMinutes, &consumedMinutes); scanErr != nil {
			return 0, scanErr
		}

		entryRemaining := purchasedMinutes - consumedMinutes
		if entryRemaining <= 0 {
			continue
		}

		consumeNow := minInt(remainingNeeded, entryRemaining)
		_, updateErr := tx.Exec(
			ctx,
			`UPDATE app_voice_wallet_entries
				SET consumed_minutes = consumed_minutes + $2,
					updated_at = NOW()
			 WHERE id = $1`,
			entryID,
			consumeNow,
		)
		if updateErr != nil {
			return 0, updateErr
		}

		consumed += consumeNow
		remainingNeeded -= consumeNow
	}

	if err := rows.Err(); err != nil {
		return 0, err
	}

	return consumed, nil
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
