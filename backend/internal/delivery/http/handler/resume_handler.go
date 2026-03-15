package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/delivery/http/middleware"
	"github.com/interview_app/backend/internal/domain"
)

type saveResumeRequest struct {
	Content string `json:"content"`
}

// ResumeHandler handles resume-related APIs.
type ResumeHandler struct {
	interviewUC domain.InterviewUseCase
}

func NewResumeHandler(interviewUC domain.InterviewUseCase) *ResumeHandler {
	return &ResumeHandler{interviewUC: interviewUC}
}

// SaveResume handles POST /api/resume.
func (h *ResumeHandler) SaveResume(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	upload, err := parseResumeUpload(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resume, err := h.interviewUC.SaveResume(userID, upload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resume)
}

// GetLatestResume handles GET /api/resume.
func (h *ResumeHandler) GetLatestResume(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	resume, err := h.interviewUC.GetLatestResume(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if resume == nil || strings.TrimSpace(resume.Content) == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "resume not found"})
		return
	}

	c.JSON(http.StatusOK, resume)
}

// AnalyzeResume handles POST /api/resume/analyze.
func (h *ResumeHandler) AnalyzeResume(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	upload, err := parseResumeUpload(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.interviewUC.AnalyzeResume(userID, upload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DownloadLatestResume handles GET /api/resume/download.
func (h *ResumeHandler) DownloadLatestResume(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	resumeFile, err := h.interviewUC.DownloadLatestResume(userID)
	if err != nil {
		message := strings.ToLower(err.Error())
		switch {
		case strings.Contains(message, "no uploaded cv"):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case strings.Contains(message, "not configured"):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	contentType := strings.TrimSpace(resumeFile.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	fileName := strings.TrimSpace(resumeFile.FileName)
	if fileName == "" {
		fileName = "resume-download"
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, contentType, resumeFile.Data)
}

func parseResumeUpload(c *gin.Context) (domain.ResumeUpload, error) {
	if strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		upload := domain.ResumeUpload{
			Content: strings.TrimSpace(c.PostForm("content")),
		}

		fileHeader, err := c.FormFile("file")
		if err != nil {
			if upload.Content == "" {
				return domain.ResumeUpload{}, err
			}
			return upload, nil
		}

		file, err := fileHeader.Open()
		if err != nil {
			return domain.ResumeUpload{}, err
		}
		defer file.Close()

		fileData, err := io.ReadAll(file)
		if err != nil {
			return domain.ResumeUpload{}, err
		}

		upload.FileName = fileHeader.Filename
		upload.ContentType = fileHeader.Header.Get("Content-Type")
		upload.FileData = fileData

		return upload, nil
	}

	var req saveResumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return domain.ResumeUpload{}, nil
		}
		return domain.ResumeUpload{}, err
	}

	return domain.ResumeUpload{Content: req.Content}, nil
}

func extractUserID(c *gin.Context) (string, bool) {
	userIDValue, exists := c.Get(middleware.UserIDContextKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}

	userID, ok := userIDValue.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user context"})
		return "", false
	}

	return userID, true
}
