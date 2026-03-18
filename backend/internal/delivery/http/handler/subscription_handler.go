package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/delivery/http/middleware"
	"github.com/interview_app/backend/internal/service/subscription"
)

// SubscriptionHandler handles subscription and trial status APIs.
type SubscriptionHandler struct {
	subscriptionService *subscription.Service
}

func NewSubscriptionHandler(subscriptionService *subscription.Service) *SubscriptionHandler {
	return &SubscriptionHandler{subscriptionService: subscriptionService}
}

// GetStatus handles GET /api/subscription/status.
func (h *SubscriptionHandler) GetStatus(c *gin.Context) {
	if h.subscriptionService == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscription service is not configured"})
		return
	}

	userIDValue, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, ok := userIDValue.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return
	}

	status, err := h.subscriptionService.GetSubscriptionStatus(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}
