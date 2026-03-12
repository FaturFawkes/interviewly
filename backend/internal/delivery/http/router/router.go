package router

import (
	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/delivery/http/handler"
)

// Setup registers all application routes and returns the configured Gin engine.
func Setup(
	healthHandler *handler.HealthHandler,
	meHandler *handler.MeHandler,
	authMiddleware gin.HandlerFunc,
) *gin.Engine {
	r := gin.Default()

	// Health check endpoint
	r.GET("/health", healthHandler.GetHealth)

	api := r.Group("/api")
	api.Use(authMiddleware)
	api.GET("/me", meHandler.GetMe)

	return r
}
