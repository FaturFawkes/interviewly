package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/delivery/http/middleware"
	"github.com/interview_app/backend/internal/domain"
)

type startSessionRequest struct {
	ResumeID            string   `json:"resume_id" binding:"required"`
	JobParseID          string   `json:"job_parse_id" binding:"required"`
	QuestionIDs         []string `json:"question_ids" binding:"required"`
	InterviewMode       string   `json:"interview_mode"`
	InterviewLanguage   string   `json:"interview_language"`
	InterviewDifficulty string   `json:"interview_difficulty"`
	TargetRole          string   `json:"target_role"`
	TargetCompany       string   `json:"target_company"`
}

type submitAnswerRequest struct {
	SessionID  string `json:"session_id" binding:"required"`
	QuestionID string `json:"question_id" binding:"required"`
	Answer     string `json:"answer" binding:"required"`
}

type completeSessionRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// SessionHandler handles interview session APIs.
type SessionHandler struct {
	interviewUC domain.InterviewUseCase
}

func NewSessionHandler(interviewUC domain.InterviewUseCase) *SessionHandler {
	return &SessionHandler{interviewUC: interviewUC}
}

// StartSession handles POST /api/session/start.
func (h *SessionHandler) StartSession(c *gin.Context) {
	var req startSessionRequest
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

	session, err := h.interviewUC.CreatePracticeSession(userID, req.ResumeID, req.JobParseID, req.QuestionIDs, domain.SessionMetadata{
		InterviewMode:       req.InterviewMode,
		InterviewLanguage:   domain.NormalizeInterviewLanguage(req.InterviewLanguage),
		InterviewDifficulty: domain.NormalizeInterviewDifficulty(req.InterviewDifficulty),
		TargetRole:          req.TargetRole,
		TargetCompany:       req.TargetCompany,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

// SubmitAnswer handles POST /api/session/answer.
func (h *SessionHandler) SubmitAnswer(c *gin.Context) {
	var req submitAnswerRequest
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

	result, err := h.interviewUC.SubmitSessionAnswer(userID, req.SessionID, req.QuestionID, req.Answer)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CompleteSession handles POST /api/session/complete.
func (h *SessionHandler) CompleteSession(c *gin.Context) {
	var req completeSessionRequest
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

	session, err := h.interviewUC.CompletePracticeSession(userID, req.SessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

// GetSessionHistory handles GET /api/session/history.
func (h *SessionHandler) GetSessionHistory(c *gin.Context) {
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

	sessions, err := h.interviewUC.ListPracticeSessions(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}
