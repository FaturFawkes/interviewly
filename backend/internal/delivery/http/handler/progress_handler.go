package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/delivery/http/middleware"
	"github.com/interview_app/backend/internal/domain"
)

// ProgressHandler handles analytics/progress APIs.
type ProgressHandler struct {
	interviewUC domain.InterviewUseCase
}

func NewProgressHandler(interviewUC domain.InterviewUseCase) *ProgressHandler {
	return &ProgressHandler{interviewUC: interviewUC}
}

// GetProgress handles GET /api/progress.
func (h *ProgressHandler) GetProgress(c *gin.Context) {
	userIDValue, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, ok := userIDValue.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return
	}

	progress, err := h.interviewUC.GetProgress(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, progress)
}
