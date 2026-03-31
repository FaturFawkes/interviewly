package main

import (
	"context"
	"fmt"
	"time"

	"github.com/interview_app/backend/config"
	"github.com/interview_app/backend/internal/delivery/http/handler"
	"github.com/interview_app/backend/internal/delivery/http/middleware"
	"github.com/interview_app/backend/internal/delivery/http/router"
	"github.com/interview_app/backend/internal/domain"
	"github.com/interview_app/backend/internal/infrastructure/cache"
	"github.com/interview_app/backend/internal/infrastructure/database"
	"github.com/interview_app/backend/internal/logger"
	"github.com/interview_app/backend/internal/repository"
	"github.com/interview_app/backend/internal/service/ai"
	"github.com/interview_app/backend/internal/service/notification"
	"github.com/interview_app/backend/internal/service/payment"
	"github.com/interview_app/backend/internal/service/storage"
	"github.com/interview_app/backend/internal/service/subscription"
	"github.com/interview_app/backend/internal/service/voice"
	"github.com/interview_app/backend/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg := config.Load()
	logger.Init(cfg.Env)
	defer logger.Sync()

	log := logger.L()

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
	resumeStorage := setupResumeStorage(cfg)
	subscriptionService := subscription.NewService(cfg, postgresPool, redisCache)
	interviewUC := usecase.NewInterviewUseCase(aiService, interviewRepo, resumeStorage, subscriptionService)
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
	reviewHandler := handler.NewReviewHandler(interviewUC, subscriptionService)
	meHandler := handler.NewMeHandler()
	authRepo := repository.NewAuthRepository(postgresPool, redisCache)
	otpSender := notification.NewRegistrationOTPSender(cfg)
	authUC := usecase.NewAuthUseCase(authRepo, otpSender, cfg.JWTSecret, cfg.JWTIssuer, time.Duration(cfg.AccessTokenTTLMinutes)*time.Minute, time.Duration(cfg.OTPExpiryMinutes)*time.Minute, time.Duration(cfg.RefreshTokenTTLHours)*time.Hour)
	authHandler := handler.NewAuthHandler(authUC)
	authMiddleware := middleware.AuthMiddleware(cfg)
	rateLimitMiddleware := middleware.RateLimitMiddleware(redisCache, cfg.SubscriptionRateLimitPerMinute, time.Minute)

	// Setup router
	r := router.Setup(
		healthHandler,
		authHandler,
		meHandler,
		jobHandler,
		resumeHandler,
		questionHandler,
		voiceHandler,
		paymentHandler,
		sessionHandler,
		subscriptionHandler,
		feedbackHandler,
		progressHandler,
		reviewHandler,
		authMiddleware,
		rateLimitMiddleware,
	)
	startIdleSessionSweeper(cfg, interviewUC)

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Info("Server starting", zap.String("addr", addr), zap.String("env", cfg.Env))

	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start server", zap.Error(err))
	}
}

func setupPostgres(cfg *config.Config) (*pgxpool.Pool, func()) {
	log := logger.L()
	pool, err := database.NewPostgresPool(cfg)
	if err != nil {
		log.Warn("PostgreSQL not initialized", zap.Error(err))
		return nil, nil
	}
	if pool == nil {
		log.Warn("PostgreSQL not initialized: DATABASE_URL is empty")
		return nil, nil
	}

	log.Info("database connected")
	return pool, pool.Close
}

func setupRedis(cfg *config.Config) (*cache.RedisCache, func()) {
	log := logger.L()
	redisCache, err := cache.NewRedisCache(cfg)
	if err != nil {
		log.Warn("Redis not initialized", zap.Error(err))
		return nil, nil
	}
	if redisCache == nil {
		log.Warn("Redis not initialized: REDIS_ADDR is empty")
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := redisCache.Set(ctx, "cache:startup:ping", "pong", time.Minute); err != nil {
		log.Warn("Redis SET failed", zap.Error(err))
		return redisCache, func() {
			_ = redisCache.Close()
		}
	}

	value, err := redisCache.Get(ctx, "cache:startup:ping")
	if err != nil {
		log.Warn("Redis GET failed", zap.Error(err))
		return redisCache, func() {
			_ = redisCache.Close()
		}
	}

	log.Info("Redis cache verified", zap.String("value", value))
	return redisCache, func() {
		_ = redisCache.Close()
	}
}

func setupResumeStorage(cfg *config.Config) domain.ResumeFileStorage {
	log := logger.L()
	resumeStorage, err := storage.NewSupabaseResumeStorage(cfg)
	if err != nil {
		log.Warn("Supabase resume storage not initialized", zap.Error(err))
	} else if resumeStorage != nil {
		log.Info("Resume storage initialized", zap.String("provider", "supabase"))
		return resumeStorage
	}

	resumeStorage, err = storage.NewMinIOResumeStorage(cfg)
	if err != nil {
		log.Warn("MinIO resume storage not initialized", zap.Error(err))
		return nil
	}
	if resumeStorage != nil {
		log.Info("Resume storage initialized", zap.String("provider", "minio"))
	}

	return resumeStorage
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

	log := logger.L()
	log.Info("Idle session sweeper started",
		zap.Duration("timeout", idleTimeout),
		zap.Duration("interval", sweepInterval),
	)

	go func() {
		ticker := time.NewTicker(sweepInterval)
		defer ticker.Stop()

		for range ticker.C {
			abandonedCount, err := interviewUC.AbandonIdleSessions(idleTimeout)
			if err != nil {
				log.Error("Idle session sweeper error", zap.Error(err))
				continue
			}

			if abandonedCount > 0 {
				log.Info("Idle session sweeper abandoned sessions", zap.Int64("count", abandonedCount))
			}
		}
	}()
}
