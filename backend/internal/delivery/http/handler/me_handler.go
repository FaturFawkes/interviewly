package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/delivery/http/middleware"
)

// MeHandler handles authenticated user profile endpoints.
type MeHandler struct{}

// NewMeHandler creates a new MeHandler.
func NewMeHandler() *MeHandler {
	return &MeHandler{}
}

// GetMe returns currently authenticated user information.
func (h *MeHandler) GetMe(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id": userID,
	})
}
