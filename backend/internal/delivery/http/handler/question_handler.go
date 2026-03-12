package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/delivery/http/middleware"
	"github.com/interview_app/backend/internal/domain"
)

type generateQuestionsRequest struct {
	ResumeText     string `json:"resume_text" binding:"required"`
	JobDescription string `json:"job_description" binding:"required"`
}

// QuestionHandler handles question generation APIs.
type QuestionHandler struct {
	interviewUC domain.InterviewUseCase
}

func NewQuestionHandler(interviewUC domain.InterviewUseCase) *QuestionHandler {
	return &QuestionHandler{interviewUC: interviewUC}
}

// GenerateQuestions handles POST /api/questions/generate.
func (h *QuestionHandler) GenerateQuestions(c *gin.Context) {
	var req generateQuestionsRequest
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

	questions, err := h.interviewUC.GenerateQuestions(userID, req.ResumeText, req.JobDescription)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"questions": questions,
	}

	if len(questions) > 0 {
		response["resume_id"] = questions[0].ResumeID
		response["job_parse_id"] = questions[0].JobParseID
	}

	c.JSON(http.StatusOK, response)
}
