package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/losts/syun-eng/backend/internal/service"
)

// StatsHandler handles statistics HTTP requests
type StatsHandler struct {
	statsService *service.StatsService
}

// NewStatsHandler creates a new StatsHandler
func NewStatsHandler(statsService *service.StatsService) *StatsHandler {
	return &StatsHandler{
		statsService: statsService,
	}
}

// GetSummary returns overall statistics
func (h *StatsHandler) GetSummary(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	summary, err := h.statsService.GetSummary(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetWeaknesses returns weak areas
func (h *StatsHandler) GetWeaknesses(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	weaknesses, err := h.statsService.GetWeaknesses(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"weaknesses": weaknesses})
}
