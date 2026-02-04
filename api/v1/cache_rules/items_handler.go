package cache_rules

import (
	"strconv"

	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ItemUpsertRequest 规则项批量upsert请求
type ItemUpsertRequest struct {
	CacheRuleID int                 `json:"cacheRuleId" binding:"required"`
	Items       []ItemUpsertInput `json:"items" binding:"required,min=1"`
}

// ItemUpsertInput 规则项输入
type ItemUpsertInput struct {
	MatchType  string `json:"matchType" binding:"required,oneof=path suffix exact"`
	MatchValue string `json:"matchValue" binding:"required"`
	TTLSeconds int    `json:"ttlSeconds" binding:"required,min=0"`
	Enabled    bool   `json:"enabled"`
}

// ItemsUpsert 规则项批量upsert
// POST /api/v1/cache-rules/items/upsert
func (h *Handler) ItemsUpsert(c *gin.Context) {
	var req ItemUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing("invalid request"))
		return
	}

	// 检查cache rule是否存在
	var rule model.CacheRule
	if err := h.db.First(&rule, req.CacheRuleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("cache rule not found"))
		} else {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to find cache rule", err))
		}
		return
	}

	// 开始事务
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除旧规则项
	if err := tx.Where("cache_rule_id = ?", req.CacheRuleID).Delete(&model.CacheRuleItem{}).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to delete old items", err))
		return
	}

	// 创建新规则项 - 使用原生 SQL 确保 enabled 字段被正确设置
	for _, item := range req.Items {
		if err := tx.Exec(
			"INSERT INTO cache_rule_items (cache_rule_id, match_type, match_value, ttl_seconds, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NOW(), NOW())",
			req.CacheRuleID, item.MatchType, item.MatchValue, item.TTLSeconds, item.Enabled,
		).Error; err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to create item", err))
			return
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to commit transaction", err))
		return
	}

	httpx.OK(c, nil)
}

// ItemDTO 规则项DTO
type ItemDTO struct {
	ID         int    `json:"id"`
	MatchType  string `json:"matchType"`
	MatchValue string `json:"matchValue"`
	TTLSeconds int    `json:"ttlSeconds"`
	Enabled    bool   `json:"enabled"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

// GetItems 获取规则组下的规则项列表
// GET /api/v1/cache-rules/:id/items
func (h *Handler) GetItems(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		httpx.FailErr(c, httpx.ErrParamMissing("id is required"))
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing("invalid id"))
		return
	}

	// 检查cache rule是否存在
	var rule model.CacheRule
	if err := h.db.First(&rule, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("cache rule not found"))
		} else {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to find cache rule", err))
		}
		return
	}

	// 查询规则项
	var items []model.CacheRuleItem
	if err := h.db.Where("cache_rule_id = ?", id).Order("id ASC").Find(&items).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to find items", err))
		return
	}

	// 构造响应
	itemDTOs := make([]ItemDTO, len(items))
	for i, item := range items {
		itemDTOs[i] = ItemDTO{
			ID:         item.ID,
			MatchType:  item.MatchType,
			MatchValue: item.MatchValue,
			TTLSeconds: item.TTLSeconds,
			Enabled:    item.Enabled,
			CreatedAt:  item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:  item.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	httpx.OK(c, gin.H{"items": itemDTOs})
}

// DeleteItemsRequest 删除规则项请求
type DeleteItemsRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}

// DeleteItems 删除规则项
// POST /api/v1/cache-rules/items/delete
func (h *Handler) DeleteItems(c *gin.Context) {
	var req DeleteItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing("invalid request"))
		return
	}

	// 删除规则项
	if err := h.db.Where("id IN ?", req.IDs).Delete(&model.CacheRuleItem{}).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to delete items", err))
		return
	}

	httpx.OK(c, nil)
}
