package release_tasks

import (
	"strconv"

	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler release_tasks handler
type Handler struct {
	db *gorm.DB
}

// NewHandler 创建 handler 实例
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// List 查询 release_tasks 列表
// GET /api/v1/release-tasks
func (h *Handler) List(c *gin.Context) {
	targetType := c.Query("targetType")
	targetIDStr := c.Query("targetId")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "20")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := h.db.Model(&model.ReleaseTask{})

	// 过滤条件
	if targetType != "" {
		query = query.Where("target_type = ?", targetType)
	}
	if targetIDStr != "" {
		targetID, err := strconv.ParseInt(targetIDStr, 10, 64)
		if err == nil {
			query = query.Where("target_id = ?", targetID)
		}
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	// 查询列表
	var tasks []model.ReleaseTask
	offset := (page - 1) * pageSize
	if err := query.Order("id DESC").Limit(pageSize).Offset(offset).Find(&tasks).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	// 构建响应
	items := make([]map[string]interface{}, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, map[string]interface{}{
			"id":          task.ID,
			"type":        task.Type,
			"targetType":  task.TargetType,
			"targetId":    task.TargetID,
			"status":      task.Status,
			"contentHash": task.ContentHash,
			"retryCount":  task.RetryCount,
			"createdAt":   task.CreatedAt,
			"updatedAt":   task.UpdatedAt,
		})
	}

	httpx.OK(c, map[string]interface{}{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
