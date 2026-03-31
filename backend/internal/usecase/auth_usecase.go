package usecase

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/interview_app/backend/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type authUseCase struct {
	repo            domain.AuthRepository
	otpSender       otpSender
	jwtSecret       string
	jwtIssuer       string
	tokenTTL        time.Duration
	otpTTL          time.Duration
	resendTTL       time.Duration
	refreshTokenTTL time.Duration
	nowFactory      func() time.Time
}

type otpSender interface {
	SendRegistrationOTP(email, otp string) error
}

// NewAuthUseCase creates a use case for authentication workflows.
func NewAuthUseCase(repo domain.AuthRepository, sender otpSender, jwtSecret, jwtIssuer string, tokenTTL, otpTTL, refreshTokenTTL time.Duration) domain.AuthUseCase {
	if tokenTTL <= 0 {
		tokenTTL = time.Hour
	}
	if otpTTL <= 0 {
		otpTTL = 10 * time.Minute
	}
	if refreshTokenTTL <= 0 {
		refreshTokenTTL = 24 * time.Hour
	}

	return &authUseCase{
		repo:            repo,
		otpSender:       sender,
		jwtSecret:       strings.TrimSpace(jwtSecret),
		jwtIssuer:       strings.TrimSpace(jwtIssuer),
		tokenTTL:        tokenTTL,
		otpTTL:          otpTTL,
		resendTTL:       5 * time.Minute,
		refreshTokenTTL: refreshTokenTTL,
		nowFactory:      time.Now,
	}
}

func (uc *authUseCase) SocialLogin(input domain.SocialLoginInput) (*domain.AuthResult, error) {
	provider := strings.ToLower(strings.TrimSpace(input.Provider))
	if provider != "google" && provider != "microsoft" {
		return nil, errors.New("unsupported provider")
	}

	if strings.TrimSpace(input.ProviderAccountID) == "" {
		return nil, errors.New("provider account id is required")
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" {
		return nil, errors.New("email is required")
	}

	if uc.jwtSecret == "" {
		return nil, errors.New("jwt secret is not configured")
	}

	user, err := uc.repo.UpsertUserByEmail(email, input.FullName)
	if err != nil {
		return nil, err
	}

	return uc.buildAuthResult(user, provider)
}

func (uc *authUseCase) RequestRegisterOTP(input domain.RegisterOTPInput) (*domain.RegisterOTPResult, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" {
		return nil, errors.New("email is required")
	}

	password := strings.TrimSpace(input.Password)
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	existingCredential, err := uc.repo.GetCredentialByEmail(email)
	if err != nil {
		return nil, err
	}
	if existingCredential != nil {
		return nil, domain.ErrEmailAlreadyRegistered
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return uc.issueRegistrationOTP(email, strings.TrimSpace(input.FullName), string(passwordHash))
}

func (uc *authUseCase) ResendRegisterOTP(input domain.ResendRegisterOTPInput) (*domain.RegisterOTPResult, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" {
		return nil, errors.New("email is required")
	}

	record, err := uc.repo.GetRegistrationOTP(email)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, errors.New("otp request not found")
	}

	now := uc.nowFactory().UTC()
	if !record.RequestedAt.IsZero() {
		elapsed := now.Sub(record.RequestedAt)
		if elapsed < uc.resendTTL {
			retryAfter := int64((uc.resendTTL - elapsed).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			return nil, domain.ErrOTPResendTooSoon{RetryAfterSeconds: retryAfter}
		}
	}

	return uc.issueRegistrationOTP(record.Email, record.FullName, record.PasswordHash)
}

func (uc *authUseCase) issueRegistrationOTP(email, fullName, passwordHash string) (*domain.RegisterOTPResult, error) {
	otpCode, err := generateOTPCode(6)
	if err != nil {
		return nil, err
	}
	otpHash, err := bcrypt.GenerateFromPassword([]byte(otpCode), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := uc.nowFactory().UTC()
	expiresAt := now.Add(uc.otpTTL)

	err = uc.repo.SaveRegistrationOTP(domain.RegistrationOTP{
		Email:        email,
		FullName:     fullName,
		PasswordHash: passwordHash,
		OTPHash:      string(otpHash),
		ExpiresAt:    expiresAt,
		RequestedAt:  now,
		ConsumedAt:   nil,
	})
	if err != nil {
		return nil, err
	}

	if uc.otpSender == nil {
		return nil, errors.New("otp sender is not configured")
	}
	if err := uc.otpSender.SendRegistrationOTP(email, otpCode); err != nil {
		return nil, err
	}

	return &domain.RegisterOTPResult{
		ExpiresIn:       int64(uc.otpTTL.Seconds()),
		ResendAvailable: int64(uc.resendTTL.Seconds()),
	}, nil
}

func (uc *authUseCase) VerifyRegisterOTP(input domain.VerifyRegisterOTPInput) (*domain.VerifyRegisterOTPResult, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" {
		return nil, errors.New("email is required")
	}

	otpValue := strings.TrimSpace(input.OTP)
	if otpValue == "" {
		return nil, errors.New("otp is required")
	}

	record, err := uc.repo.GetRegistrationOTP(email)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, errors.New("otp request not found")
	}

	now := uc.nowFactory().UTC()
	if record.ConsumedAt != nil || now.After(record.ExpiresAt) {
		return nil, errors.New("otp expired")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(record.OTPHash), []byte(otpValue)); err != nil {
		return nil, errors.New("invalid otp")
	}

	_, err = uc.repo.CreateUserWithPassword(record.Email, record.FullName, record.PasswordHash)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.DeleteRegistrationOTP(record.Email); err != nil {
		return nil, err
	}

	return &domain.VerifyRegisterOTPResult{Message: "registration successful"}, nil
}

func (uc *authUseCase) Login(input domain.LoginInput) (*domain.AuthResult, error) {
	if uc.jwtSecret == "" {
		return nil, errors.New("jwt secret is not configured")
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" {
		return nil, errors.New("email is required")
	}

	password := strings.TrimSpace(input.Password)
	if password == "" {
		return nil, errors.New("password is required")
	}

	credential, err := uc.repo.GetCredentialByEmail(email)
	if err != nil {
		return nil, err
	}
	if credential == nil || strings.TrimSpace(credential.PasswordHash) == "" {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(credential.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return uc.buildAuthResult(&credential.User, "password")
}

func (uc *authUseCase) buildAuthResult(user *domain.User, provider string) (*domain.AuthResult, error) {
	now := uc.nowFactory().UTC()
	expiresAt := now.Add(uc.tokenTTL)

	claims := jwt.MapClaims{
		"sub":      user.ID,
		"email":    user.Email,
		"provider": provider,
		"iat":      now.Unix(),
		"exp":      expiresAt.Unix(),
	}

	if uc.jwtIssuer != "" {
		claims["iss"] = uc.jwtIssuer
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenValue, err := token.SignedString([]byte(uc.jwtSecret))
	if err != nil {
		return nil, err
	}

	refreshTokenValue, err := uc.generateRefreshToken(user.ID, now)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResult{
		AccessToken:           tokenValue,
		TokenType:             "Bearer",
		ExpiresIn:             int64(uc.tokenTTL.Seconds()),
		RefreshToken:          refreshTokenValue,
		RefreshTokenExpiresIn: int64(uc.refreshTokenTTL.Seconds()),
		User:                  *user,
	}, nil
}

func (uc *authUseCase) generateRefreshToken(userID string, now time.Time) (string, error) {
	if err := uc.repo.DeleteRefreshTokenByUserID(userID); err != nil {
		return "", err
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	tokenSecret := fmt.Sprintf("%x", buf)
	tokenValue := userID + "." + tokenSecret

	tokenHash, err := bcrypt.GenerateFromPassword([]byte(tokenSecret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	err = uc.repo.SaveRefreshToken(domain.RefreshTokenRecord{
		UserID:    userID,
		TokenHash: string(tokenHash),
		ExpiresAt: now.Add(uc.refreshTokenTTL),
		CreatedAt: now,
	})
	if err != nil {
		return "", err
	}

	return tokenValue, nil
}

func (uc *authUseCase) Refresh(input domain.RefreshInput) (*domain.AuthResult, error) {
	if uc.jwtSecret == "" {
		return nil, errors.New("jwt secret is not configured")
	}

	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return nil, errors.New("refresh token is required")
	}

	parts := strings.SplitN(refreshToken, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid refresh token")
	}
	userID := parts[0]
	tokenSecret := parts[1]

	record, err := uc.repo.GetRefreshTokenByUserID(userID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, errors.New("invalid refresh token")
	}

	now := uc.nowFactory().UTC()
	if now.After(record.ExpiresAt) {
		_ = uc.repo.DeleteRefreshTokenByUserID(userID)
		return nil, errors.New("refresh token expired")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(record.TokenHash), []byte(tokenSecret)); err != nil {
		return nil, errors.New("invalid refresh token")
	}

	user, err := uc.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	return uc.buildAuthResult(user, "refresh")
}

func generateOTPCode(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("invalid otp length")
	}

	max := big.NewInt(10)
	buf := make([]byte, length)
	for index := 0; index < length; index++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		buf[index] = byte('0' + n.Int64())
	}

	return fmt.Sprintf("%s", string(buf)), nil
}
