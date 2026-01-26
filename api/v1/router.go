package v1

import (
	"go_cmdb/api/v1/acme"
	"go_cmdb/api/v1/agent_identities"
	"go_cmdb/api/v1/agent_tasks"
	"go_cmdb/api/v1/api_keys"
	"go_cmdb/api/v1/auth"
	"go_cmdb/api/v1/cert"
	"go_cmdb/api/v1/certificate_renew"
	configHandler "go_cmdb/api/v1/config"
	dnsHandler "go_cmdb/api/v1/dns"
	"go_cmdb/api/v1/domains"
	"go_cmdb/api/v1/line_groups"
	"go_cmdb/api/v1/middleware"
	"go_cmdb/api/v1/node_groups"
	"go_cmdb/api/v1/nodes"
	"go_cmdb/api/v1/origin_groups"
	"go_cmdb/api/v1/origins"
	"go_cmdb/api/v1/releases"
	"go_cmdb/api/v1/risks"
	"go_cmdb/api/v1/websites"
	"go_cmdb/internal/config"
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

	// SetupRouter sets up the API v1 routes
func SetupRouter(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
	// Mount Socket.IO server with JWT authentication
	// Socket.IO will be available at /socket.io/ (default path)
	if ws.Server != nil {
		r.Any("/socket.io/*any", gin.WrapH(ws.WrapWithAuth(ws.Server)))
	}
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

			// Websites routes
			websitesHandler := websites.NewHandler(db)
			websitesGroup := protected.Group("/websites")
			{
				websitesGroup.GET("", websitesHandler.List)
				websitesGroup.GET("/:id", websitesHandler.GetByID)
				websitesGroup.POST("/create", websitesHandler.Create)
				websitesGroup.POST("/update", websitesHandler.Update)
				websitesGroup.POST("/delete", websitesHandler.Delete)
			}

			// Agent tasks routes
			agentTasksHandler := agent_tasks.NewHandler(db, cfg)
			agentTasksGroup := protected.Group("/agent-tasks")
			{
				agentTasksGroup.GET("", agentTasksHandler.List)
				agentTasksGroup.GET("/:id", agentTasksHandler.GetByID)
				agentTasksGroup.POST("/create", agentTasksHandler.Create)
				agentTasksGroup.POST("/retry", agentTasksHandler.Retry)
			}

			// Agent identities routes (admin only)
			agentIdentitiesHandler := agent_identities.NewHandler(db)
			agentIdentitiesGroup := protected.Group("/agent-identities")
			agentIdentitiesGroup.Use(middleware.AdminRequired())
			{
				agentIdentitiesGroup.GET("", agentIdentitiesHandler.List)
				agentIdentitiesGroup.POST("/create", agentIdentitiesHandler.Create)
				agentIdentitiesGroup.POST("/revoke", agentIdentitiesHandler.Revoke)
			}

				// Config routes
				configHandlerInstance := configHandler.NewHandler(db, cfg)
				configGroup := protected.Group("/config")
				{
					configGroup.POST("/apply", configHandlerInstance.Apply)
					configGroup.GET("/versions", configHandlerInstance.ListVersions)
					configGroup.GET("/versions/:version", configHandlerInstance.GetVersion)
				}

				// DNS routes
				dnsHandlerInstance := dnsHandler.NewHandler(db)
				dnsGroup := protected.Group("/dns")
				{
					dnsGroup.GET("/records", dnsHandlerInstance.ListRecords)
					dnsGroup.GET("/records/:id", dnsHandlerInstance.GetRecord)
					dnsGroup.POST("/records/create", dnsHandlerInstance.CreateRecord)
					dnsGroup.POST("/records/delete", dnsHandlerInstance.DeleteRecord)
					dnsGroup.POST("/records/retry", dnsHandlerInstance.RetryRecord)
					dnsGroup.POST("/records/sync", dnsHandlerInstance.SyncRecords)
				}

				// ACME routes
				acmeHandlerInstance := acme.NewHandler(db)
				acmeGroup := protected.Group("/acme")
				{
					// Provider routes
					acmeGroup.GET("/providers", acmeHandlerInstance.ListProviders)
					// Account routes
					acmeGroup.GET("/accounts", acmeHandlerInstance.ListAccounts)
					acmeGroup.GET("/accounts/:id", acmeHandlerInstance.GetAccount)
					acmeGroup.POST("/accounts/enable", acmeHandlerInstance.EnableAccount)
					acmeGroup.POST("/accounts/disable", acmeHandlerInstance.DisableAccount)
					acmeGroup.POST("/accounts/delete", acmeHandlerInstance.DeleteAccount)
					acmeGroup.GET("/accounts/defaults", acmeHandlerInstance.ListDefaults)
					acmeGroup.POST("/accounts/set-default", acmeHandlerInstance.SetDefault)
					// Legacy routes
					acmeGroup.POST("/account/create", acmeHandlerInstance.CreateAccount)
					acmeGroup.POST("/certificate/request", acmeHandlerInstance.RequestCertificate)
					acmeGroup.POST("/certificate/retry", acmeHandlerInstance.RetryRequest)
					acmeGroup.GET("/certificate/requests", acmeHandlerInstance.ListRequests)
					acmeGroup.GET("/certificate/requests/:id", acmeHandlerInstance.GetRequest)
					// T2-22: Delete failed certificate request (requires certHandlerInstance, so defined here)
				}

				// Certificate renewal routes
				certificateRenewHandlerInstance := certificate_renew.NewHandler(db)
				certificateRenewGroup := protected.Group("/certificates/renewal")
				{
					certificateRenewGroup.GET("/candidates", certificateRenewHandlerInstance.GetRenewalCandidates)
					certificateRenewGroup.POST("/trigger", certificateRenewHandlerInstance.TriggerRenewal)
					certificateRenewGroup.POST("/disable-auto", certificateRenewHandlerInstance.DisableAutoRenew)
				}

					// Certificate routes (T2-07, T2-18, T2-19)
					certHandlerInstance := cert.NewHandler(db)
					// T2-22: Delete failed certificate request (add to acmeGroup after certHandlerInstance is defined)
					acmeGroup.POST("/certificate/requests/:requestId/delete", certHandlerInstance.DeleteFailedCertificateRequest)
					// Certificate resource APIs (T2-18, T2-19)
					protected.GET("/certificates", certHandlerInstance.ListCertificatesLifecycle) // T2-19: Unified lifecycle view
					protected.GET("/certificates/:id", certHandlerInstance.GetCertificate)
					protected.POST("/certificates/upload", certHandlerInstance.UploadCertificate)
					// Certificate coverage routes (T2-07)
					protected.GET("/certificates/:id/websites", certHandlerInstance.GetCertificateWebsites)
					protected.GET("/websites/:id/certificates/candidates", certHandlerInstance.GetWebsiteCertificateCandidates)

					// Risk routes (T2-08)
					risksHandlerInstance := risks.NewHandler(db)
					risksGroup := protected.Group("/risks")
					{
						risksGroup.GET("", risksHandlerInstance.ListRisks)
						risksGroup.POST("/:id/resolve", risksHandlerInstance.ResolveRisk)
					}
					protected.GET("/websites/:id/risks", risksHandlerInstance.ListWebsiteRisks)
				protected.GET("/certificates/:id/risks", risksHandlerInstance.ListCertificateRisks)
				protected.POST("/websites/:id/precheck/https", risksHandlerInstance.PrecheckHTTPS)

				// Release routes (B0-01-02, B0-01-03, B0-01-04)
				releasesHandlerInstance := releases.NewHandler(db)
				protected.GET("/releases", releasesHandlerInstance.ListReleases)
				protected.POST("/releases", releasesHandlerInstance.CreateRelease)
				protected.GET("/releases/:id", releasesHandlerInstance.GetRelease)

				// Domain routes (T2-10-02, T2-10-03, T2-10-04)
				domainsGroup := protected.Group("/domains")
				{
					domainsGroup.GET("", domains.ListDomains)
					domainsGroup.POST("/sync", domains.SyncDomains)
					domainsGroup.POST("/:id/enable-cdn", domains.EnableCDN)
					domainsGroup.POST("/:id/disable-cdn", domains.DisableCDN)
				}

				// API Keys routes (T2-10-00)
				apiKeysGroup := protected.Group("/api-keys")
				{
					apiKeysGroup.GET("", api_keys.List)
					apiKeysGroup.POST("/create", api_keys.Create)
					apiKeysGroup.POST("/update", api_keys.Update)
					apiKeysGroup.POST("/delete", api_keys.Delete)
					apiKeysGroup.POST("/toggle-status", api_keys.ToggleStatus)
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
