package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/delivery/http/middleware"
	"github.com/interview_app/backend/internal/service/payment"
)

type paymentCheckoutRequest struct {
	PlanID       string `json:"plan_id"`
	CheckoutType string `json:"checkout_type"`
	PackageCode  string `json:"package_code"`
}

type PaymentHandler struct {
	service *payment.Service
}

func NewPaymentHandler(service *payment.Service) *PaymentHandler {
	return &PaymentHandler{service: service}
}

func (h *PaymentHandler) CreateCheckoutSession(c *gin.Context) {
	if h.service == nil || !h.service.IsReady() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment service is not configured"})
		return
	}

	var req paymentCheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := ""
	if userIDValue, exists := c.Get(middleware.UserIDContextKey); exists {
		if value, ok := userIDValue.(string); ok {
			userID = strings.TrimSpace(value)
		}
	}

	checkoutType := strings.ToLower(strings.TrimSpace(req.CheckoutType))
	if checkoutType == "" {
		if strings.TrimSpace(req.PackageCode) != "" {
			checkoutType = "voice_topup"
		} else {
			checkoutType = "subscription"
		}
	}

	var (
		result *payment.CheckoutResult
		err    error
	)

	switch checkoutType {
	case "voice_topup":
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required for voice top-up checkout"})
			return
		}

		result, err = h.service.CreateVoiceTopupCheckoutSession(userID, req.PackageCode)
	default:
		if strings.TrimSpace(req.PlanID) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "plan_id is required"})
			return
		}

		result, err = h.service.CreateSubscriptionCheckoutSession(req.PlanID, userID)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PaymentHandler) HandleStripeWebhook(c *gin.Context) {
	if h.service == nil || !h.service.IsWebhookReady() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stripe webhook is not configured"})
		return
	}

	signatureHeader := strings.TrimSpace(c.GetHeader("Stripe-Signature"))
	if signatureHeader == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing Stripe-Signature header"})
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read webhook body"})
		return
	}

	if err := h.service.HandleStripeWebhook(signatureHeader, body); err != nil {
		statusCode := http.StatusBadRequest
		errMessage := strings.ToLower(strings.TrimSpace(err.Error()))
		if !strings.Contains(errMessage, "signature") {
			statusCode = http.StatusInternalServerError
		}

		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "received"})
}
