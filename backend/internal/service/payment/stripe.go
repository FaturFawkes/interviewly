package payment

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/interview_app/backend/config"
)

type Plan struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AmountCents int64  `json:"amount_cents"`
	PriceID     string `json:"-"`
}

type CheckoutResult struct {
	CheckoutURL string `json:"checkout_url"`
	PlanID      string `json:"plan_id"`
	Currency    string `json:"currency"`
	AmountCents int64  `json:"amount_cents"`
}

type Service struct {
	secretKey  string
	successURL string
	cancelURL  string
	currency   string
	plans      map[string]Plan
	client     *http.Client
}

func NewService(cfg *config.Config) *Service {
	secretKey := ""
	successURL := "http://localhost:3000?payment=success"
	cancelURL := "http://localhost:3000?payment=cancel"
	currency := "usd"
	starterPriceID := ""
	proPriceID := ""
	elitePriceID := ""

	if cfg != nil {
		secretKey = strings.TrimSpace(cfg.StripeSecretKey)
		successURL = strings.TrimSpace(cfg.StripeSuccessURL)
		cancelURL = strings.TrimSpace(cfg.StripeCancelURL)
		currency = strings.ToLower(strings.TrimSpace(cfg.StripeCurrency))
		starterPriceID = strings.TrimSpace(cfg.StripePriceStarterMonthly)
		proPriceID = strings.TrimSpace(cfg.StripePriceProMonthly)
		elitePriceID = strings.TrimSpace(cfg.StripePriceEliteMonthly)
	}

	if currency == "" {
		currency = "usd"
	}

	plans := map[string]Plan{
		"starter": {
			ID:          "starter",
			Name:        "Starter",
			Description: "For focused solo prep",
			AmountCents: 1900,
			PriceID:     starterPriceID,
		},
		"pro": {
			ID:          "pro",
			Name:        "Pro Career Boost",
			Description: "For high-intent job seekers",
			AmountCents: 3900,
			PriceID:     proPriceID,
		},
		"elite": {
			ID:          "elite",
			Name:        "Elite",
			Description: "For accelerated interview mastery",
			AmountCents: 7900,
			PriceID:     elitePriceID,
		},
	}

	return &Service{
		secretKey:  secretKey,
		successURL: successURL,
		cancelURL:  cancelURL,
		currency:   currency,
		plans:      plans,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Service) IsReady() bool {
	return s != nil && strings.TrimSpace(s.secretKey) != ""
}

func (s *Service) PlanByID(planID string) (Plan, bool) {
	if s == nil {
		return Plan{}, false
	}

	plan, ok := s.plans[strings.ToLower(strings.TrimSpace(planID))]
	return plan, ok
}

func (s *Service) CreateCheckoutSession(planID string) (*CheckoutResult, error) {
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
	form.Set("metadata[plan_id]", plan.ID)

	if strings.TrimSpace(plan.PriceID) != "" {
		form.Set("line_items[0][price]", plan.PriceID)
	} else {
		form.Set("line_items[0][price_data][currency]", s.currency)
		form.Set("line_items[0][price_data][unit_amount]", fmt.Sprintf("%d", plan.AmountCents))
		form.Set("line_items[0][price_data][recurring][interval]", "month")
		form.Set("line_items[0][price_data][product_data][name]", plan.Name)
		form.Set("line_items[0][price_data][product_data][description]", plan.Description)
	}

	form.Set("line_items[0][quantity]", "1")

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

	var payload struct {
		URL string `json:"url"`
	}

	if err := json.Unmarshal(responseBody, &payload); err != nil {
		return nil, err
	}

	if strings.TrimSpace(payload.URL) == "" {
		return nil, fmt.Errorf("stripe checkout URL is empty")
	}

	return &CheckoutResult{
		CheckoutURL: payload.URL,
		PlanID:      plan.ID,
		Currency:    s.currency,
		AmountCents: plan.AmountCents,
	}, nil
}
