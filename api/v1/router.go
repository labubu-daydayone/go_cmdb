package v1

import (
	"go_cmdb/api/v1/auth"
	"go_cmdb/api/v1/line_groups"
	"go_cmdb/api/v1/middleware"
	"go_cmdb/api/v1/node_groups"
	"go_cmdb/api/v1/nodes"
	"go_cmdb/api/v1/origin_groups"
	"go_cmdb/api/v1/origins"
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

			// Node groups routes
			nodeGroupsHandler := node_groups.NewHandler(db)
			nodeGroupsGroup := protected.Group("/node-groups")
			{
				nodeGroupsGroup.GET("", nodeGroupsHandler.List)
				nodeGroupsGroup.POST("/create", nodeGroupsHandler.Create)
				nodeGroupsGroup.POST("/update", nodeGroupsHandler.Update)
				nodeGroupsGroup.POST("/delete", nodeGroupsHandler.Delete)
			}

			// Line groups routes
			lineGroupsHandler := line_groups.NewHandler(db)
			lineGroupsGroup := protected.Group("/line-groups")
			{
				lineGroupsGroup.GET("", lineGroupsHandler.List)
				lineGroupsGroup.POST("/create", lineGroupsHandler.Create)
				lineGroupsGroup.POST("/update", lineGroupsHandler.Update)
				lineGroupsGroup.POST("/delete", lineGroupsHandler.Delete)
			}

			// Origin groups routes
			originGroupsHandler := origin_groups.NewHandler(db)
			originGroupsGroup := protected.Group("/origin-groups")
			{
				originGroupsGroup.GET("", originGroupsHandler.List)
				originGroupsGroup.POST("/create", originGroupsHandler.Create)
				originGroupsGroup.POST("/update", originGroupsHandler.Update)
				originGroupsGroup.POST("/delete", originGroupsHandler.Delete)
			}

			// Origins routes (website origin sets)
			originsHandler := origins.NewHandler(db)
			originsGroup := protected.Group("/origins")
			{
				originsGroup.POST("/create-from-group", originsHandler.CreateFromGroup)
				originsGroup.POST("/create-manual", originsHandler.CreateManual)
				originsGroup.POST("/update", originsHandler.Update)
				originsGroup.POST("/delete", originsHandler.Delete)
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
