package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/interview_app/backend/config"
	"github.com/interview_app/backend/internal/service/subscription"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	stripeProvider               = "stripe"
	checkoutTypeSubscription     = "subscription"
	checkoutTypeVoiceTopup       = "voice_topup"
	voiceTopupCode10             = "voice_topup_10"
	voiceTopupCode30             = "voice_topup_30"
	stripeWebhookTolerance       = 5 * time.Minute
	billingStatusProcessed       = "processed"
	billingStatusIgnored         = "ignored"
	billingStatusFailed          = "failed"
	voiceTopupOrderStatusPending = "pending"
	voiceTopupOrderStatusPaid    = "paid"
)

type Plan struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AmountCents int64  `json:"amount_cents"`
	PriceID     string `json:"-"`
}

type VoiceTopupPackage struct {
	Code        string `json:"package_code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	VoiceMins   int    `json:"voice_minutes"`
	AmountIDR   int    `json:"amount_idr"`
	PriceID     string `json:"-"`
}

type CheckoutResult struct {
	CheckoutURL       string `json:"checkout_url"`
	CheckoutSessionID string `json:"checkout_session_id,omitempty"`
	CheckoutType      string `json:"checkout_type"`
	PlanID            string `json:"plan_id,omitempty"`
	PackageCode       string `json:"package_code,omitempty"`
	VoiceMinutes      int    `json:"voice_minutes,omitempty"`
	Currency          string `json:"currency"`
	AmountCents       int64  `json:"amount_cents"`
}

type stripeCreateCheckoutSessionResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type stripeEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

type stripeCheckoutSession struct {
	ID            string            `json:"id"`
	Mode          string            `json:"mode"`
	PaymentStatus string            `json:"payment_status"`
	Metadata      map[string]string `json:"metadata"`
	PaymentIntent string            `json:"payment_intent"`
}

type Service struct {
	secretKey           string
	webhookSecret       string
	successURL          string
	cancelURL           string
	currency            string
	plans               map[string]Plan
	topupPackages       map[string]VoiceTopupPackage
	pool                *pgxpool.Pool
	subscriptionService *subscription.Service
	client              *http.Client
}

func NewService(cfg *config.Config, pool *pgxpool.Pool, subscriptionService *subscription.Service) *Service {
	secretKey := ""
	webhookSecret := ""
	successURL := "http://localhost:3000?payment=success"
	cancelURL := "http://localhost:3000?payment=cancel"
	currency := "idr"
	starterPriceID := ""
	proPriceID := ""
	elitePriceID := ""
	topup10PriceID := ""
	topup30PriceID := ""
	topup10AmountIDR := 19000
	topup30AmountIDR := 49000

	if cfg != nil {
		secretKey = strings.TrimSpace(cfg.StripeSecretKey)
		webhookSecret = strings.TrimSpace(cfg.StripeWebhookSecret)
		successURL = strings.TrimSpace(cfg.StripeSuccessURL)
		cancelURL = strings.TrimSpace(cfg.StripeCancelURL)
		currency = strings.ToLower(strings.TrimSpace(cfg.StripeCurrency))
		starterPriceID = strings.TrimSpace(cfg.StripePriceStarterMonthly)
		proPriceID = strings.TrimSpace(cfg.StripePriceProMonthly)
		elitePriceID = strings.TrimSpace(cfg.StripePriceEliteMonthly)
		topup10PriceID = strings.TrimSpace(cfg.StripePriceVoiceTopup10)
		topup30PriceID = strings.TrimSpace(cfg.StripePriceVoiceTopup30)
		if cfg.VoiceTopup10AmountIDR > 0 {
			topup10AmountIDR = cfg.VoiceTopup10AmountIDR
		}
		if cfg.VoiceTopup30AmountIDR > 0 {
			topup30AmountIDR = cfg.VoiceTopup30AmountIDR
		}
	}

	if currency == "" {
		currency = "idr"
	}

	plans := map[string]Plan{
		"starter": {
			ID:          "starter",
			Name:        "Starter",
			Description: "30 sessions/month + 30 voice minutes/month",
			AmountCents: 59000,
			PriceID:     starterPriceID,
		},
		"pro": {
			ID:          "pro",
			Name:        "Pro Career Boost",
			Description: "Unlimited text sessions + 120 voice minutes/month",
			AmountCents: 129000,
			PriceID:     proPriceID,
		},
		"elite": {
			ID:          "elite",
			Name:        "Elite",
			Description: "Unlimited text sessions + 300 voice minutes/month",
			AmountCents: 279000,
			PriceID:     elitePriceID,
		},
	}

	topupPackages := map[string]VoiceTopupPackage{
		voiceTopupCode10: {
			Code:        voiceTopupCode10,
			Name:        "Voice Top-Up 10 Menit",
			Description: "Tambah 10 menit voice interview",
			VoiceMins:   10,
			AmountIDR:   topup10AmountIDR,
			PriceID:     topup10PriceID,
		},
		voiceTopupCode30: {
			Code:        voiceTopupCode30,
			Name:        "Voice Top-Up 30 Menit",
			Description: "Tambah 30 menit voice interview",
			VoiceMins:   30,
			AmountIDR:   topup30AmountIDR,
			PriceID:     topup30PriceID,
		},
	}

	return &Service{
		secretKey:           secretKey,
		webhookSecret:       webhookSecret,
		successURL:          successURL,
		cancelURL:           cancelURL,
		currency:            currency,
		plans:               plans,
		topupPackages:       topupPackages,
		pool:                pool,
		subscriptionService: subscriptionService,
		client:              &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Service) IsReady() bool {
	return s != nil && strings.TrimSpace(s.secretKey) != ""
}

func (s *Service) IsWebhookReady() bool {
	return s.IsReady() && strings.TrimSpace(s.webhookSecret) != ""
}

func (s *Service) PlanByID(planID string) (Plan, bool) {
	if s == nil {
		return Plan{}, false
	}

	plan, ok := s.plans[strings.ToLower(strings.TrimSpace(planID))]
	return plan, ok
}

func (s *Service) VoiceTopupPackageByCode(code string) (VoiceTopupPackage, bool) {
	if s == nil {
		return VoiceTopupPackage{}, false
	}

	pkg, ok := s.topupPackages[strings.ToLower(strings.TrimSpace(code))]
	return pkg, ok
}

func (s *Service) CreateCheckoutSession(planID string) (*CheckoutResult, error) {
	return s.CreateSubscriptionCheckoutSession(planID, "")
}

func (s *Service) CreateSubscriptionCheckoutSession(planID, userID string) (*CheckoutResult, error) {
	if !s.IsReady() {
		return nil, fmt.Errorf("stripe payment service is not configured")
	}

	plan, ok := s.PlanByID(planID)
	if !ok {
		return nil, fmt.Errorf("invalid plan")
	}

	form := url.Values{}
	form.Set("mode", "subscription")
	form.Set("success_url", s.successURL)
	form.Set("cancel_url", s.cancelURL)
	form.Set("allow_promotion_codes", "true")
	form.Set("metadata[checkout_type]", checkoutTypeSubscription)
	form.Set("metadata[plan_id]", plan.ID)
	if trimmedUserID := strings.TrimSpace(userID); trimmedUserID != "" {
		form.Set("metadata[user_id]", trimmedUserID)
	}

	if isStripePriceID(plan.PriceID) {
		form.Set("line_items[0][price]", plan.PriceID)
	} else {
		form.Set("line_items[0][price_data][currency]", s.currency)
		form.Set("line_items[0][price_data][unit_amount]", fmt.Sprintf("%d", plan.AmountCents))
		form.Set("line_items[0][price_data][recurring][interval]", "month")
		form.Set("line_items[0][price_data][product_data][name]", plan.Name)
		form.Set("line_items[0][price_data][product_data][description]", plan.Description)
	}

	form.Set("line_items[0][quantity]", "1")

	payload, err := s.createStripeCheckout(form)
	if err != nil {
		return nil, err
	}

	return &CheckoutResult{
		CheckoutURL:       payload.URL,
		CheckoutSessionID: payload.ID,
		CheckoutType:      checkoutTypeSubscription,
		PlanID:            plan.ID,
		Currency:          s.currency,
		AmountCents:       plan.AmountCents,
	}, nil
}

func (s *Service) CreateVoiceTopupCheckoutSession(userID, packageCode string) (*CheckoutResult, error) {
	if !s.IsReady() {
		return nil, fmt.Errorf("stripe payment service is not configured")
	}
	if s.pool == nil {
		return nil, fmt.Errorf("database is not configured")
	}

	trimmedUserID := strings.TrimSpace(userID)
	if trimmedUserID == "" {
		return nil, fmt.Errorf("user id is required")
	}

	pkg, ok := s.VoiceTopupPackageByCode(packageCode)
	if !ok {
		return nil, fmt.Errorf("invalid voice top-up package")
	}

	orderID, err := s.createPendingTopupOrder(trimmedUserID, pkg)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("mode", "payment")
	form.Set("success_url", s.successURL)
	form.Set("cancel_url", s.cancelURL)
	form.Set("allow_promotion_codes", "true")
	form.Set("metadata[checkout_type]", checkoutTypeVoiceTopup)
	form.Set("metadata[user_id]", trimmedUserID)
	form.Set("metadata[package_code]", pkg.Code)
	form.Set("metadata[topup_order_id]", orderID)

	if isStripePriceID(pkg.PriceID) {
		form.Set("line_items[0][price]", pkg.PriceID)
	} else {
		form.Set("line_items[0][price_data][currency]", s.currency)
		form.Set("line_items[0][price_data][unit_amount]", fmt.Sprintf("%d", pkg.AmountIDR))
		form.Set("line_items[0][price_data][product_data][name]", pkg.Name)
		form.Set("line_items[0][price_data][product_data][description]", pkg.Description)
	}

	form.Set("line_items[0][quantity]", "1")

	payload, err := s.createStripeCheckout(form)
	if err != nil {
		_ = s.markTopupOrderFailed(orderID, err)
		return nil, err
	}

	if err := s.attachCheckoutSessionToTopupOrder(orderID, payload.ID); err != nil {
		return nil, err
	}

	return &CheckoutResult{
		CheckoutURL:       payload.URL,
		CheckoutSessionID: payload.ID,
		CheckoutType:      checkoutTypeVoiceTopup,
		PackageCode:       pkg.Code,
		VoiceMinutes:      pkg.VoiceMins,
		Currency:          s.currency,
		AmountCents:       int64(pkg.AmountIDR),
	}, nil
}

func (s *Service) HandleStripeWebhook(signatureHeader string, payload []byte) error {
	if !s.IsWebhookReady() {
		return fmt.Errorf("stripe webhook is not configured")
	}
	if s.pool == nil {
		return fmt.Errorf("database is not configured")
	}

	if !verifyStripeSignature(payload, signatureHeader, s.webhookSecret) {
		return fmt.Errorf("invalid stripe signature")
	}

	event := stripeEvent{}
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}
	if strings.TrimSpace(event.ID) == "" {
		return fmt.Errorf("stripe event id is empty")
	}

	shouldProcess, err := s.beginBillingEvent(event.ID, event.Type, payload)
	if err != nil {
		return err
	}
	if !shouldProcess {
		return nil
	}

	finalStatus := billingStatusProcessed
	finalErrorMessage := ""

	switch strings.TrimSpace(event.Type) {
	case "checkout.session.completed":
		err = s.handleCheckoutSessionCompleted(event)
	default:
		finalStatus = billingStatusIgnored
	}

	if err != nil {
		finalStatus = billingStatusFailed
		finalErrorMessage = err.Error()
	}

	if updateErr := s.finishBillingEvent(event.ID, finalStatus, finalErrorMessage); updateErr != nil {
		return updateErr
	}

	return err
}

func (s *Service) handleCheckoutSessionCompleted(event stripeEvent) error {
	checkoutSession := stripeCheckoutSession{}
	if err := json.Unmarshal(event.Data.Object, &checkoutSession); err != nil {
		return err
	}

	checkoutType := strings.ToLower(strings.TrimSpace(checkoutSession.Metadata["checkout_type"]))
	if checkoutType == "" {
		if strings.EqualFold(checkoutSession.Mode, checkoutTypeSubscription) {
			checkoutType = checkoutTypeSubscription
		}
		if checkoutSession.Metadata["package_code"] != "" {
			checkoutType = checkoutTypeVoiceTopup
		}
	}

	switch checkoutType {
	case checkoutTypeSubscription:
		if s.subscriptionService == nil {
			return errors.New("subscription service is not configured")
		}

		userID := strings.TrimSpace(checkoutSession.Metadata["user_id"])
		planID := strings.TrimSpace(checkoutSession.Metadata["plan_id"])
		if userID == "" || planID == "" {
			return fmt.Errorf("missing subscription metadata in checkout session")
		}

		_, err := s.subscriptionService.ApplyPaidPlan(userID, planID)
		return err

	case checkoutTypeVoiceTopup:
		if strings.ToLower(strings.TrimSpace(checkoutSession.PaymentStatus)) != "paid" {
			return nil
		}

		return s.creditTopupOrderFromCheckout(checkoutSession)

	default:
		return nil
	}
}

func (s *Service) creditTopupOrderFromCheckout(checkoutSession stripeCheckoutSession) error {
	topupOrderID := strings.TrimSpace(checkoutSession.Metadata["topup_order_id"])
	if topupOrderID == "" {
		return errors.New("missing top-up order id in checkout metadata")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var userID string
	var packageCode string
	var purchasedMinutes int
	var status string
	err = tx.QueryRow(
		ctx,
		`SELECT user_id, package_code, purchased_minutes, status
   FROM app_voice_topup_orders
  WHERE id = $1
  FOR UPDATE`,
		topupOrderID,
	).Scan(&userID, &packageCode, &purchasedMinutes, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("top-up order not found")
		}
		return err
	}

	if status == voiceTopupOrderStatusPaid {
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return nil
	}

	walletMetadata, _ := json.Marshal(map[string]interface{}{
		"provider":            stripeProvider,
		"topup_order_id":      topupOrderID,
		"checkout_session_id": strings.TrimSpace(checkoutSession.ID),
		"package_code":        packageCode,
	})

	_, err = tx.Exec(
		ctx,
		`INSERT INTO app_voice_wallet_entries
(id, user_id, source, purchased_minutes, consumed_minutes, metadata, created_at, updated_at)
 VALUES
($1, $2, $3, $4, 0, $5::jsonb, NOW(), NOW())`,
		uuid.NewString(),
		userID,
		"stripe_topup",
		purchasedMinutes,
		string(walletMetadata),
	)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	usageMetadata := fmt.Sprintf(`{"source":"stripe_topup","topup_order_id":%q}`, topupOrderID)
	_, _ = tx.Exec(
		ctx,
		`INSERT INTO app_usage_tracking
(id, user_id, session_id, usage_type, consumed_minutes, consumed_sessions, period_start, period_end, metadata)
 VALUES
($1, $2, NULL, 'voice_topup', $3, 0, $4, $5, $6::jsonb)`,
		uuid.NewString(),
		userID,
		purchasedMinutes,
		now,
		now.AddDate(0, 1, 0),
		usageMetadata,
	)

	orderMetadata, _ := json.Marshal(map[string]interface{}{
		"credited_at":         now,
		"checkout_session_id": strings.TrimSpace(checkoutSession.ID),
	})

	_, err = tx.Exec(
		ctx,
		`UPDATE app_voice_topup_orders
SET status = $2,
provider_checkout_session_id = COALESCE(NULLIF($3, ''), provider_checkout_session_id),
provider_payment_intent_id = COALESCE(NULLIF($4, ''), provider_payment_intent_id),
metadata = COALESCE(metadata, '{}'::jsonb) || $5::jsonb,
updated_at = NOW()
  WHERE id = $1`,
		topupOrderID,
		voiceTopupOrderStatusPaid,
		strings.TrimSpace(checkoutSession.ID),
		strings.TrimSpace(checkoutSession.PaymentIntent),
		string(orderMetadata),
	)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Service) createStripeCheckout(form url.Values) (*stripeCreateCheckoutSessionResponse, error) {
	request, err := http.NewRequest(http.MethodPost, "https://api.stripe.com/v1/checkout/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Authorization", "Bearer "+s.secretKey)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("stripe checkout error: %s", strings.TrimSpace(string(responseBody)))
	}

	payload := stripeCreateCheckoutSessionResponse{}
	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return nil, err
	}

	if strings.TrimSpace(payload.URL) == "" {
		return nil, fmt.Errorf("stripe checkout URL is empty")
	}
	if strings.TrimSpace(payload.ID) == "" {
		return nil, fmt.Errorf("stripe checkout session id is empty")
	}

	return &payload, nil
}

func (s *Service) createPendingTopupOrder(userID string, pkg VoiceTopupPackage) (string, error) {
	if s.pool == nil {
		return "", errors.New("database is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	orderID := uuid.NewString()
	metadata, _ := json.Marshal(map[string]interface{}{
		"source": "checkout",
	})

	_, err := s.pool.Exec(
		ctx,
		`INSERT INTO app_voice_topup_orders
(id, user_id, provider, package_code, purchased_minutes, amount_idr, status, metadata, created_at, updated_at)
 VALUES
($1, $2, $3, $4, $5, $6, $7, $8::jsonb, NOW(), NOW())`,
		orderID,
		userID,
		stripeProvider,
		pkg.Code,
		pkg.VoiceMins,
		pkg.AmountIDR,
		voiceTopupOrderStatusPending,
		string(metadata),
	)
	if err != nil {
		return "", err
	}

	return orderID, nil
}

func (s *Service) attachCheckoutSessionToTopupOrder(orderID, checkoutSessionID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.pool.Exec(
		ctx,
		`UPDATE app_voice_topup_orders
SET provider_checkout_session_id = $2,
updated_at = NOW()
  WHERE id = $1`,
		orderID,
		checkoutSessionID,
	)
	return err
}

func (s *Service) markTopupOrderFailed(orderID string, checkoutErr error) error {
	if s.pool == nil || strings.TrimSpace(orderID) == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	metadata, _ := json.Marshal(map[string]interface{}{
		"checkout_error": strings.TrimSpace(checkoutErr.Error()),
	})

	_, err := s.pool.Exec(
		ctx,
		`UPDATE app_voice_topup_orders
SET status = 'failed',
metadata = COALESCE(metadata, '{}'::jsonb) || $2::jsonb,
updated_at = NOW()
  WHERE id = $1`,
		orderID,
		string(metadata),
	)
	return err
}

func (s *Service) beginBillingEvent(eventID, eventType string, payload []byte) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	insertTag, err := s.pool.Exec(
		ctx,
		`INSERT INTO app_billing_events
(id, provider, event_id, event_type, payload, processing_status, error_message, processed_at, created_at)
 VALUES
($1, $2, $3, $4, $5::jsonb, $6, $7, NOW(), NOW())
 ON CONFLICT (provider, event_id)
 DO NOTHING`,
		uuid.NewString(),
		stripeProvider,
		eventID,
		eventType,
		string(payload),
		billingStatusFailed,
		"processing",
	)
	if err != nil {
		return false, err
	}

	if insertTag.RowsAffected() > 0 {
		return true, nil
	}

	var currentStatus string
	err = s.pool.QueryRow(
		ctx,
		`SELECT processing_status
   FROM app_billing_events
  WHERE provider = $1
    AND event_id = $2`,
		stripeProvider,
		eventID,
	).Scan(&currentStatus)
	if err != nil {
		return false, err
	}

	if currentStatus == billingStatusProcessed || currentStatus == billingStatusIgnored {
		return false, nil
	}

	_, err = s.pool.Exec(
		ctx,
		`UPDATE app_billing_events
SET event_type = $3,
payload = $4::jsonb,
error_message = NULL,
processed_at = NOW()
  WHERE provider = $1
    AND event_id = $2`,
		stripeProvider,
		eventID,
		eventType,
		string(payload),
	)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *Service) finishBillingEvent(eventID, status, errorMessage string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	var errMessageValue interface{}
	trimmed := strings.TrimSpace(errorMessage)
	if trimmed != "" {
		errMessageValue = trimmed
	}

	_, err := s.pool.Exec(
		ctx,
		`UPDATE app_billing_events
SET processing_status = $3,
error_message = $4,
processed_at = NOW()
  WHERE provider = $1
    AND event_id = $2`,
		stripeProvider,
		eventID,
		status,
		errMessageValue,
	)

	return err
}

func verifyStripeSignature(payload []byte, signatureHeader, secret string) bool {
	trimmedHeader := strings.TrimSpace(signatureHeader)
	trimmedSecret := strings.TrimSpace(secret)
	if trimmedHeader == "" || trimmedSecret == "" || len(payload) == 0 {
		return false
	}

	parts := strings.Split(trimmedHeader, ",")
	timestamp := ""
	v1Signatures := make([]string, 0, 2)

	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "t":
			timestamp = value
		case "v1":
			if value != "" {
				v1Signatures = append(v1Signatures, value)
			}
		}
	}

	if timestamp == "" || len(v1Signatures) == 0 {
		return false
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}

	now := time.Now().UTC().Unix()
	if now-ts > int64(stripeWebhookTolerance.Seconds()) || ts-now > int64(stripeWebhookTolerance.Seconds()) {
		return false
	}

	signedPayload := fmt.Sprintf("%s.%s", timestamp, payload)
	h := hmac.New(sha256.New, []byte(trimmedSecret))
	_, _ = h.Write([]byte(signedPayload))
	expected := hex.EncodeToString(h.Sum(nil))

	for _, provided := range v1Signatures {
		if hmac.Equal([]byte(expected), []byte(strings.ToLower(provided))) {
			return true
		}
	}

	return false
}

func isStripePriceID(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}

	return strings.HasPrefix(trimmed, "price_")
}
