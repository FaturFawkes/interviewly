package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/delivery/http/middleware"
	"github.com/interview_app/backend/internal/domain"
	"github.com/interview_app/backend/internal/service/subscription"
)

type startReviewRequest struct {
	SessionType       string `json:"session_type"`
	InputMode         string `json:"input_mode"`
	InterviewLanguage string `json:"interview_language"`
	InputText         string `json:"input_text"`
	VoiceURL          string `json:"voice_url"`
	TranscriptText    string `json:"transcript_text"`
	InterviewPrompt   string `json:"interview_prompt"`
	TargetRole        string `json:"target_role"`
	TargetCompany     string `json:"target_company"`
}

type respondReviewRequest struct {
	SessionID         string `json:"session_id" binding:"required"`
	InterviewLanguage string `json:"interview_language"`
	InputText         string `json:"input_text"`
	VoiceURL          string `json:"voice_url"`
	TranscriptText    string `json:"transcript_text"`
	InterviewPrompt   string `json:"interview_prompt"`
}

type endReviewRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

type ReviewHandler struct {
	interviewUC         domain.InterviewUseCase
	subscriptionService *subscription.Service
}

func NewReviewHandler(interviewUC domain.InterviewUseCase, subscriptionService *subscription.Service) *ReviewHandler {
	return &ReviewHandler{interviewUC: interviewUC, subscriptionService: subscriptionService}
}

func (h *ReviewHandler) StartReview(c *gin.Context) {
	var req startReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	if h.subscriptionService != nil {
		if _, err := h.subscriptionService.CanStartReviewSession(userID, req.InputMode); err != nil {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
			return
		}
	}

	session, err := h.interviewUC.StartReviewSession(userID, domain.ReviewStartInput{
		SessionType:       req.SessionType,
		InputMode:         req.InputMode,
		InterviewLanguage: req.InterviewLanguage,
		InputText:         req.InputText,
		VoiceURL:          req.VoiceURL,
		TranscriptText:    req.TranscriptText,
		InterviewPrompt:   req.InterviewPrompt,
		TargetRole:        req.TargetRole,
		TargetCompany:     req.TargetCompany,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.subscriptionService != nil {
		if _, err := h.subscriptionService.ConsumeReviewSession(userID, session.ID, session.InputMode); err != nil {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"session":          session,
		"feedback":         session.Feedback,
		"score":            session.Feedback.Score,
		"improvement_tips": session.Feedback.Suggestions,
	})
}

func (h *ReviewHandler) RespondReview(c *gin.Context) {
	var req respondReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	if h.subscriptionService != nil && strings.TrimSpace(req.VoiceURL) != "" {
		if _, err := h.subscriptionService.CheckReviewVoiceQuota(userID); err != nil {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
			return
		}
	}

	session, err := h.interviewUC.RespondReviewSession(userID, domain.ReviewRespondInput{
		SessionID:         req.SessionID,
		InterviewLanguage: req.InterviewLanguage,
		InputText:         req.InputText,
		VoiceURL:          req.VoiceURL,
		TranscriptText:    req.TranscriptText,
		InterviewPrompt:   req.InterviewPrompt,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session":          session,
		"feedback":         session.Feedback,
		"score":            session.Feedback.Score,
		"improvement_tips": session.Feedback.Suggestions,
	})
}

func (h *ReviewHandler) EndReview(c *gin.Context) {
	var req endReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	result, err := h.interviewUC.EndReviewSession(userID, req.SessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":       result.SessionID,
		"feedback":         result.Feedback,
		"score":            result.Feedback.Score,
		"improvement_tips": result.ImprovementPlan.PracticePlan,
		"improvement_plan": result.ImprovementPlan,
		"coaching_summary": result.CoachingSummary,
	})
}

func (h *ReviewHandler) GetCoachingSummary(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		return
	}

	result, err := h.interviewUC.GetCoachingSummary(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":       result.SessionID,
		"feedback":         result.Feedback,
		"score":            result.Feedback.Score,
		"improvement_tips": result.ImprovementPlan.PracticePlan,
		"improvement_plan": result.ImprovementPlan,
		"coaching_summary": result.CoachingSummary,
	})
}

func getUserIDFromContext(c *gin.Context) (string, bool) {
	userIDValue, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}

	userID, ok := userIDValue.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return "", false
	}

	return userID, true
}
