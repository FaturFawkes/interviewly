package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration.
type Config struct {
	ServerPort                    string
	Env                           string
	DatabaseURL                   string
	PostgresMaxConns              int32
	PostgresMinConns              int32
	PostgresMaxConnLifetimeMinute int
	RedisAddr                     string
	RedisPassword                 string
	RedisDB                       int
	JWTSecret                     string
	JWTIssuer                     string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		ServerPort:                    getEnv("PORT", "8080"),
		Env:                           getEnv("APP_ENV", "development"),
		DatabaseURL:                   getEnv("DATABASE_URL", ""),
		PostgresMaxConns:              getEnvInt32("POSTGRES_MAX_CONNS", 10),
		PostgresMinConns:              getEnvInt32("POSTGRES_MIN_CONNS", 1),
		PostgresMaxConnLifetimeMinute: getEnvInt("POSTGRES_MAX_CONN_LIFETIME_MINUTES", 30),
		RedisAddr:                     getEnv("REDIS_ADDR", ""),
		RedisPassword:                 getEnv("REDIS_PASSWORD", ""),
		RedisDB:                       getEnvInt("REDIS_DB", 0),
		JWTSecret:                     getEnv("JWT_SECRET", ""),
		JWTIssuer:                     getEnv("JWT_ISSUER", ""),
	}
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvInt32(key string, fallback int32) int32 {
	value := getEnv(key, "")
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return fallback
	}

	return int32(parsed)
}
