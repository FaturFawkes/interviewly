package domain

import (
	"errors"
	"fmt"
	"time"
)

var ErrEmailAlreadyRegistered = errors.New("email is already registered")

type ErrOTPResendTooSoon struct {
	RetryAfterSeconds int64
}

func (e ErrOTPResendTooSoon) Error() string {
	return fmt.Sprintf("please wait %d seconds before resending otp", e.RetryAfterSeconds)
}

// User represents an authenticated application user.
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SocialLoginInput is the payload required to login via social provider.
type SocialLoginInput struct {
	Provider          string `json:"provider"`
	ProviderAccountID string `json:"provider_account_id"`
	Email             string `json:"email"`
	FullName          string `json:"full_name"`
}

// RegisterInput is the payload required to register via email/password.
type RegisterInput struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
}

// LoginInput is the payload required to login via email/password.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterOTPInput is the payload to request registration OTP.
type RegisterOTPInput struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
}

// VerifyRegisterOTPInput is the payload to verify registration OTP.
type VerifyRegisterOTPInput struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

// ResendRegisterOTPInput is the payload to resend registration OTP.
type ResendRegisterOTPInput struct {
	Email string `json:"email"`
}

// UserCredential stores user profile and password hash.
type UserCredential struct {
	User         User
	PasswordHash string
}

// RegistrationOTP stores pending registration data waiting for OTP verification.
type RegistrationOTP struct {
	Email        string
	FullName     string
	PasswordHash string
	OTPHash      string
	ExpiresAt    time.Time
	RequestedAt  time.Time
	ConsumedAt   *time.Time
}

// RegisterOTPResult is returned when OTP is created and sent.
type RegisterOTPResult struct {
	ExpiresIn       int64 `json:"expires_in"`
	ResendAvailable int64 `json:"resend_available_in"`
}

// VerifyRegisterOTPResult is returned after successful OTP verification.
type VerifyRegisterOTPResult struct {
	Message string `json:"message"`
}

// AuthResult is returned by auth workflows.
type AuthResult struct {
	AccessToken           string `json:"access_token"`
	TokenType             string `json:"token_type"`
	ExpiresIn             int64  `json:"expires_in"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresIn int64  `json:"refresh_token_expires_in"`
	User                  User   `json:"user"`
}

// RefreshTokenRecord stores a hashed refresh token for a user.
type RefreshTokenRecord struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// RefreshInput is the payload to refresh tokens.
type RefreshInput struct {
	RefreshToken string `json:"refresh_token"`
}

// AuthRepository defines persistence required by authentication workflows.
type AuthRepository interface {
	UpsertUserByEmail(email, fullName string) (*User, error)
	CreateUserWithPassword(email, fullName, passwordHash string) (*User, error)
	GetCredentialByEmail(email string) (*UserCredential, error)
	SaveRegistrationOTP(record RegistrationOTP) error
	GetRegistrationOTP(email string) (*RegistrationOTP, error)
	DeleteRegistrationOTP(email string) error
	SaveRefreshToken(record RefreshTokenRecord) error
	GetRefreshTokenByUserID(userID string) (*RefreshTokenRecord, error)
	DeleteRefreshTokenByUserID(userID string) error
	GetUserByID(userID string) (*User, error)
}

// AuthUseCase defines authentication workflows.
type AuthUseCase interface {
	SocialLogin(input SocialLoginInput) (*AuthResult, error)
	RequestRegisterOTP(input RegisterOTPInput) (*RegisterOTPResult, error)
	ResendRegisterOTP(input ResendRegisterOTPInput) (*RegisterOTPResult, error)
	VerifyRegisterOTP(input VerifyRegisterOTPInput) (*VerifyRegisterOTPResult, error)
	Login(input LoginInput) (*AuthResult, error)
	Refresh(input RefreshInput) (*AuthResult, error)
}
