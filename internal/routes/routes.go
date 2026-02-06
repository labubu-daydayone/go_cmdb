package routes

import (
	"github.com/gin-gonic/gin"
	"go_cmdb/internal/handler"
)

func RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		agent := api.Group("/agent")
		{
			agent.GET("/tasks/pull", handler.PullAgentTasks)
			agent.POST("/tasks/update-status", handler.UpdateAgentTaskStatus)
		}
	}
}
