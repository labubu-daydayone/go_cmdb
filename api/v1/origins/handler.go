package origins

import (
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler 网站回源快照handler
type Handler struct {
	db *gorm.DB
}

// NewHandler 创建handler实例
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// CreateFromGroupRequest 从分组创建回源快照请求
type CreateFromGroupRequest struct {
	WebsiteID     int `json:"website_id" binding:"required"`
	OriginGroupID int `json:"origin_group_id" binding:"required"`
}

// CreateFromGroup 从分组创建回源快照
// POST /api/v1/origins/create-from-group
func (h *Handler) CreateFromGroup(c *gin.Context) {
	var req CreateFromGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(""))
		return
	}

	// 检查website是否存在
	var website model.Website
	if err := h.db.First(&website, req.WebsiteID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound(""))
		} else {
			httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		}
		return
	}

	// 检查origin_group是否存在
	var originGroup model.OriginGroup
	if err := h.db.Preload("Addresses").First(&originGroup, req.OriginGroupID).Error; err != nil {
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

	// 创建origin_set
	originSet := model.OriginSet{
		Source:        model.OriginSetSourceGroup,
		OriginGroupID: int64(req.OriginGroupID),
	}

	if err := tx.Create(&originSet).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	// 拷贝addresses
	if len(originGroup.Addresses) > 0 {
		addresses := make([]model.OriginAddress, len(originGroup.Addresses))
		for i, addr := range originGroup.Addresses {
			addresses[i] = model.OriginAddress{
				OriginSetID: originSet.ID,
				Role:        addr.Role,
				Protocol:    addr.Protocol,
				Address:     addr.Address,
				Weight:      addr.Weight,
				Enabled:     addr.Enabled,
			}
		}

		if err := tx.Create(&addresses).Error; err != nil {
			tx.Rollback()
			httpx.FailErr(c, httpx.ErrDatabaseError("", err))
			return
		}
	}

	// 更新website
	if err := tx.Model(&website).Updates(map[string]interface{}{
		"origin_mode":     model.OriginModeGroup,
		"origin_group_id": req.OriginGroupID,
		"origin_set_id":   originSet.ID,
	}).Error; err != nil {
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
	if err := h.db.Preload("Addresses").First(&originSet, originSet.ID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	httpx.OK(c, originSet)
}

// AddressInput 地址输入
type AddressInput struct {
	Role     string `json:"role" binding:"required,oneof=primary backup"`
	Protocol string `json:"protocol" binding:"required,oneof=http https"`
	Address  string `json:"address" binding:"required"`
	Weight   int    `json:"weight"`
	Enabled  *bool  `json:"enabled"`
}

// CreateManualRequest 手动创建回源快照请求
type CreateManualRequest struct {
	WebsiteID int            `json:"website_id" binding:"required"`
	Addresses []AddressInput `json:"addresses" binding:"required,min=1"`
}

// CreateManual 手动创建回源快照
// POST /api/v1/origins/create-manual
func (h *Handler) CreateManual(c *gin.Context) {
	var req CreateManualRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(""))
		return
	}

	// 检查website是否存在
	var website model.Website
	if err := h.db.First(&website, req.WebsiteID).Error; err != nil {
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

	// 创建origin_set
	originSet := model.OriginSet{
		Source:        model.OriginSetSourceManual,
		OriginGroupID: 0,
	}

	if err := tx.Create(&originSet).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	// 创建addresses
	addresses := make([]model.OriginAddress, len(req.Addresses))
	for i, addr := range req.Addresses {
		enabled := true
		if addr.Enabled != nil {
			enabled = *addr.Enabled
		}
		weight := 10
		if addr.Weight > 0 {
			weight = addr.Weight
		}

		addresses[i] = model.OriginAddress{
			OriginSetID: originSet.ID,
			Role:        addr.Role,
			Protocol:    addr.Protocol,
			Address:     addr.Address,
			Weight:      weight,
			Enabled:     enabled,
		}
	}

	if err := tx.Create(&addresses).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	// 更新website
	if err := tx.Model(&website).Updates(map[string]interface{}{
		"origin_mode":     model.OriginModeManual,
		"origin_group_id": 0,
		"origin_set_id":   originSet.ID,
	}).Error; err != nil {
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
	if err := h.db.Preload("Addresses").First(&originSet, originSet.ID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	httpx.OK(c, originSet)
}

// UpdateRequest 更新回源快照请求
type UpdateRequest struct {
	WebsiteID int            `json:"website_id" binding:"required"`
	Addresses []AddressInput `json:"addresses" binding:"required,min=1"`
}

// Update 更新回源快照（只能更新manual的）
// POST /api/v1/origins/update
func (h *Handler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(""))
		return
	}

	// 查询website
	var website model.Website
	if err := h.db.Preload("OriginSet").First(&website, req.WebsiteID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound(""))
		} else {
			httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		}
		return
	}

	// 检查是否有origin_set
	if !website.OriginSetID.Valid || website.OriginSet == nil {
		httpx.FailErr(c, httpx.ErrNotFound(""))
		return
	}

	// 检查是否是manual模式
	if website.OriginSet.Source != model.OriginSetSourceManual {
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

	// 删除旧addresses
	if err := tx.Where("origin_set_id = ?", website.OriginSetID.Int32).
		Delete(&model.OriginAddress{}).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	// 创建新addresses
	addresses := make([]model.OriginAddress, len(req.Addresses))
	for i, addr := range req.Addresses {
		enabled := true
		if addr.Enabled != nil {
			enabled = *addr.Enabled
		}
		weight := 10
		if addr.Weight > 0 {
			weight = addr.Weight
		}

		addresses[i] = model.OriginAddress{
			OriginSetID: int(website.OriginSetID.Int32),
			Role:        addr.Role,
			Protocol:    addr.Protocol,
			Address:     addr.Address,
			Weight:      weight,
			Enabled:     enabled,
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
	var originSet model.OriginSet
	if err := h.db.Preload("Addresses").First(&originSet, website.OriginSetID.Int32).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	httpx.OK(c, originSet)
}

// DeleteRequest 删除回源快照请求
type DeleteRequest struct {
	WebsiteID int `json:"website_id" binding:"required"`
}

// Delete 删除回源快照
// POST /api/v1/origins/delete
func (h *Handler) Delete(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing(""))
		return
	}

	// 查询website
	var website model.Website
	if err := h.db.First(&website, req.WebsiteID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound(""))
		} else {
			httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		}
		return
	}

	// 检查是否有origin_set
	if !website.OriginSetID.Valid {
		httpx.OK(c, gin.H{"message": "no origin_set to delete"})
		return
	}

	// 开始事务
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除origin_set（级联删除addresses）
	if err := tx.Delete(&model.OriginSet{}, website.OriginSetID.Int32).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	// 更新website（设置为redirect模式）
	if err := tx.Model(&website).Updates(map[string]interface{}{
		"origin_mode":     model.OriginModeRedirect,
		"origin_group_id": nil,
		"origin_set_id":   nil,
	}).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("", err))
		return
	}

	httpx.OK(c, gin.H{"message": "origin_set deleted, website set to redirect mode"})
}
