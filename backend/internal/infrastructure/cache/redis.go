package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/interview_app/backend/config"
	"github.com/redis/go-redis/v9"
)

// RedisCache provides simple set/get operations backed by Redis.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache creates and validates a Redis client.
func NewRedisCache(cfg *config.Config) (*RedisCache, error) {
	if strings.TrimSpace(cfg.RedisAddr) == "" {
		return nil, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed pinging redis: %w", err)
	}

	return &RedisCache{client: client}, nil
}

func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

func (r *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.client.Expire(ctx, key, ttl).Err()
}

func (r *RedisCache) Close() error {
	return r.client.Close()
}
