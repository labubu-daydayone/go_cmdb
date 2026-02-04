package websites

import (
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

		// 3.2 触发发布（Publisher 内部会判断幂等性）
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

		items = append(items, BatchBindOriginSetItem{
			WebsiteID:   websiteID,
			OriginSetID: req.OriginSetID,
			TaskID:      publishResp.TaskID, // 0 表示幂等跳过，>0 表示创建了新任务
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
