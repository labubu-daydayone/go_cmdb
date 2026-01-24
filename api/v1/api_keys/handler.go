package api_keys

import (
	"go_cmdb/internal/api_keys"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// List handles GET /api/v1/api-keys
func List(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	keyword := c.Query("keyword")
	provider := c.DefaultQuery("provider", "cloudflare")
	status := c.DefaultQuery("status", "all")

	// Validate pagination
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Call service
	result, err := api_keys.List(c.Request.Context(), api_keys.ListParams{
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
		Provider: provider,
		Status:   status,
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1003,
			"message": "failed to list API keys: " + err.Error(),
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

// Create handles POST /api/v1/api-keys/create
func Create(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		Provider string `json:"provider" binding:"required"`
		Account  string `json:"account"`
		APIToken string `json:"apiToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1002,
			"message": "invalid request: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Call service
	err := api_keys.Create(c.Request.Context(), api_keys.CreateParams{
		Name:     req.Name,
		Provider: req.Provider,
		Account:  req.Account,
		APIToken: req.APIToken,
	})

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1003,
			"message": "failed to create API key: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}

// Update handles POST /api/v1/api-keys/update
func Update(c *gin.Context) {
	var req struct {
		ID       int64   `json:"id" binding:"required"`
		Name     *string `json:"name"`
		Account  *string `json:"account"`
		APIToken *string `json:"apiToken"`
		Status   *string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1002,
			"message": "invalid request: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Call service
	err := api_keys.Update(c.Request.Context(), api_keys.UpdateParams{
		ID:       req.ID,
		Name:     req.Name,
		Account:  req.Account,
		APIToken: req.APIToken,
		Status:   req.Status,
	})

	if err != nil {
		// Check if it's a dependency error
		if strings.Contains(err.Error(), "is being used by") && strings.Contains(err.Error(), "domains") {
			c.JSON(http.StatusOK, gin.H{
				"code":    3003,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    1003,
			"message": "failed to update API key: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}

// Delete handles POST /api/v1/api-keys/delete
func Delete(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required,min=1"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1002,
			"message": "invalid request: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Call service
	err := api_keys.Delete(c.Request.Context(), api_keys.DeleteParams{
		IDs: req.IDs,
	})

	if err != nil {
		// Check if it's a dependency error
		if strings.Contains(err.Error(), "is being used by") && strings.Contains(err.Error(), "domains") {
			c.JSON(http.StatusOK, gin.H{
				"code":    3003,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    1003,
			"message": "failed to delete API keys: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}

// ToggleStatus handles POST /api/v1/api-keys/toggle-status
func ToggleStatus(c *gin.Context) {
	var req struct {
		ID     int64  `json:"id" binding:"required"`
		Status string `json:"status" binding:"required,oneof=active inactive"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1002,
			"message": "invalid request: " + err.Error(),
			"data":    nil,
		})
		return
	}

	// Call service
	err := api_keys.ToggleStatus(c.Request.Context(), api_keys.ToggleStatusParams{
		ID:     req.ID,
		Status: req.Status,
	})

	if err != nil {
		// Check if it's a dependency error
		if strings.Contains(err.Error(), "is being used by") && strings.Contains(err.Error(), "domains") {
			c.JSON(http.StatusOK, gin.H{
				"code":    3003,
				"message": err.Error(),
				"data":    nil,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    1003,
			"message": "failed to toggle API key status: " + err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	})
}
