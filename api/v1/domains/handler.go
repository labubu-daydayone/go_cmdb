package domains

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"go_cmdb/internal/domain"
)

// SyncDomainsRequest represents the request body for syncing domains
type SyncDomainsRequest struct {
	APIKeyID int `json:"apiKeyId" binding:"required"`
}

// SyncDomainsResponse represents the response for syncing domains
type SyncDomainsResponse struct {
	Total   int `json:"total"`
	Created int `json:"created"`
	Updated int `json:"updated"`
}

// SyncDomains handles POST /api/v1/domains/sync
func SyncDomains(c *gin.Context) {
	var req SyncDomainsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1002,
			"message": "invalid request: " + err.Error(),
			"data":    nil,
		})
		return
	}

	result, err := domain.SyncDomainsByAPIKey(c.Request.Context(), req.APIKeyID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1003,
			"message": "sync failed: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": SyncDomainsResponse{
			Total:   result.Total,
			Created: result.Created,
			Updated: result.Updated,
		},
	})
}
