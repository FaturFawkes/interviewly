package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/domain"
)

type socialLoginRequest struct {
	Provider          string `json:"provider" binding:"required"`
	ProviderAccountID string `json:"provider_account_id" binding:"required"`
	Email             string `json:"email" binding:"required,email"`
	FullName          string `json:"full_name"`
}

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	FullName string `json:"full_name"`
	Password string `json:"password" binding:"required"`
}

type verifyRegisterOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required"`
}

type resendRegisterOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// AuthHandler handles authentication APIs.
type AuthHandler struct {
	authUC domain.AuthUseCase
}

func NewAuthHandler(authUC domain.AuthUseCase) *AuthHandler {
	return &AuthHandler{authUC: authUC}
}

// SocialLogin handles POST /auth/social-login.
func (h *AuthHandler) SocialLogin(c *gin.Context) {
	var req socialLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.authUC.SocialLogin(domain.SocialLoginInput{
		Provider:          req.Provider,
		ProviderAccountID: req.ProviderAccountID,
		Email:             req.Email,
		FullName:          req.FullName,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.authUC.RequestRegisterOTP(domain.RegisterOTPInput{
		Email:    req.Email,
		FullName: req.FullName,
		Password: req.Password,
	})
	if err != nil {
		status := http.StatusBadRequest
		if err == domain.ErrEmailAlreadyRegistered {
			status = http.StatusConflict
		}
		if resendErr, ok := err.(domain.ErrOTPResendTooSoon); ok {
			status = http.StatusTooManyRequests
			c.JSON(status, gin.H{"error": resendErr.Error(), "retry_after": resendErr.RetryAfterSeconds})
			return
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ResendRegisterOTP handles POST /auth/register/resend.
func (h *AuthHandler) ResendRegisterOTP(c *gin.Context) {
	var req resendRegisterOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.authUC.ResendRegisterOTP(domain.ResendRegisterOTPInput{Email: req.Email})
	if err != nil {
		if resendErr, ok := err.(domain.ErrOTPResendTooSoon); ok {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": resendErr.Error(), "retry_after": resendErr.RetryAfterSeconds})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// VerifyRegisterOTP handles POST /auth/register/verify.
func (h *AuthHandler) VerifyRegisterOTP(c *gin.Context) {
	var req verifyRegisterOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.authUC.VerifyRegisterOTP(domain.VerifyRegisterOTPInput{
		Email: req.Email,
		OTP:   req.OTP,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.authUC.Login(domain.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// RefreshToken handles POST /auth/refresh.
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.authUC.Refresh(domain.RefreshInput{RefreshToken: req.RefreshToken})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
