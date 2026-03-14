package handler

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/service/voice"
)

type voiceTTSRequest struct {
	Text string `json:"text" binding:"required"`
}

type VoiceHandler struct {
	service *voice.Service
}

func NewVoiceHandler(service *voice.Service) *VoiceHandler {
	return &VoiceHandler{service: service}
}

// TextToSpeech handles POST /api/voice/tts.
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

// SpeechToText handles POST /api/voice/stt.
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
