package router

import (
	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/delivery/http/handler"
)

// Setup registers all application routes and returns the configured Gin engine.
func Setup(
	healthHandler *handler.HealthHandler,
	authHandler *handler.AuthHandler,
	meHandler *handler.MeHandler,
	jobHandler *handler.JobHandler,
	resumeHandler *handler.ResumeHandler,
	questionHandler *handler.QuestionHandler,
	voiceHandler *handler.VoiceHandler,
	sessionHandler *handler.SessionHandler,
	feedbackHandler *handler.FeedbackHandler,
	progressHandler *handler.ProgressHandler,
	authMiddleware gin.HandlerFunc,
) *gin.Engine {
	r := gin.Default()

	// Health check endpoint
	r.GET("/health", healthHandler.GetHealth)
	r.POST("/auth/register", authHandler.Register)
	r.POST("/auth/register/resend", authHandler.ResendRegisterOTP)
	r.POST("/auth/register/verify", authHandler.VerifyRegisterOTP)
	r.POST("/auth/login", authHandler.Login)
	r.POST("/auth/social-login", authHandler.SocialLogin)

	api := r.Group("/api")
	api.Use(authMiddleware)
	api.GET("/me", meHandler.GetMe)
	api.POST("/job/parse", jobHandler.ParseJobDescription)
	api.POST("/resume", resumeHandler.SaveResume)
	api.POST("/questions/generate", questionHandler.GenerateQuestions)
	api.POST("/voice/tts", voiceHandler.TextToSpeech)
	api.POST("/voice/stt", voiceHandler.SpeechToText)
	api.POST("/session/start", sessionHandler.StartSession)
	api.POST("/session/answer", sessionHandler.SubmitAnswer)
	api.POST("/session/complete", sessionHandler.CompleteSession)
	api.GET("/session/history", sessionHandler.GetSessionHistory)
	api.POST("/feedback/generate", feedbackHandler.GenerateFeedback)
	api.GET("/progress", progressHandler.GetProgress)
	api.GET("/analytics/overview", progressHandler.GetAnalyticsOverview)

	return r
}
