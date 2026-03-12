package router

import (
	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/delivery/http/handler"
)

// Setup registers all application routes and returns the configured Gin engine.
func Setup(
	healthHandler *handler.HealthHandler,
	meHandler *handler.MeHandler,
	jobHandler *handler.JobHandler,
	resumeHandler *handler.ResumeHandler,
	questionHandler *handler.QuestionHandler,
	sessionHandler *handler.SessionHandler,
	feedbackHandler *handler.FeedbackHandler,
	authMiddleware gin.HandlerFunc,
) *gin.Engine {
	r := gin.Default()

	// Health check endpoint
	r.GET("/health", healthHandler.GetHealth)

	api := r.Group("/api")
	api.Use(authMiddleware)
	api.GET("/me", meHandler.GetMe)
	api.POST("/job/parse", jobHandler.ParseJobDescription)
	api.POST("/resume", resumeHandler.SaveResume)
	api.POST("/questions/generate", questionHandler.GenerateQuestions)
	api.POST("/session/start", sessionHandler.StartSession)
	api.POST("/session/answer", sessionHandler.SubmitAnswer)
	api.GET("/session/history", sessionHandler.GetSessionHistory)
	api.POST("/feedback/generate", feedbackHandler.GenerateFeedback)

	return r
}
