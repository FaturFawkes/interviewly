package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/interview_app/backend/internal/domain"
)

// HealthHandler handles HTTP requests related to application health.
type HealthHandler struct {
	healthUC domain.HealthUseCase
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(uc domain.HealthUseCase) *HealthHandler {
	return &HealthHandler{healthUC: uc}
}

// GetHealth godoc
// @Summary  Health check
// @Tags     health
// @Produce  json
// @Success  200 {object} domain.HealthStatus
// @Router   /health [get]
func (h *HealthHandler) GetHealth(c *gin.Context) {
	status, err := h.healthUC.GetHealthStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service unavailable"})
		return
	}
	c.JSON(http.StatusOK, status)
}
