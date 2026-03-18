package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/service/payment"
)

type paymentCheckoutRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
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

	result, err := h.service.CreateCheckoutSession(req.PlanID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
