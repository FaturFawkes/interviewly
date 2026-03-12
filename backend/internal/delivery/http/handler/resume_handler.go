package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/delivery/http/middleware"
	"github.com/interview_app/backend/internal/domain"
)

type saveResumeRequest struct {
	Content string `json:"content" binding:"required"`
}

// ResumeHandler handles resume-related APIs.
type ResumeHandler struct {
	interviewUC domain.InterviewUseCase
}

func NewResumeHandler(interviewUC domain.InterviewUseCase) *ResumeHandler {
	return &ResumeHandler{interviewUC: interviewUC}
}

// SaveResume handles POST /api/resume.
func (h *ResumeHandler) SaveResume(c *gin.Context) {
	var req saveResumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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

	resume, err := h.interviewUC.SaveResume(userID, req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resume)
}
