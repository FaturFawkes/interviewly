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
	SMTPHost                      string
	SMTPPort                      int
	SMTPUsername                  string
	SMTPPassword                  string
	SMTPFromEmail                 string
	SMTPFromName                  string
	OTPExpiryMinutes              int
	AIProvider                    string
	AIModel                       string
	AIAPIBaseURL                  string
	AIAPIKey                      string
	VoiceProvider                 string
	ElevenLabsAPIKey              string
	ElevenLabsVoiceID             string
	ElevenLabsTTSModel            string
	ElevenLabsSTTModel            string
	ElevenLabsAgentID             string
	ElevenLabsAgentBranchID       string
	MinIOEndpoint                 string
	MinIOAccessKey                string
	MinIOSecretKey                string
	MinIOBucket                   string
	MinIORegion                   string
	MinIOUseSSL                   bool
	StripeSecretKey               string
	StripeSuccessURL              string
	StripeCancelURL               string
	StripeCurrency                string
	StripePriceStarterMonthly     string
	StripePriceProMonthly         string
	StripePriceEliteMonthly       string
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
		SMTPHost:                      getEnv("SMTP_HOST", ""),
		SMTPPort:                      getEnvInt("SMTP_PORT", 587),
		SMTPUsername:                  getEnv("SMTP_USERNAME", ""),
		SMTPPassword:                  getEnv("SMTP_PASSWORD", ""),
		SMTPFromEmail:                 getEnv("SMTP_FROM_EMAIL", "no-reply@interviewly.local"),
		SMTPFromName:                  getEnv("SMTP_FROM_NAME", "Interviewly"),
		OTPExpiryMinutes:              getEnvInt("OTP_EXPIRY_MINUTES", 10),
		AIProvider:                    strings.ToLower(getEnv("AI_PROVIDER", "local")),
		AIModel:                       getEnv("AI_MODEL", "gpt-4o-mini"),
		AIAPIBaseURL:                  getEnv("AI_API_BASE_URL", "https://api.openai.com/v1"),
		AIAPIKey:                      getEnv("AI_API_KEY", ""),
		VoiceProvider:                 strings.ToLower(getEnv("VOICE_PROVIDER", "elevenlabs")),
		ElevenLabsAPIKey:              getEnv("ELEVENLABS_API_KEY", ""),
		ElevenLabsVoiceID:             getEnv("ELEVENLABS_VOICE_ID", "EXAVITQu4vr4xnSDxMaL"),
		ElevenLabsTTSModel:            getEnv("ELEVENLABS_TTS_MODEL", "eleven_multilingual_v2"),
		ElevenLabsSTTModel:            getEnv("ELEVENLABS_STT_MODEL", "scribe_v1"),
		ElevenLabsAgentID:             getEnv("ELEVENLABS_AGENT_ID", ""),
		ElevenLabsAgentBranchID:       getEnv("ELEVENLABS_AGENT_BRANCH_ID", ""),
		MinIOEndpoint:                 getEnv("MINIO_ENDPOINT", ""),
		MinIOAccessKey:                getEnv("MINIO_ACCESS_KEY", ""),
		MinIOSecretKey:                getEnv("MINIO_SECRET_KEY", ""),
		MinIOBucket:                   getEnv("MINIO_BUCKET", "interview-cv"),
		MinIORegion:                   getEnv("MINIO_REGION", "us-east-1"),
		MinIOUseSSL:                   getEnvBool("MINIO_USE_SSL", false),
		StripeSecretKey:               getEnv("STRIPE_SECRET_KEY", ""),
		StripeSuccessURL:              getEnv("STRIPE_SUCCESS_URL", "http://localhost:3000?payment=success"),
		StripeCancelURL:               getEnv("STRIPE_CANCEL_URL", "http://localhost:3000?payment=cancel"),
		StripeCurrency:                strings.ToLower(getEnv("STRIPE_CURRENCY", "usd")),
		StripePriceStarterMonthly:     getEnv("STRIPE_PRICE_STARTER_MONTHLY", ""),
		StripePriceProMonthly:         getEnv("STRIPE_PRICE_PRO_MONTHLY", ""),
		StripePriceEliteMonthly:       getEnv("STRIPE_PRICE_ELITE_MONTHLY", ""),
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

func getEnvBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(getEnv(key, "")))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}
