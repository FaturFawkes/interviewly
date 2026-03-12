package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/interview_app/backend/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool creates and validates a pgx connection pool.
func NewPostgresPool(cfg *config.Config) (*pgxpool.Pool, error) {
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		return nil, nil
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed parsing DATABASE_URL: %w", err)
	}

	poolConfig.MinConns = cfg.PostgresMinConns
	poolConfig.MaxConns = cfg.PostgresMaxConns
	poolConfig.MaxConnLifetime = time.Duration(cfg.PostgresMaxConnLifetimeMinute) * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed creating postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed pinging postgres: %w", err)
	}

	return pool, nil
}
