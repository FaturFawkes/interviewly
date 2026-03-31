package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/interview_app/backend/internal/domain"
	"github.com/interview_app/backend/internal/infrastructure/cache"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const refreshTokenCachePrefix = "auth:rt:"

type cacheClient interface {
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
}

type authRepository struct {
	pool          *pgxpool.Pool
	cache         cacheClient
	mu            sync.RWMutex
	users         map[string]domain.User
	credentials   map[string]string
	registerOTPs  map[string]domain.RegistrationOTP
	refreshTokens map[string]domain.RefreshTokenRecord
}

// NewAuthRepository creates auth repository with postgres and optional Redis cache.
func NewAuthRepository(pool *pgxpool.Pool, rc *cache.RedisCache) domain.AuthRepository {
	var c cacheClient
	if rc != nil {
		c = rc
	}
	return &authRepository{
		pool:          pool,
		cache:         c,
		users:         make(map[string]domain.User),
		credentials:   make(map[string]string),
		registerOTPs:  make(map[string]domain.RegistrationOTP),
		refreshTokens: make(map[string]domain.RefreshTokenRecord),
	}
}

func (r *authRepository) UpsertUserByEmail(email, fullName string) (*domain.User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	normalizedName := strings.TrimSpace(fullName)

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var user domain.User
		err := r.pool.QueryRow(
			ctx,
			`INSERT INTO users (email, full_name)
			 VALUES ($1, NULLIF($2, ''))
			 ON CONFLICT (email)
			 DO UPDATE SET
			   full_name = COALESCE(NULLIF(EXCLUDED.full_name, ''), users.full_name),
			   updated_at = NOW()
			 RETURNING id::text, email, COALESCE(full_name, ''), created_at, updated_at`,
			normalizedEmail,
			normalizedName,
		).Scan(
			&user.ID,
			&user.Email,
			&user.FullName,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err == nil {
			return &user, nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.users[normalizedEmail]
	if exists {
		if normalizedName != "" {
			existing.FullName = normalizedName
		}
		existing.UpdatedAt = time.Now().UTC()
		r.users[normalizedEmail] = existing
		return &existing, nil
	}

	now := time.Now().UTC()
	user := domain.User{
		ID:        uuid.NewString(),
		Email:     normalizedEmail,
		FullName:  normalizedName,
		CreatedAt: now,
		UpdatedAt: now,
	}
	r.users[normalizedEmail] = user

	return &user, nil
}

func (r *authRepository) CreateUserWithPassword(email, fullName, passwordHash string) (*domain.User, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	normalizedName := strings.TrimSpace(fullName)

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var user domain.User
		err := r.pool.QueryRow(
			ctx,
			`INSERT INTO users (email, full_name, password_hash)
			 VALUES ($1, NULLIF($2, ''), $3)
			 RETURNING id::text, email, COALESCE(full_name, ''), created_at, updated_at`,
			normalizedEmail,
			normalizedName,
			passwordHash,
		).Scan(
			&user.ID,
			&user.Email,
			&user.FullName,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err == nil {
			return &user, nil
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.ErrEmailAlreadyRegistered
		}
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[normalizedEmail]; exists {
		return nil, domain.ErrEmailAlreadyRegistered
	}

	now := time.Now().UTC()
	user := domain.User{
		ID:        uuid.NewString(),
		Email:     normalizedEmail,
		FullName:  normalizedName,
		CreatedAt: now,
		UpdatedAt: now,
	}
	r.users[normalizedEmail] = user
	r.credentials[normalizedEmail] = passwordHash

	return &user, nil
}

func (r *authRepository) GetCredentialByEmail(email string) (*domain.UserCredential, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var user domain.User
		var passwordHash string
		err := r.pool.QueryRow(
			ctx,
			`SELECT id::text, email, COALESCE(full_name, ''), COALESCE(password_hash, ''), created_at, updated_at
			   FROM users
			  WHERE email = $1`,
			normalizedEmail,
		).Scan(
			&user.ID,
			&user.Email,
			&user.FullName,
			&passwordHash,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err == nil {
			return &domain.UserCredential{User: user, PasswordHash: passwordHash}, nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[normalizedEmail]
	if !exists {
		return nil, nil
	}

	return &domain.UserCredential{
		User:         user,
		PasswordHash: r.credentials[normalizedEmail],
	}, nil
}

func (r *authRepository) SaveRegistrationOTP(record domain.RegistrationOTP) error {
	normalizedEmail := strings.ToLower(strings.TrimSpace(record.Email))
	record.Email = normalizedEmail
	if record.RequestedAt.IsZero() {
		record.RequestedAt = time.Now().UTC()
	}

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO registration_otps (email, full_name, password_hash, otp_hash, expires_at, consumed_at, updated_at)
			 VALUES ($1, NULLIF($2, ''), $3, $4, $5, NULL, $6)
			 ON CONFLICT (email)
			 DO UPDATE SET
			   full_name = EXCLUDED.full_name,
			   password_hash = EXCLUDED.password_hash,
			   otp_hash = EXCLUDED.otp_hash,
			   expires_at = EXCLUDED.expires_at,
			   consumed_at = NULL,
			   updated_at = EXCLUDED.updated_at`,
			record.Email,
			record.FullName,
			record.PasswordHash,
			record.OTPHash,
			record.ExpiresAt,
			record.RequestedAt,
		)
		if err == nil {
			return nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.registerOTPs[normalizedEmail] = record
	return nil
}

func (r *authRepository) GetRegistrationOTP(email string) (*domain.RegistrationOTP, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var record domain.RegistrationOTP
		err := r.pool.QueryRow(
			ctx,
			`SELECT email, COALESCE(full_name, ''), password_hash, otp_hash, expires_at, updated_at, consumed_at
			   FROM registration_otps
			  WHERE email = $1`,
			normalizedEmail,
		).Scan(
			&record.Email,
			&record.FullName,
			&record.PasswordHash,
			&record.OTPHash,
			&record.ExpiresAt,
			&record.RequestedAt,
			&record.ConsumedAt,
		)
		if err == nil {
			return &record, nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	record, exists := r.registerOTPs[normalizedEmail]
	if !exists {
		return nil, nil
	}

	copy := record
	return &copy, nil
}

func (r *authRepository) DeleteRegistrationOTP(email string) error {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := r.pool.Exec(ctx, `DELETE FROM registration_otps WHERE email = $1`, normalizedEmail)
		if err == nil {
			return nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.registerOTPs, normalizedEmail)
	return nil
}

func (r *authRepository) SaveRefreshToken(record domain.RefreshTokenRecord) error {
	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
			 VALUES ($1::uuid, $2, $3)
			 ON CONFLICT DO NOTHING`,
			record.UserID,
			record.TokenHash,
			record.ExpiresAt,
		)
		if err != nil {
			return err
		}
	} else {
		r.mu.Lock()
		r.refreshTokens[record.UserID] = record
		r.mu.Unlock()
	}

	if r.cache != nil {
		ttl := time.Until(record.ExpiresAt)
		if ttl > 0 {
			val := fmt.Sprintf("%s|%d", record.TokenHash, record.ExpiresAt.Unix())
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = r.cache.Set(ctx, refreshTokenCachePrefix+record.UserID, val, ttl)
		}
	}

	return nil
}

func (r *authRepository) GetRefreshTokenByUserID(userID string) (*domain.RefreshTokenRecord, error) {
	// 1. Redis-first
	if r.cache != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		val, err := r.cache.Get(ctx, refreshTokenCachePrefix+userID)
		cancel()
		if err == nil && val != "" {
			parts := strings.SplitN(val, "|", 2)
			if len(parts) == 2 {
				expiresAtUnix, parseErr := strconv.ParseInt(parts[1], 10, 64)
				if parseErr == nil {
					expiresAt := time.Unix(expiresAtUnix, 0).UTC()
					if time.Now().UTC().Before(expiresAt) {
						return &domain.RefreshTokenRecord{
							UserID:    userID,
							TokenHash: parts[0],
							ExpiresAt: expiresAt,
						}, nil
					}
				}
			}
		}
	}

	// 2. Database fallback
	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var record domain.RefreshTokenRecord
		err := r.pool.QueryRow(
			ctx,
			`SELECT id::text, user_id::text, token_hash, expires_at, created_at
			   FROM refresh_tokens
			  WHERE user_id = $1::uuid
			  ORDER BY created_at DESC
			  LIMIT 1`,
			userID,
		).Scan(
			&record.ID,
			&record.UserID,
			&record.TokenHash,
			&record.ExpiresAt,
			&record.CreatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil
			}
			return nil, err
		}

		// Populate Redis on cache miss
		if r.cache != nil {
			ttl := time.Until(record.ExpiresAt)
			if ttl > 0 {
				val := fmt.Sprintf("%s|%d", record.TokenHash, record.ExpiresAt.Unix())
				cCtx, cCancel := context.WithTimeout(context.Background(), 2*time.Second)
				_ = r.cache.Set(cCtx, refreshTokenCachePrefix+userID, val, ttl)
				cCancel()
			}
		}

		return &record, nil
	}

	// 3. In-memory fallback
	r.mu.RLock()
	defer r.mu.RUnlock()
	record, exists := r.refreshTokens[userID]
	if !exists {
		return nil, nil
	}
	cp := record
	return &cp, nil
}

func (r *authRepository) DeleteRefreshTokenByUserID(userID string) error {
	// Delete from Redis first (fire-and-forget)
	if r.cache != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = r.cache.Del(ctx, refreshTokenCachePrefix+userID)
		cancel()
	}

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := r.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1::uuid`, userID)
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.refreshTokens, userID)
	return nil
}

func (r *authRepository) GetUserByID(userID string) (*domain.User, error) {
	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var user domain.User
		err := r.pool.QueryRow(
			ctx,
			`SELECT id::text, email, COALESCE(full_name, ''), created_at, updated_at
			   FROM users
			  WHERE id = $1::uuid`,
			userID,
		).Scan(
			&user.ID,
			&user.Email,
			&user.FullName,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err == nil {
			return &user, nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, u := range r.users {
		if u.ID == userID {
			cp := u
			return &cp, nil
		}
	}
	return nil, nil
}
