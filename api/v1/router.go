package v1

import (
	"go_cmdb/internal/httpx"

	"github.com/gin-gonic/gin"
)

// SetupRouter sets up the API v1 routes
func SetupRouter(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		v1.GET("/ping", pingHandler)

		// Demo routes for testing error responses
		demo := v1.Group("/demo")
		{
			demo.GET("/error", demoErrorHandler)
			demo.GET("/param", demoParamHandler)
			demo.GET("/notfound", demoNotFoundHandler)
		}
	}
}

// pingHandler handles the ping request using unified response
func pingHandler(c *gin.Context) {
	httpx.OK(c, gin.H{
		"pong": true,
	})
}

// demoErrorHandler demonstrates internal error response (500)
func demoErrorHandler(c *gin.Context) {
	httpx.FailErr(c, httpx.ErrInternalError("internal error", nil))
}

// demoParamHandler demonstrates parameter error response (400)
func demoParamHandler(c *gin.Context) {
	x := c.Query("x")
	if x == "" {
		httpx.FailErr(c, httpx.ErrParamMissing("parameter 'x' is required"))
		return
	}

	httpx.OK(c, gin.H{
		"x": x,
	})
}

// demoNotFoundHandler demonstrates not found error response (404)
func demoNotFoundHandler(c *gin.Context) {
	httpx.FailErr(c, httpx.ErrNotFound("resource not found"))
}
