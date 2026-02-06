package website_release_tasks

import (
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"
	"go_cmdb/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler website_release_tasks handler
type Handler struct {
	db *gorm.DB
}

// NewHandler creates a new handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// ListRequest 列表请求
type ListRequest struct {
	WebsiteID *int64 `form:"websiteId"`
	Status    string `form:"status"`
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
}

// ListResponse 列表响应
type ListResponse struct {
	Items    []TaskListItemDTO `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

// TaskListItemDTO 任务列表项 DTO
type TaskListItemDTO struct {
	ID          int64   `json:"id"`
	WebsiteID   int64   `json:"websiteId"`
	Status      string  `json:"status"`
	ContentHash string  `json:"contentHash"`
	LastError   *string `json:"lastError"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

// List 查询任务列表
// GET /api/v1/website-release-tasks
func (h *Handler) List(c *gin.Context) {
	var req ListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request"))
		return
	}

	// 默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 15
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	// 构建查询
	query := h.db.Model(&model.WebsiteReleaseTask{})

	if req.WebsiteID != nil {
		query = query.Where("website_id = ?", *req.WebsiteID)
	}

	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to count", err))
		return
	}

	// 查询列表
	var tasks []model.WebsiteReleaseTask
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("id DESC").Limit(req.PageSize).Offset(offset).Find(&tasks).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query", err))
		return
	}

	// 转换为 DTO
	items := make([]TaskListItemDTO, 0, len(tasks))
	for _, task := range tasks {
		dto := TaskListItemDTO{
			ID:          task.ID,
			WebsiteID:   task.WebsiteID,
			Status:      string(task.Status),
			ContentHash: task.ContentHash,
			CreatedAt:   task.CreatedAt.Format("2006-01-02T15:04:05+08:00"),
			UpdatedAt:   task.UpdatedAt.Format("2006-01-02T15:04:05+08:00"),
		}
		if task.ErrorMessage != nil {
			dto.LastError = task.ErrorMessage
		}
		items = append(items, dto)
	}

	httpx.OK(c, ListResponse{
		Items:    items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

// DetailResponse 详情响应
type DetailResponse struct {
	Item TaskDetailDTO `json:"item"`
}

// TaskDetailDTO 任务详情 DTO
type TaskDetailDTO struct {
	ID          int64   `json:"id"`
	WebsiteID   int64   `json:"websiteId"`
	Status      string  `json:"status"`
	ContentHash string  `json:"contentHash"`
	Payload     string  `json:"payload"`
	LastError   *string `json:"lastError"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

// Detail 查询任务详情
// GET /api/v1/website-release-tasks/:id
func (h *Handler) Detail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid id"))
		return
	}

	var task model.WebsiteReleaseTask
	if err := h.db.First(&task, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("task not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query", err))
		return
	}

	dto := TaskDetailDTO{
		ID:          task.ID,
		WebsiteID:   task.WebsiteID,
		Status:      string(task.Status),
		ContentHash: task.ContentHash,
		Payload:     task.Payload,
		CreatedAt:   task.CreatedAt.Format("2006-01-02T15:04:05+08:00"),
		UpdatedAt:   task.UpdatedAt.Format("2006-01-02T15:04:05+08:00"),
	}
	if task.ErrorMessage != nil {
		dto.LastError = task.ErrorMessage
	}

	httpx.OK(c, DetailResponse{Item: dto})
}

// RetryRequest 重试请求
type RetryRequest struct {
	ID int64 `json:"id" binding:"required"`
}

// Retry 重试任务
// POST /api/v1/website-release-tasks/retry
func (h *Handler) Retry(c *gin.Context) {
	var req RetryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request"))
		return
	}

	svc := service.NewWebsiteReleaseTaskService(h.db)
	if err := svc.RetryTask(req.ID); err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to retry task", err))
		return
	}

	httpx.OK(c, nil)
}
