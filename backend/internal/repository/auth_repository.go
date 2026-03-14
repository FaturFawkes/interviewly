package repository

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/interview_app/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type authRepository struct {
	pool         *pgxpool.Pool
	mu           sync.RWMutex
	users        map[string]domain.User
	credentials  map[string]string
	registerOTPs map[string]domain.RegistrationOTP
}

// NewAuthRepository creates auth repository with postgres support and in-memory fallback.
func NewAuthRepository(pool *pgxpool.Pool) domain.AuthRepository {
	return &authRepository{
		pool:         pool,
		users:        make(map[string]domain.User),
		credentials:  make(map[string]string),
		registerOTPs: make(map[string]domain.RegistrationOTP),
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
