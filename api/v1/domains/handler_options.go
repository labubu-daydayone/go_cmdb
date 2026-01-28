package domains

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"go_cmdb/internal/model"
)

// OptionsHandler handles domain options requests
type OptionsHandler struct {
	db *gorm.DB
}

// NewOptionsHandler creates a new options handler
func NewOptionsHandler(db *gorm.DB) *OptionsHandler {
	return &OptionsHandler{db: db}
}

// GetOptions handles GET /api/v1/domains/options
// Returns list of active CDN domains for dropdown selection
func (h *OptionsHandler) GetOptions(c *gin.Context) {
	var domains []model.Domain
	
	// Query active CDN domains, ordered by ID descending
	if err := h.db.Where("status = ? AND purpose = ?", "active", "cdn").
		Order("id DESC").
		Find(&domains).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1003,
			"message": "failed to query domains: " + err.Error(),
			"data":    nil,
		})
		return
	}
	
	// Convert to DTO
	items := make([]DomainOptionDTO, 0, len(domains))
	for _, d := range domains {
		items = append(items, DomainOptionDTO{
			ID:        int64(d.ID),
			Domain:    d.Domain,
			Status:    string(d.Status),
			Purpose:   string(d.Purpose),
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": items,
		},
	})
}
