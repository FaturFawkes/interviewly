package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/delivery/http/middleware"
	"github.com/interview_app/backend/internal/domain"
)

type parseJobRequest struct {
	JobDescription string `json:"job_description" binding:"required"`
}

// JobHandler handles job-description related endpoints.
type JobHandler struct {
	interviewUC domain.InterviewUseCase
}

func NewJobHandler(interviewUC domain.InterviewUseCase) *JobHandler {
	return &JobHandler{interviewUC: interviewUC}
}

// ParseJobDescription handles POST /api/job/parse.
func (h *JobHandler) ParseJobDescription(c *gin.Context) {
	var req parseJobRequest
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

	parsed, err := h.interviewUC.ParseJobDescription(userID, req.JobDescription)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, parsed)
}
