package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/interview_app/backend/config"
	"github.com/interview_app/backend/internal/delivery/http/handler"
	"github.com/interview_app/backend/internal/delivery/http/middleware"
	"github.com/interview_app/backend/internal/delivery/http/router"
	"github.com/interview_app/backend/internal/infrastructure/cache"
	"github.com/interview_app/backend/internal/infrastructure/database"
	"github.com/interview_app/backend/internal/repository"
	"github.com/interview_app/backend/internal/service/ai"
	"github.com/interview_app/backend/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// Load configuration
	cfg := config.Load()
	postgresPool, postgresCleanup := setupPostgres(cfg)
	if postgresCleanup != nil {
		defer postgresCleanup()
	}

	redisCleanup := setupRedis(cfg)
	if redisCleanup != nil {
		defer redisCleanup()
	}

	// Wire dependencies (repository → usecase → handler)
	healthRepo := repository.NewHealthRepository()
	healthUC := usecase.NewHealthUseCase(healthRepo)
	healthHandler := handler.NewHealthHandler(healthUC)
	aiService := ai.NewService()
	interviewRepo := repository.NewInterviewRepository(postgresPool)
	interviewUC := usecase.NewInterviewUseCase(aiService, interviewRepo)
	jobHandler := handler.NewJobHandler(interviewUC)
	resumeHandler := handler.NewResumeHandler(interviewUC)
	questionHandler := handler.NewQuestionHandler(interviewUC)
	sessionHandler := handler.NewSessionHandler(interviewUC)
	feedbackHandler := handler.NewFeedbackHandler(interviewUC)
	meHandler := handler.NewMeHandler()
	authMiddleware := middleware.AuthMiddleware(cfg)

	// Setup router
	r := router.Setup(healthHandler, meHandler, jobHandler, resumeHandler, questionHandler, sessionHandler, feedbackHandler, authMiddleware)

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Server starting on %s (env: %s)", addr, cfg.Env)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func setupPostgres(cfg *config.Config) (*pgxpool.Pool, func()) {
	pool, err := database.NewPostgresPool(cfg)
	if err != nil {
		log.Printf("PostgreSQL not initialized: %v", err)
		return nil, nil
	}
	if pool == nil {
		log.Println("PostgreSQL not initialized: DATABASE_URL is empty")
		return nil, nil
	}

	log.Println("database connected")
	return pool, pool.Close
}

func setupRedis(cfg *config.Config) func() {
	redisCache, err := cache.NewRedisCache(cfg)
	if err != nil {
		log.Printf("Redis not initialized: %v", err)
		return nil
	}
	if redisCache == nil {
		log.Println("Redis not initialized: REDIS_ADDR is empty")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := redisCache.Set(ctx, "cache:startup:ping", "pong", time.Minute); err != nil {
		log.Printf("Redis SET failed: %v", err)
		return func() {
			_ = redisCache.Close()
		}
	}

	value, err := redisCache.Get(ctx, "cache:startup:ping")
	if err != nil {
		log.Printf("Redis GET failed: %v", err)
		return func() {
			_ = redisCache.Close()
		}
	}

	log.Printf("Redis cache verified, value=%s", value)
	return func() {
		_ = redisCache.Close()
	}
}
