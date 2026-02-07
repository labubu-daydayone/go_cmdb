package cache_rules

import (
	"strconv"

	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler 缓存规则handler
type Handler struct {
	db *gorm.DB
}

// NewHandler 创建handler实例
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// CreateRequest 创建缓存规则组请求
type CreateRequest struct {
	Name    string `json:"name" binding:"required"`
	Enabled bool   `json:"enabled"`
}

// CreateResponse 创建缓存规则组响应
type CreateResponse struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// Create 创建缓存规则组
// POST /api/v1/cache-rules/create
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing("invalid request"))
		return
	}

	// 检查name唯一性
	var exists int64
	if err := h.db.Model(&model.CacheRule{}).Where("name = ?", req.Name).Count(&exists).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to check name uniqueness", err))
		return
	}
	if exists > 0 {
		httpx.FailErr(c, httpx.ErrStateConflict("cache rule name already exists"))
		return
	}
	// 创建 cache_rule
	rule := model.CacheRule{
		Name:    req.Name,
		Enabled: req.Enabled,
	}

	if err := h.db.Create(&rule).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to create cache rule", err))
		return
	}

	// 构造响应
	resp := CreateResponse{
		ID:        rule.ID,
		Name:      rule.Name,
		Enabled:   rule.Enabled,
		CreatedAt: rule.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: rule.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	httpx.OK(c, gin.H{"item": resp})
}

// ListResponse 列表响应
type ListResponse struct {
	Items    []ListItemDTO `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

// ListItemDTO 列表项DTO
type ListItemDTO struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	ItemCount int    `json:"itemCount"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// List 缓存规则组列表
// GET /api/v1/cache-rules
func (h *Handler) List(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "15"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 15
	}

	// 构建查询
	query := h.db.Model(&model.CacheRule{})

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to count cache rules", err))
		return
	}

	// 查询列表
	var rules []model.CacheRule
	offset := (page - 1) * pageSize
	if err := query.
		Order("id DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&rules).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to list cache rules", err))
		return
	}

	// 查询每个规则组的规则项数量
	ruleIDs := make([]int, len(rules))
	for i, rule := range rules {
		ruleIDs[i] = rule.ID
	}

	type CountResult struct {
		CacheRuleID int `json:"cache_rule_id"`
		Count       int `json:"count"`
	}
	var counts []CountResult
	if len(ruleIDs) > 0 {
		if err := h.db.Model(&model.CacheRuleItem{}).
			Select("cache_rule_id, COUNT(*) as count").
			Where("cache_rule_id IN ?", ruleIDs).
			Group("cache_rule_id").
			Find(&counts).Error; err != nil {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to count rule items", err))
			return
		}
	}

	// 构建计数映射
	countMap := make(map[int]int)
	for _, c := range counts {
		countMap[c.CacheRuleID] = c.Count
	}

	// 构造响应
	items := make([]ListItemDTO, len(rules))
	for i, rule := range rules {
		items[i] = ListItemDTO{
			ID:        rule.ID,
			Name:      rule.Name,
			Enabled:   rule.Enabled,
			ItemCount: countMap[rule.ID],
			CreatedAt: rule.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: rule.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	resp := ListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}

	httpx.OK(c, resp)
}

// UpdateRequest 更新缓存规则组请求
type UpdateRequest struct {
	ID      int    `json:"id" binding:"required"`
	Name    string `json:"name" binding:"required"`
	Enabled bool   `json:"enabled"`
}

// Update 更新缓存规则组
// POST /api/v1/cache-rules/update
func (h *Handler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing("invalid request"))
		return
	}

	// 检查规则组是否存在
	var rule model.CacheRule
	if err := h.db.First(&rule, req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("cache rule not found"))
		} else {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to find cache rule", err))
		}
		return
	}

	// 检查name唯一性（排除自己）
	var exists int64
	if err := h.db.Model(&model.CacheRule{}).
		Where("name = ? AND id != ?", req.Name, req.ID).
		Count(&exists).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to check name uniqueness", err))
		return
	}
	if exists > 0 {
		httpx.FailErr(c, httpx.ErrStateConflict("cache rule name already exists"))
		return
	}

	// 更新
	if err := h.db.Model(&rule).Updates(map[string]interface{}{
		"name":    req.Name,
		"enabled": req.Enabled,
	}).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to update cache rule", err))
		return
	}

	httpx.OK(c, nil)
}

// DeleteRequest 删除缓存规则组请求
type DeleteRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}

// Delete 删除缓存规则组
// POST /api/v1/cache-rules/delete
func (h *Handler) Delete(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing("invalid request"))
		return
	}

	// TODO: 检查是否被 Website 使用
	// 当前暂时跳过，等 Website 模块实现后再补充

	// 删除规则组（级联删除规则项）
	if err := h.db.Where("id IN ?", req.IDs).Delete(&model.CacheRule{}).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to delete cache rules", err))
		return
	}

	httpx.OK(c, nil)
}
