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
	paymentHandler *handler.PaymentHandler,
	sessionHandler *handler.SessionHandler,
	subscriptionHandler *handler.SubscriptionHandler,
	feedbackHandler *handler.FeedbackHandler,
	progressHandler *handler.ProgressHandler,
	authMiddleware gin.HandlerFunc,
	rateLimitMiddleware gin.HandlerFunc,
) *gin.Engine {
	r := gin.Default()

	// Health check endpoint
	r.GET("/health", healthHandler.GetHealth)
	r.POST("/auth/register", authHandler.Register)
	r.POST("/auth/register/resend", authHandler.ResendRegisterOTP)
	r.POST("/auth/register/verify", authHandler.VerifyRegisterOTP)
	r.POST("/auth/login", authHandler.Login)
	r.POST("/auth/social-login", authHandler.SocialLogin)
	r.POST("/payments/checkout", paymentHandler.CreateCheckoutSession)
	r.POST("/payments/webhook/stripe", paymentHandler.HandleStripeWebhook)

	api := r.Group("/api")
	api.Use(authMiddleware)
	writeLimited := api.Group("")
	if rateLimitMiddleware != nil {
		writeLimited.Use(rateLimitMiddleware)
	}

	api.GET("/me", meHandler.GetMe)
	writeLimited.POST("/payments/checkout", paymentHandler.CreateCheckoutSession)
	writeLimited.POST("/job/parse", jobHandler.ParseJobDescription)
	writeLimited.POST("/resume", resumeHandler.SaveResume)
	api.GET("/resume", resumeHandler.GetLatestResume)
	writeLimited.POST("/resume/analyze", resumeHandler.AnalyzeResume)
	api.GET("/resume/download", resumeHandler.DownloadLatestResume)
	writeLimited.POST("/questions/generate", questionHandler.GenerateQuestions)
	writeLimited.POST("/voice/tts", voiceHandler.TextToSpeech)
	writeLimited.POST("/voice/stt", voiceHandler.SpeechToText)
	writeLimited.POST("/voice/agent/session", voiceHandler.CreateAgentSession)
	writeLimited.POST("/voice/usage/commit", voiceHandler.CommitVoiceUsage)
	writeLimited.POST("/session/start", sessionHandler.StartSession)
	writeLimited.POST("/session/answer", sessionHandler.SubmitAnswer)
	writeLimited.POST("/session/heartbeat", sessionHandler.TouchSessionActivity)
	api.POST("/session/complete", sessionHandler.CompleteSession)
	api.GET("/session/history", sessionHandler.GetSessionHistory)
	api.GET("/subscription/status", subscriptionHandler.GetStatus)
	writeLimited.POST("/feedback/generate", feedbackHandler.GenerateFeedback)
	writeLimited.POST("/feedback/agent", feedbackHandler.SubmitAgentFeedback)
	api.GET("/progress", progressHandler.GetProgress)
	api.GET("/analytics/overview", progressHandler.GetAnalyticsOverview)

	return r
}
