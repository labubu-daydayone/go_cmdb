package releases

import (
	"fmt"

	"go_cmdb/internal/httpx"
	"go_cmdb/internal/release"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler 发布API处理器
type Handler struct {
	service *release.Service
}

// NewHandler 创建发布API处理器
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		service: release.NewService(db),
	}
}

// CreateRelease 创建发布任务
// POST /api/v1/releases
func (h *Handler) CreateRelease(c *gin.Context) {
	var req release.CreateReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
		return
	}

	// 验证target字段
	if req.Target != "cdn" {
		httpx.FailErr(c, httpx.ErrParamInvalid("target must be 'cdn'"))
		return
	}

	resp, err := h.service.CreateRelease(&req)
	if err != nil {
		// 如果是AppError，直接返回；否则包装为内部错误
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.FailErr(c, appErr)
		} else {
			httpx.FailErr(c, httpx.ErrInternalError("failed to create release", err))
		}
		return
	}

	httpx.OK(c, resp)
}

// GetRelease 获取发布任务详情
// GET /api/v1/releases/:id
func (h *Handler) GetRelease(c *gin.Context) {
	// 获取releaseID参数
	releaseID := c.Param("id")
	if releaseID == "" {
		httpx.FailErr(c, httpx.ErrParamInvalid("releaseId is required"))
		return
	}

	// 转换为int64
	var id int64
	if _, err := fmt.Sscanf(releaseID, "%d", &id); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid releaseId"))
		return
	}

	resp, err := h.service.GetRelease(id)
	if err != nil {
		// 如果是AppError，直接返回；否则包装为内部错误
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.FailErr(c, appErr)
		} else {
			httpx.FailErr(c, httpx.ErrInternalError("failed to get release", err))
		}
		return
	}

	httpx.OK(c, resp)
}
