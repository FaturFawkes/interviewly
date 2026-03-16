package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/delivery/http/middleware"
	"github.com/interview_app/backend/internal/domain"
)

type generateFeedbackRequest struct {
	SessionID  string `json:"session_id" binding:"required"`
	QuestionID string `json:"question_id" binding:"required"`
	Question   string `json:"question" binding:"required"`
	Answer     string `json:"answer" binding:"required"`
}

type submitAgentFeedbackRequest struct {
	SessionID    string   `json:"session_id" binding:"required"`
	QuestionID   string   `json:"question_id" binding:"required"`
	Question     string   `json:"question" binding:"required"`
	Answer       string   `json:"answer" binding:"required"`
	Score        int      `json:"score" binding:"required"`
	Strengths    []string `json:"strengths"`
	Weaknesses   []string `json:"weaknesses"`
	Improvements []string `json:"improvements"`
	STARFeedback string   `json:"star_feedback"`
}

// FeedbackHandler handles feedback APIs.
type FeedbackHandler struct {
	interviewUC domain.InterviewUseCase
}

func NewFeedbackHandler(interviewUC domain.InterviewUseCase) *FeedbackHandler {
	return &FeedbackHandler{interviewUC: interviewUC}
}

// GenerateFeedback handles POST /api/feedback/generate.
func (h *FeedbackHandler) GenerateFeedback(c *gin.Context) {
	var req generateFeedbackRequest
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

	feedback, err := h.interviewUC.GenerateFeedback(
		userID,
		req.SessionID,
		req.QuestionID,
		req.Question,
		req.Answer,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, feedback)
}

// SubmitAgentFeedback handles POST /api/feedback/agent.
func (h *FeedbackHandler) SubmitAgentFeedback(c *gin.Context) {
	var req submitAgentFeedbackRequest
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

	analysis := &domain.AnswerAnalysis{
		Score:        req.Score,
		Strengths:    req.Strengths,
		Weaknesses:   req.Weaknesses,
		Improvements: req.Improvements,
		STARFeedback: req.STARFeedback,
	}

	feedback, err := h.interviewUC.SubmitAgentFeedback(
		userID,
		req.SessionID,
		req.QuestionID,
		req.Question,
		req.Answer,
		analysis,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, feedback)
}
