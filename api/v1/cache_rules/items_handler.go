package cache_rules

import (
	"strconv"

	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"
	"go_cmdb/internal/validator"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ItemCreateRequest 创建规则项请求
type ItemCreateRequest struct {
	CacheRuleID int    `json:"cacheRuleId" binding:"required"`
	MatchType   string `json:"matchType" binding:"required,oneof=path suffix exact"`
	MatchValue  string `json:"matchValue" binding:"required"`
	Mode        string `json:"mode" binding:"required,oneof=default follow force bypass"`
	TTLSeconds  int    `json:"ttlSeconds"`
	Enabled     bool   `json:"enabled"`
}

// ItemCreate 创建规则项
// POST /api/v1/cache-rules/items/create
func (h *Handler) ItemCreate(c *gin.Context) {
	var req ItemCreateRequest
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

	// 规范化 matchValue
	validatorInst := validator.NewCacheRuleItemValidator()
	req.MatchValue = validatorInst.Normalize(req.MatchType, req.MatchValue)

	// 校验规则项
	if err := validatorInst.Validate(req.MatchType, req.MatchValue, req.TTLSeconds); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
		return
	}

	// 检查唯一性：同一 cache_rule_id 下不能有重复的 (match_type, match_value)
	var existingItem model.CacheRuleItem
	err := h.db.Where("cache_rule_id = ? AND match_type = ? AND match_value = ?",
		req.CacheRuleID, req.MatchType, req.MatchValue).First(&existingItem).Error
	if err == nil {
		// 找到了重复的规则项
		httpx.FailErr(c, httpx.ErrParamInvalid("重复的缓存规则项"))
		return
	} else if err != gorm.ErrRecordNotFound {
		// 数据库错误
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to check uniqueness", err))
		return
	}

	// 创建规则项
	item := model.CacheRuleItem{
		CacheRuleID: req.CacheRuleID,
		MatchType:   req.MatchType,
		MatchValue:  req.MatchValue,
		Mode:        req.Mode,
		TTLSeconds:  req.TTLSeconds,
		Enabled:     req.Enabled,
	}

	if err := h.db.Create(&item).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to create item", err))
		return
	}

	httpx.OK(c, nil)
}

// ItemUpsertRequest 规则项批量upsert请求
type ItemUpsertRequest struct {
	CacheRuleID int               `json:"cacheRuleId" binding:"required"`
	Items       []ItemUpsertInput `json:"items" binding:"required,min=1"`
}

// ItemUpsertInput 规则项输入
type ItemUpsertInput struct {
	MatchType  string `json:"matchType" binding:"required,oneof=path suffix exact"`
	MatchValue string `json:"matchValue" binding:"required"`
	Mode       string `json:"mode" binding:"required,oneof=default follow force bypass"`
	TTLSeconds int    `json:"ttlSeconds"`
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

	// 校验每个规则项
	validatorInst := validator.NewCacheRuleItemValidator()
	seenItems := make(map[string]bool) // 用于检查批量内部的重复

	for i, item := range req.Items {
		// 规范化 matchValue
		item.MatchValue = validatorInst.Normalize(item.MatchType, item.MatchValue)
		req.Items[i].MatchValue = item.MatchValue

		// 校验规则项
		if err := validatorInst.Validate(item.MatchType, item.MatchValue, item.TTLSeconds); err != nil {
			httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
			return
		}

		// 检查批量内部的唯一性
		key := item.MatchType + ":" + item.MatchValue
		if seenItems[key] {
			httpx.FailErr(c, httpx.ErrParamInvalid("重复的缓存规则项"))
			return
		}
		seenItems[key] = true
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

	// 创建新规则项
	for _, item := range req.Items {
		newItem := model.CacheRuleItem{
			CacheRuleID: req.CacheRuleID,
			MatchType:   item.MatchType,
			MatchValue:  item.MatchValue,
			Mode:        item.Mode,
			TTLSeconds:  item.TTLSeconds,
			Enabled:     item.Enabled,
		}
		if err := tx.Create(&newItem).Error; err != nil {
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
	Mode       string `json:"mode"`
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
			Mode:       item.Mode,
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
