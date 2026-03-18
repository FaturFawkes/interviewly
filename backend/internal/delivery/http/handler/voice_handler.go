package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/delivery/http/middleware"
	"github.com/interview_app/backend/internal/service/subscription"
	"github.com/interview_app/backend/internal/service/voice"
)

type voiceTTSRequest struct {
	Text string `json:"text" binding:"required"`
}

type voiceAgentSessionRequest struct {
	IncludeConversationID bool `json:"include_conversation_id"`
}

type commitVoiceUsageRequest struct {
	SessionID      string `json:"session_id" binding:"required"`
	ElapsedSeconds int    `json:"elapsed_seconds" binding:"required"`
}

type VoiceHandler struct {
	service             *voice.Service
	subscriptionService *subscription.Service
}

func NewVoiceHandler(service *voice.Service, subscriptionService *subscription.Service) *VoiceHandler {
	return &VoiceHandler{service: service, subscriptionService: subscriptionService}
}

func (h *VoiceHandler) TextToSpeech(c *gin.Context) {
	if h.service == nil || !h.service.IsReady() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "elevenlabs voice service is not configured"})
		return
	}

	var req voiceTTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	text := strings.TrimSpace(req.Text)
	if text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text is required"})
		return
	}

	audio, err := h.service.TextToSpeech(text)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "audio/mpeg", audio)
}

func (h *VoiceHandler) SpeechToText(c *gin.Context) {
	if h.service == nil || !h.service.IsReady() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "elevenlabs voice service is not configured"})
		return
	}

	fileHeader, err := c.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "audio file is required"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to open audio file"})
		return
	}
	defer file.Close()

	audioBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read audio file"})
		return
	}

	result, err := h.service.SpeechToText(audioBytes, fileHeader.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *VoiceHandler) CreateAgentSession(c *gin.Context) {
	if h.service == nil || !h.service.AgentIsReady() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "elevenlabs agent is not configured"})
		return
	}
	if h.subscriptionService == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscription service is not configured"})
		return
	}

	userIDValue, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, ok := userIDValue.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return
	}

	req := voiceAgentSessionRequest{}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	voiceQuota, err := h.subscriptionService.CheckVoiceQuota(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !voiceQuota.CanStart {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": voiceQuota.Message})
		return
	}

	result, err := h.service.GetAgentSignedURL(req.IncludeConversationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"signed_url":                result.SignedURL,
		"conversation_id":           result.ConversationID,
		"total_voice_minutes":       voiceQuota.TotalVoiceMinutes,
		"used_voice_minutes":        voiceQuota.UsedVoiceMinutes,
		"remaining_voice_minutes":   voiceQuota.RemainingVoiceMinutes,
		"allowed_call_seconds":      voiceQuota.AllowedCallSeconds,
		"warning_threshold_reached": voiceQuota.WarningThresholdReached,
		"voice_quota_message":       voiceQuota.Message,
	})
}

func (h *VoiceHandler) CommitVoiceUsage(c *gin.Context) {
	if h.subscriptionService == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscription service is not configured"})
		return
	}

	var req commitVoiceUsageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDValue, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	userID, ok := userIDValue.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return
	}

	state, err := h.subscriptionService.CommitVoiceUsage(userID, req.SessionID, req.ElapsedSeconds)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	remainingMinutes := state.TotalVoiceMinutes - state.UsedVoiceMinutes
	if remainingMinutes < 0 {
		remainingMinutes = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"total_voice_minutes":     state.TotalVoiceMinutes,
		"used_voice_minutes":      state.UsedVoiceMinutes,
		"remaining_voice_minutes": remainingMinutes,
		"period_start":            state.PeriodStart,
		"period_end":              state.PeriodEnd,
	})
}
