package domains

import (
	"net/http"
	"strconv"

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

// ListDomains handles GET /api/v1/domains
func ListDomains(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	keyword := c.Query("keyword")
	purpose := c.Query("purpose")
	provider := c.Query("provider")
	apiKeyIDStr := c.Query("apiKeyId")
	status := c.Query("status")

	params := domain.ListDomainsParams{
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
		Purpose:  purpose,
		Provider: provider,
		Status:   status,
	}

	// Parse apiKeyId if provided
	if apiKeyIDStr != "" {
		if apiKeyID, err := strconv.ParseInt(apiKeyIDStr, 10, 64); err == nil {
			params.APIKeyID = &apiKeyID
		}
	}

	result, err := domain.ListDomains(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1003,
			"message": "query failed: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}

// EnableCDN handles POST /api/v1/domains/:id/enable-cdn
func EnableCDN(c *gin.Context) {
	// Parse domain ID from URL parameter
	domainIDStr := c.Param("id")
	domainID, err := strconv.Atoi(domainIDStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    2001,
			"message": "invalid domain id",
			"data":    nil,
		})
		return
	}

	// Call service to enable CDN
	updatedDomain, err := domain.EnableCDN(c.Request.Context(), domainID)
	if err != nil {
		// Determine error code based on error message
		code := 3003
		if err.Error() == "domain not found" {
			code = 3001
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    code,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"id":      updatedDomain.ID,
			"domain":  updatedDomain.Domain,
			"purpose": updatedDomain.Purpose,
		},
	})
}

// DisableCDN handles POST /api/v1/domains/:id/disable-cdn
func DisableCDN(c *gin.Context) {
	// Parse domain ID from URL parameter
	domainIDStr := c.Param("id")
	domainID, err := strconv.Atoi(domainIDStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    2001,
			"message": "invalid domain id",
			"data":    nil,
		})
		return
	}

	// Call service to disable CDN
	updatedDomain, err := domain.DisableCDN(c.Request.Context(), domainID)
	if err != nil {
		// Determine error code based on error message
		code := 3003
		if err.Error() == "domain not found" {
			code = 3001
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    code,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"id":      updatedDomain.ID,
			"domain":  updatedDomain.Domain,
			"purpose": updatedDomain.Purpose,
		},
	})
}
