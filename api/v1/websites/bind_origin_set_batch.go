package websites

import (
	"crypto/md5"
	"fmt"
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"
	"go_cmdb/internal/upstream"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// BatchBindOriginSetRequest 批量绑定请求
type BatchBindOriginSetRequest struct {
	WebsiteIDs  []int64 `json:"websiteIds" binding:"required,min=1"`
	OriginSetID int64   `json:"originSetId" binding:"required,gt=0"`
}

// BatchBindOriginSetResponse 批量绑定响应
type BatchBindOriginSetResponse struct {
	Items []BatchBindOriginSetItem `json:"items"`
}

// BatchBindOriginSetItem 批量绑定响应项
type BatchBindOriginSetItem struct {
	WebsiteID   int64 `json:"websiteId"`
	OriginSetID int64 `json:"originSetId"`
	TaskID      int64 `json:"taskId"` // 0 表示幂等跳过，>0 表示创建了新任务
}

// BatchBindOriginSet 批量绑定 Origin Set 并触发发布
func (h *Handler) BatchBindOriginSet(c *gin.Context) {
	var req BatchBindOriginSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request body"))
		return
	}

	// 1. 校验所有 website 存在
	var websites []model.Website
	if err := h.db.Where("id IN ?", req.WebsiteIDs).Find(&websites).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query websites", err))
		return
	}

	if len(websites) != len(req.WebsiteIDs) {
		httpx.FailErr(c, httpx.ErrNotFound("some websites not found"))
		return
	}

	// 2. 校验 originSetID 存在且 status=active
	var originSet model.OriginSet
	if err := h.db.Where("id = ? AND status = ?", req.OriginSetID, "active").
		First(&originSet).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("origin set not found or not active"))
		} else {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to query origin set", err))
		}
		return
	}

	// 3. 开始事务
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var items []BatchBindOriginSetItem
	renderer := upstream.NewRenderer(h.db)
	publisher := upstream.NewPublisher(h.db)

	for _, websiteID := range req.WebsiteIDs {
		// 3.1 更新 website 的 origin_set_id
		err := tx.Model(&model.Website{}).
			Where("id = ?", websiteID).
			Updates(map[string]interface{}{
				"origin_set_id": req.OriginSetID,
				"origin_mode":   "group",
			}).Error
		if err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to update website", err))
			return
		}

		// 3.2 渲染 upstream 内容
		renderResp, err := renderer.Render(&upstream.RenderRequest{
			OriginSetID: req.OriginSetID,
			WebsiteID:   websiteID,
		})
		if err != nil {
			tx.Rollback()
			if appErr, ok := err.(*httpx.AppError); ok {
				httpx.FailErr(c, appErr)
			} else {
				httpx.FailErr(c, httpx.ErrInternalError("failed to render upstream", err))
			}
			return
		}

		// 3.3 计算内容 hash（用于幂等判断）
		contentHash := fmt.Sprintf("%x", md5.Sum([]byte(renderResp.UpstreamConf)))

		// 3.4 检查 website 的上次绑定是否为相同的 originSetId，且内容未变化
		// 简化实现：直接对比 website 当前的 origin_set_id
		var currentWebsite model.Website
		if err := tx.First(&currentWebsite, websiteID).Error; err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to query website", err))
			return
		}

		var taskID int64 = 0
		// 如果已经绑定了相同的 originSetId，且内容 hash 一致，则跳过
		// 注意：这里简化实现，只判断 originSetId 是否相同
		if currentWebsite.OriginSetID == int(req.OriginSetID) {
			// 幂等跳过，taskID 保持为 0
			_ = contentHash // 避免未使用变量警告
		} else {
			// 3.5 创建新任务
			publishResp, err := publisher.Publish(&upstream.PublishRequest{
				WebsiteID:   websiteID,
				OriginSetID: req.OriginSetID,
			})
			if err != nil {
				tx.Rollback()
				if appErr, ok := err.(*httpx.AppError); ok {
					httpx.FailErr(c, appErr)
				} else {
					httpx.FailErr(c, httpx.ErrInternalError("failed to publish", err))
				}
				return
			}
			taskID = publishResp.ReleaseID
		}

		items = append(items, BatchBindOriginSetItem{
			WebsiteID:   websiteID,
			OriginSetID: req.OriginSetID,
			TaskID:      taskID,
		})
	}

	// 4. 提交事务
	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to commit transaction", err))
		return
	}

	httpx.OK(c, BatchBindOriginSetResponse{
		Items: items,
	})
}
