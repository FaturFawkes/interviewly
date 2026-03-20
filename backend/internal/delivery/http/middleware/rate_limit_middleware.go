package middleware

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/infrastructure/cache"
)

type inMemoryRateBucket struct {
	Count   int64
	ResetAt time.Time
}

var rateLimitMemoryStore = struct {
	mu      sync.Mutex
	buckets map[string]inMemoryRateBucket
}{
	buckets: map[string]inMemoryRateBucket{},
}

// RateLimitMiddleware limits requests per user and route in a fixed window.
func RateLimitMiddleware(redisCache *cache.RedisCache, limitPerWindow int, window time.Duration) gin.HandlerFunc {
	if limitPerWindow <= 0 {
		limitPerWindow = 10
	}
	if window <= 0 {
		window = time.Minute
	}

	return func(c *gin.Context) {
		now := time.Now().UTC()
		subject := resolveRateLimitSubject(c)
		route := strings.TrimSpace(c.FullPath())
		if route == "" {
			route = strings.TrimSpace(c.Request.URL.Path)
		}
		key := fmt.Sprintf("rate-limit:%s:%s", subject, route)

		count, resetAt := consumeRateLimit(redisCache, key, window, now)
		if count > int64(limitPerWindow) {
			retryAfter := int(math.Ceil(resetAt.Sub(now).Seconds()))
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter,
			})
			return
		}

		c.Next()
	}
}

func consumeRateLimit(redisCache *cache.RedisCache, key string, window time.Duration, now time.Time) (int64, time.Time) {
	if redisCache != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		count, err := redisCache.Incr(ctx, key)
		if err == nil {
			if count == 1 {
				_ = redisCache.Expire(ctx, key, window)
			}
			return count, now.Add(window)
		}
	}

	rateLimitMemoryStore.mu.Lock()
	defer rateLimitMemoryStore.mu.Unlock()

	bucket := rateLimitMemoryStore.buckets[key]
	if bucket.ResetAt.IsZero() || !now.Before(bucket.ResetAt) {
		bucket = inMemoryRateBucket{
			Count:   0,
			ResetAt: now.Add(window),
		}
	}
	bucket.Count++
	rateLimitMemoryStore.buckets[key] = bucket

	if len(rateLimitMemoryStore.buckets) > 10000 {
		for existingKey, existingBucket := range rateLimitMemoryStore.buckets {
			if !now.Before(existingBucket.ResetAt) {
				delete(rateLimitMemoryStore.buckets, existingKey)
			}
		}
	}

	return bucket.Count, bucket.ResetAt
}

func resolveRateLimitSubject(c *gin.Context) string {
	if userIDValue, exists := c.Get(UserIDContextKey); exists {
		if userID, ok := userIDValue.(string); ok && strings.TrimSpace(userID) != "" {
			return "user:" + strings.TrimSpace(userID)
		}
	}

	clientIP := strings.TrimSpace(c.ClientIP())
	if clientIP == "" {
		clientIP = "unknown"
	}
	return "ip:" + clientIP
}
