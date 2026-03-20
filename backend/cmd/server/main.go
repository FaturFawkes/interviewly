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
	"github.com/interview_app/backend/internal/domain"
	"github.com/interview_app/backend/internal/infrastructure/cache"
	"github.com/interview_app/backend/internal/infrastructure/database"
	"github.com/interview_app/backend/internal/repository"
	"github.com/interview_app/backend/internal/service/ai"
	"github.com/interview_app/backend/internal/service/notification"
	"github.com/interview_app/backend/internal/service/payment"
	"github.com/interview_app/backend/internal/service/subscription"
	"github.com/interview_app/backend/internal/service/voice"
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

	redisCache, redisCleanup := setupRedis(cfg)
	if redisCleanup != nil {
		defer redisCleanup()
	}

	// Wire dependencies (repository → usecase → handler)
	healthRepo := repository.NewHealthRepository()
	healthUC := usecase.NewHealthUseCase(healthRepo)
	healthHandler := handler.NewHealthHandler(healthUC)
	aiService := ai.NewService(cfg)
	interviewRepo := repository.NewInterviewRepository(postgresPool)
	subscriptionService := subscription.NewService(cfg, postgresPool, redisCache)
	interviewUC := usecase.NewInterviewUseCase(aiService, interviewRepo, subscriptionService)
	jobHandler := handler.NewJobHandler(interviewUC)
	resumeHandler := handler.NewResumeHandler(interviewUC)
	questionHandler := handler.NewQuestionHandler(interviewUC)
	subscriptionHandler := handler.NewSubscriptionHandler(subscriptionService)
	voiceService := voice.NewService(cfg)
	voiceHandler := handler.NewVoiceHandler(voiceService, subscriptionService)
	paymentService := payment.NewService(cfg, postgresPool, subscriptionService)
	paymentHandler := handler.NewPaymentHandler(paymentService)
	sessionHandler := handler.NewSessionHandler(interviewUC, subscriptionService)
	feedbackHandler := handler.NewFeedbackHandler(interviewUC)
	progressHandler := handler.NewProgressHandler(interviewUC)
	meHandler := handler.NewMeHandler()
	authRepo := repository.NewAuthRepository(postgresPool)
	otpSender := notification.NewRegistrationOTPSender(cfg)
	authUC := usecase.NewAuthUseCase(authRepo, otpSender, cfg.JWTSecret, cfg.JWTIssuer, 24*time.Hour, time.Duration(cfg.OTPExpiryMinutes)*time.Minute)
	authHandler := handler.NewAuthHandler(authUC)
	authMiddleware := middleware.AuthMiddleware(cfg)
	rateLimitMiddleware := middleware.RateLimitMiddleware(redisCache, cfg.SubscriptionRateLimitPerMinute, time.Minute)

	// Setup router
	r := router.Setup(healthHandler, authHandler, meHandler, jobHandler, resumeHandler, questionHandler, voiceHandler, paymentHandler, sessionHandler, subscriptionHandler, feedbackHandler, progressHandler, authMiddleware, rateLimitMiddleware)
	startIdleSessionSweeper(cfg, interviewUC)

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

func setupRedis(cfg *config.Config) (*cache.RedisCache, func()) {
	redisCache, err := cache.NewRedisCache(cfg)
	if err != nil {
		log.Printf("Redis not initialized: %v", err)
		return nil, nil
	}
	if redisCache == nil {
		log.Println("Redis not initialized: REDIS_ADDR is empty")
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := redisCache.Set(ctx, "cache:startup:ping", "pong", time.Minute); err != nil {
		log.Printf("Redis SET failed: %v", err)
		return redisCache, func() {
			_ = redisCache.Close()
		}
	}

	value, err := redisCache.Get(ctx, "cache:startup:ping")
	if err != nil {
		log.Printf("Redis GET failed: %v", err)
		return redisCache, func() {
			_ = redisCache.Close()
		}
	}

	log.Printf("Redis cache verified, value=%s", value)
	return redisCache, func() {
		_ = redisCache.Close()
	}
}

func startIdleSessionSweeper(cfg *config.Config, interviewUC domain.InterviewUseCase) {
	if cfg == nil || interviewUC == nil {
		return
	}

	idleTimeout := time.Duration(cfg.SessionIdleTimeoutSeconds) * time.Second
	if idleTimeout <= 0 {
		return
	}

	sweepInterval := time.Duration(cfg.SessionIdleSweepIntervalSeconds) * time.Second
	if sweepInterval <= 0 {
		sweepInterval = 30 * time.Second
	}

	log.Printf("Idle session sweeper started (timeout=%s, interval=%s)", idleTimeout, sweepInterval)

	go func() {
		ticker := time.NewTicker(sweepInterval)
		defer ticker.Stop()

		for range ticker.C {
			abandonedCount, err := interviewUC.AbandonIdleSessions(idleTimeout)
			if err != nil {
				log.Printf("Idle session sweeper error: %v", err)
				continue
			}

			if abandonedCount > 0 {
				log.Printf("Idle session sweeper abandoned %d sessions", abandonedCount)
			}
		}
	}()
}
