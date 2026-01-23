package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRouter sets up the API v1 routes
func SetupRouter(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", pingHandler)
	}
}

// pingHandler handles the ping request
func pingHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "pong",
	})
}
