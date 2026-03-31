package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration.
type Config struct {
	ServerPort                          string
	Env                                 string
	DatabaseURL                         string
	PostgresMaxConns                    int32
	PostgresMinConns                    int32
	PostgresMaxConnLifetimeMinute       int
	RedisAddr                           string
	RedisPassword                       string
	RedisDB                             int
	JWTSecret                           string
	JWTIssuer                           string
	SMTPHost                            string
	SMTPPort                            int
	SMTPUsername                        string
	SMTPPassword                        string
	SMTPFromEmail                       string
	SMTPFromName                        string
	OTPExpiryMinutes                    int
	AccessTokenTTLMinutes               int
	RefreshTokenTTLHours                int
	AIProvider                          string
	AIModel                             string
	AIModelFUPDowngrade                 string
	AIAPIBaseURL                        string
	AIAPIKey                            string
	VoiceProvider                       string
	ElevenLabsAPIKey                    string
	ElevenLabsVoiceID                   string
	ElevenLabsTTSModel                  string
	ElevenLabsSTTModel                  string
	ElevenLabsAgentID                   string
	ElevenLabsAgentBranchID             string
	ElevenLabsReviewAgentID             string
	ElevenLabsReviewAgentBranchID       string
	SupabaseURL                         string
	SupabaseS3Endpoint                  string
	SupabaseS3Region                    string
	SupabaseS3AccessKeyID               string
	SupabaseS3SecretAccessKey           string
	SupabaseStorageBucket               string
	SupabaseStoragePathPrefix           string
	MinIOEndpoint                       string
	MinIOAccessKey                      string
	MinIOSecretKey                      string
	MinIOBucket                         string
	MinIORegion                         string
	MinIOUseSSL                         bool
	StripeSecretKey                     string
	StripeSuccessURL                    string
	StripeCancelURL                     string
	StripeCurrency                      string
	StripePriceStarterMonthly           string
	StripePriceProMonthly               string
	StripePriceEliteMonthly             string
	StripePriceVoiceTopup10             string
	StripePriceVoiceTopup30             string
	StripeWebhookSecret                 string
	VoiceTopup10AmountIDR               int
	VoiceTopup30AmountIDR               int
	SubscriptionWarningThresholdPercent int
	SubscriptionFUPDelaySeconds         int
	SubscriptionRateLimitPerMinute      int
	SessionIdleTimeoutSeconds           int
	SessionIdleSweepIntervalSeconds     int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		ServerPort:                          getEnv("PORT", "8080"),
		Env:                                 getEnv("APP_ENV", "development"),
		DatabaseURL:                         getEnv("DATABASE_URL", ""),
		PostgresMaxConns:                    getEnvInt32("POSTGRES_MAX_CONNS", 10),
		PostgresMinConns:                    getEnvInt32("POSTGRES_MIN_CONNS", 1),
		PostgresMaxConnLifetimeMinute:       getEnvInt("POSTGRES_MAX_CONN_LIFETIME_MINUTES", 30),
		RedisAddr:                           getEnv("REDIS_ADDR", ""),
		RedisPassword:                       getEnv("REDIS_PASSWORD", ""),
		RedisDB:                             getEnvInt("REDIS_DB", 0),
		JWTSecret:                           getEnv("JWT_SECRET", ""),
		JWTIssuer:                           getEnv("JWT_ISSUER", ""),
		SMTPHost:                            getEnv("SMTP_HOST", ""),
		SMTPPort:                            getEnvInt("SMTP_PORT", 587),
		SMTPUsername:                        getEnv("SMTP_USERNAME", ""),
		SMTPPassword:                        getEnv("SMTP_PASSWORD", ""),
		SMTPFromEmail:                       getEnv("SMTP_FROM_EMAIL", "no-reply@interviewly.local"),
		SMTPFromName:                        getEnv("SMTP_FROM_NAME", "Interviewly"),
		OTPExpiryMinutes:                    getEnvInt("OTP_EXPIRY_MINUTES", 10),
		AccessTokenTTLMinutes:               getEnvInt("ACCESS_TOKEN_TTL_MINUTES", 60),
		RefreshTokenTTLHours:                getEnvInt("REFRESH_TOKEN_TTL_HOURS", 24),
		AIProvider:                          strings.ToLower(getEnv("AI_PROVIDER", "local")),
		AIModel:                             getEnv("AI_MODEL", "gpt-4o-mini"),
		AIModelFUPDowngrade:                 getEnv("AI_MODEL_FUP_DOWNGRADE", ""),
		AIAPIBaseURL:                        getEnv("AI_API_BASE_URL", "https://api.openai.com/v1"),
		AIAPIKey:                            getEnv("AI_API_KEY", ""),
		VoiceProvider:                       strings.ToLower(getEnv("VOICE_PROVIDER", "elevenlabs")),
		ElevenLabsAPIKey:                    getEnv("ELEVENLABS_API_KEY", ""),
		ElevenLabsVoiceID:                   getEnv("ELEVENLABS_VOICE_ID", "EXAVITQu4vr4xnSDxMaL"),
		ElevenLabsTTSModel:                  getEnv("ELEVENLABS_TTS_MODEL", "eleven_multilingual_v2"),
		ElevenLabsSTTModel:                  getEnv("ELEVENLABS_STT_MODEL", "scribe_v1"),
		ElevenLabsAgentID:                   getEnv("ELEVENLABS_AGENT_ID", ""),
		ElevenLabsAgentBranchID:             getEnv("ELEVENLABS_AGENT_BRANCH_ID", ""),
		ElevenLabsReviewAgentID:             getEnv("ELEVENLABS_REVIEW_AGENT_ID", ""),
		ElevenLabsReviewAgentBranchID:       getEnv("ELEVENLABS_REVIEW_AGENT_BRANCH_ID", ""),
		SupabaseURL:                         getEnv("SUPABASE_URL", ""),
		SupabaseS3Endpoint:                  getEnv("SUPABASE_S3_ENDPOINT", ""),
		SupabaseS3Region:                    getEnv("SUPABASE_S3_REGION", "us-east-1"),
		SupabaseS3AccessKeyID:               getEnv("SUPABASE_S3_ACCESS_KEY_ID", ""),
		SupabaseS3SecretAccessKey:           getEnv("SUPABASE_S3_SECRET_ACCESS_KEY", ""),
		SupabaseStorageBucket:               getEnv("SUPABASE_STORAGE_BUCKET", "resumes"),
		SupabaseStoragePathPrefix:           getEnv("SUPABASE_STORAGE_PATH_PREFIX", ""),
		MinIOEndpoint:                       getEnv("MINIO_ENDPOINT", ""),
		MinIOAccessKey:                      getEnv("MINIO_ACCESS_KEY", ""),
		MinIOSecretKey:                      getEnv("MINIO_SECRET_KEY", ""),
		MinIOBucket:                         getEnv("MINIO_BUCKET", "interview-cv"),
		MinIORegion:                         getEnv("MINIO_REGION", "us-east-1"),
		MinIOUseSSL:                         getEnvBool("MINIO_USE_SSL", false),
		StripeSecretKey:                     getEnv("STRIPE_SECRET_KEY", ""),
		StripeSuccessURL:                    getEnv("STRIPE_SUCCESS_URL", "http://localhost:3000?payment=success"),
		StripeCancelURL:                     getEnv("STRIPE_CANCEL_URL", "http://localhost:3000?payment=cancel"),
		StripeCurrency:                      strings.ToLower(getEnv("STRIPE_CURRENCY", "idr")),
		StripePriceStarterMonthly:           getEnv("STRIPE_PRICE_STARTER_MONTHLY", ""),
		StripePriceProMonthly:               getEnv("STRIPE_PRICE_PRO_MONTHLY", ""),
		StripePriceEliteMonthly:             getEnv("STRIPE_PRICE_ELITE_MONTHLY", ""),
		StripePriceVoiceTopup10:             getEnv("STRIPE_PRICE_VOICE_TOPUP_10", ""),
		StripePriceVoiceTopup30:             getEnv("STRIPE_PRICE_VOICE_TOPUP_30", ""),
		StripeWebhookSecret:                 getEnv("STRIPE_WEBHOOK_SECRET", ""),
		VoiceTopup10AmountIDR:               getEnvInt("VOICE_TOPUP_10_AMOUNT_IDR", 19000),
		VoiceTopup30AmountIDR:               getEnvInt("VOICE_TOPUP_30_AMOUNT_IDR", 49000),
		SubscriptionWarningThresholdPercent: getEnvInt("SUBSCRIPTION_WARNING_THRESHOLD_PERCENT", 10),
		SubscriptionFUPDelaySeconds:         getEnvInt("SUBSCRIPTION_FUP_DELAY_SECONDS", 5),
		SubscriptionRateLimitPerMinute:      getEnvInt("SUBSCRIPTION_RATE_LIMIT_PER_MINUTE", 10),
		SessionIdleTimeoutSeconds:           getEnvInt("SESSION_IDLE_TIMEOUT_SECONDS", 300),
		SessionIdleSweepIntervalSeconds:     getEnvInt("SESSION_IDLE_SWEEP_INTERVAL_SECONDS", 30),
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
