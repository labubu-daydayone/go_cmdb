package origin_groups

import (
	"strconv"

	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler 回源分组handler
type Handler struct {
	db *gorm.DB
}

// NewHandler 创建handler实例
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// List 回源分组列表
// GET /api/v1/origin-groups?page=1&pageSize=15&name=xxx&status=active
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

	// 解析筛选参数
	name := c.Query("name")
	status := c.Query("status")

	// 构建查询
	query := h.db.Model(&model.OriginGroup{})

	// 筛选条件
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	// 查询列表
	var groups []model.OriginGroup
	offset := (page - 1) * pageSize
	if err := query.
		Preload("Addresses").
		Order("id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&groups).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	// 计算addressCount
	type GroupWithCount struct {
		model.OriginGroup
		AddressCount int `json:"address_count"`
	}

	result := make([]GroupWithCount, len(groups))
	for i, g := range groups {
		result[i] = GroupWithCount{
			OriginGroup:  g,
			AddressCount: len(g.Addresses),
		}
	}

	httpx.OK(c, gin.H{
		"items":    result,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// CreateRequest 创建回源分组请求
type CreateRequest struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Addresses   []AddressInput  `json:"addresses" binding:"required,min=1"`
}

// AddressInput 地址输入
type AddressInput struct {
	Role     string `json:"role" binding:"required,oneof=primary backup"`
	Protocol string `json:"protocol" binding:"required,oneof=http https"`
	Address  string `json:"address" binding:"required"`
	Weight   int    `json:"weight"`
	Enabled  *bool  `json:"enabled"`
}

// Create 创建回源分组
// POST /api/v1/origin-groups/create
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(""))
		return
	}

	// 检查name唯一性
	var exists int64
	if err := h.db.Model(&model.OriginGroup{}).Where("name = ?", req.Name).Count(&exists).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}
	if exists > 0 {
		httpx.FailErr(c, httpx.ErrStateConflict(""))
		return
	}

	// 开始事务
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建origin_group
	group := model.OriginGroup{
		Name:        req.Name,
		Description: req.Description,
		Status:      model.OriginGroupStatusActive,
	}

	if err := tx.Create(&group).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	// 创建addresses
	addresses := make([]model.OriginGroupAddress, len(req.Addresses))
	for i, addr := range req.Addresses {
		enabled := true
		if addr.Enabled != nil {
			enabled = *addr.Enabled
		}
		weight := 10
		if addr.Weight > 0 {
			weight = addr.Weight
		}

		addresses[i] = model.OriginGroupAddress{
			OriginGroupID: group.ID,
			Role:          addr.Role,
			Protocol:      addr.Protocol,
			Address:       addr.Address,
			Weight:        weight,
			Enabled:       enabled,
		}
	}

	if err := tx.Create(&addresses).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	// 重新加载完整数据
	if err := h.db.Preload("Addresses").First(&group, group.ID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	httpx.OK(c, group)
}

// UpdateRequest 更新回源分组请求
type UpdateRequest struct {
	ID          int             `json:"id" binding:"required"`
	Name        *string         `json:"name"`
	Description *string         `json:"description"`
	Status      *string         `json:"status"`
	Addresses   []AddressInput  `json:"addresses"` // 如果传入则全量覆盖
}

// Update 更新回源分组
// POST /api/v1/origin-groups/update
func (h *Handler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(""))
		return
	}

	// 查询group是否存在
	var group model.OriginGroup
	if err := h.db.First(&group, req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound(""))
		} else {
			httpx.FailErr(c, httpx.ErrDatabaseError("", err))
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

	// 更新基本字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		// 检查name唯一性
		var exists int64
		if err := tx.Model(&model.OriginGroup{}).
			Where("name = ? AND id != ?", *req.Name, req.ID).
			Count(&exists).Error; err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("", err))
			return
		}
		if exists > 0 {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrStateConflict(""))
			return
		}
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if len(updates) > 0 {
		if err := tx.Model(&group).Updates(updates).Error; err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("", err))
			return
		}
	}

	// 如果传入addresses，全量覆盖
	if req.Addresses != nil {
		// 删除旧addresses
		if err := tx.Where("origin_group_id = ?", req.ID).Delete(&model.OriginGroupAddress{}).Error; err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("", err))
			return
		}

		// 创建新addresses
		if len(req.Addresses) > 0 {
			addresses := make([]model.OriginGroupAddress, len(req.Addresses))
			for i, addr := range req.Addresses {
				enabled := true
				if addr.Enabled != nil {
					enabled = *addr.Enabled
				}
				weight := 10
				if addr.Weight > 0 {
					weight = addr.Weight
				}

				addresses[i] = model.OriginGroupAddress{
					OriginGroupID: req.ID,
					Role:          addr.Role,
					Protocol:      addr.Protocol,
					Address:       addr.Address,
					Weight:        weight,
					Enabled:       enabled,
				}
			}

			if err := tx.Create(&addresses).Error; err != nil {
				tx.Rollback()
				httpx.FailErr(c, httpx.ErrDatabaseError("", err))
				return
			}
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	// 重新加载完整数据
	if err := h.db.Preload("Addresses").First(&group, req.ID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	httpx.OK(c, group)
}

// DeleteRequest 删除回源分组请求
type DeleteRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}

// Delete 删除回源分组
// POST /api/v1/origin-groups/delete
func (h *Handler) Delete(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(""))
		return
	}

	// 检查是否被website使用
	var usedCount int64
	if err := h.db.Model(&model.Website{}).
		Where("origin_group_id IN ?", req.IDs).
		Count(&usedCount).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	if usedCount > 0 {
		httpx.FailErr(c, httpx.ErrStateConflict(""))
		return
	}

	// 删除（级联删除addresses）
	if err := h.db.Where("id IN ?", req.IDs).Delete(&model.OriginGroup{}).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	httpx.OK(c, gin.H{"deleted": len(req.IDs)})
}
