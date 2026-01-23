package v1

import (
	"go_cmdb/api/v1/auth"
	"go_cmdb/api/v1/middleware"
	"go_cmdb/api/v1/nodes"
	"go_cmdb/internal/config"
	"go_cmdb/internal/httpx"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter sets up the API v1 routes
func SetupRouter(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
	v1 := r.Group("/api/v1")
	{
		// Public routes (no authentication required)
		v1.GET("/ping", pingHandler)

		// Auth routes
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", auth.LoginHandler(db, cfg))
		}

		// Demo routes for testing error responses
		demo := v1.Group("/demo")
		{
			demo.GET("/error", demoErrorHandler)
			demo.GET("/param", demoParamHandler)
			demo.GET("/notfound", demoNotFoundHandler)
		}

		// Protected routes (authentication required)
		protected := v1.Group("")
		protected.Use(middleware.AuthRequired())
		{
			protected.GET("/me", meHandler)

			// Nodes routes
			nodesHandler := nodes.NewHandler(db)
			nodesGroup := protected.Group("/nodes")
			{
				nodesGroup.GET("", nodesHandler.List)
				nodesGroup.POST("/create", nodesHandler.Create)
				nodesGroup.POST("/update", nodesHandler.Update)
				nodesGroup.POST("/delete", nodesHandler.Delete)

				// Sub IPs routes
				nodesGroup.POST("/sub-ips/add", nodesHandler.AddSubIPs)
				nodesGroup.POST("/sub-ips/delete", nodesHandler.DeleteSubIPs)
				nodesGroup.POST("/sub-ips/toggle", nodesHandler.ToggleSubIP)
			}
		}
	}
}

// pingHandler handles the ping request using unified response
func pingHandler(c *gin.Context) {
	httpx.OK(c, gin.H{
		"pong": true,
	})
}

// meHandler returns current user information
func meHandler(c *gin.Context) {
	uid, _ := c.Get("uid")
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	httpx.OK(c, gin.H{
		"uid":      uid,
		"username": username,
		"role":     role,
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
