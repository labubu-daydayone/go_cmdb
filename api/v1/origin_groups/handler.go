package origin_groups

import (
	"fmt"
	"log"
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

// CreateRequest 创建回源分组请求
type CreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// CreateResponse 创建回源分组响应
type CreateResponse struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// Create 创建回源分组
// POST /api/v1/origin-groups/create
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing("invalid request"))
		return
	}

	// 检查name唯一性
	var exists int64
	if err := h.db.Model(&model.OriginGroup{}).Where("name = ?", req.Name).Count(&exists).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to check name uniqueness", err))
		return
	}
	if exists > 0 {
		httpx.FailErr(c, httpx.ErrStateConflict("origin group name already exists"))
		return
	}

	// 创建origin_group
	group := model.OriginGroup{
		Name:        req.Name,
		Description: req.Description,
		Status:      model.OriginGroupStatusActive,
	}

	if err := h.db.Create(&group).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to create origin group", err))
		return
	}

	// 构造响应
	resp := CreateResponse{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
		Status:      group.Status,
		CreatedAt:   group.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   group.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// 触发自动下发
	go h.triggerAutoDispatch(group.ID)

	httpx.OK(c, gin.H{"item": resp})
}

// AddressUpsertRequest 地址批量upsert请求
type AddressUpsertRequest struct {
	OriginGroupID int                  `json:"originGroupId" binding:"required"`
	Items         []AddressUpsertInput `json:"items" binding:"required,min=1"`
}

// AddressUpsertInput 地址输入
type AddressUpsertInput struct {
	Address string `json:"address" binding:"required"`
	Role    string `json:"role" binding:"required,oneof=primary backup"`
	Weight  int    `json:"weight"`
	Enabled bool   `json:"enabled"`
}

// AddressUpsertResponse 地址upsert响应
type AddressUpsertResponse struct {
	OriginGroupID int                    `json:"originGroupId"`
	Items         []AddressUpsertItemDTO `json:"items"`
}

// AddressUpsertItemDTO 地址DTO
type AddressUpsertItemDTO struct {
	ID        int    `json:"id"`
	Address   string `json:"address"`
	Role      string `json:"role"`
	Weight    int    `json:"weight"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// AddressesUpsert 地址批量upsert
// POST /api/v1/origin-groups/addresses/upsert
func (h *Handler) AddressesUpsert(c *gin.Context) {
	var req AddressUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing("invalid request"))
		return
	}

	// 校验role和weight
	for _, item := range req.Items {
		if item.Role == model.OriginRolePrimary && item.Weight < 1 {
			httpx.FailErr(c, httpx.ErrParamMissing("primary role requires weight >= 1"))
			return
		}
	}

	// 检查origin group是否存在
	var group model.OriginGroup
	if err := h.db.First(&group, req.OriginGroupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("origin group not found"))
		} else {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to find origin group", err))
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

	// 删除旧地址
	if err := tx.Where("origin_group_id = ?", req.OriginGroupID).Delete(&model.OriginGroupAddress{}).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to delete old addresses", err))
		return
	}

	// 创建新地址
	addresses := make([]model.OriginGroupAddress, len(req.Items))
	for i, item := range req.Items {
		addresses[i] = model.OriginGroupAddress{
			OriginGroupID: req.OriginGroupID,
			Address:       item.Address,
			Role:          item.Role,
			Weight:        item.Weight,
			Enabled:       item.Enabled,
			Protocol:      model.OriginProtocolHTTP, // 默认http
		}
	}

	// 使用 Select 显式指定所有字段，避免 bool 零值被忽略
	if err := tx.Select("OriginGroupID", "Address", "Role", "Weight", "Enabled", "Protocol").Create(&addresses).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to create addresses", err))
		return
	}

	// 校验至少存在1个enabled=true的primary
	var enabledPrimaryCount int64
	if err := tx.Model(&model.OriginGroupAddress{}).
		Where("origin_group_id = ? AND role = ? AND enabled = ?", req.OriginGroupID, model.OriginRolePrimary, true).
		Count(&enabledPrimaryCount).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to count enabled primary addresses", err))
		return
	}

	if enabledPrimaryCount == 0 {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrStateConflict("no enabled primary"))
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to commit transaction", err))
		return
	}

	// 重新加载地址
	var savedAddresses []model.OriginGroupAddress
	if err := h.db.Where("origin_group_id = ?", req.OriginGroupID).Find(&savedAddresses).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to reload addresses", err))
		return
	}

	// 构造响应
	items := make([]AddressUpsertItemDTO, len(savedAddresses))
	for i, addr := range savedAddresses {
		items[i] = AddressUpsertItemDTO{
			ID:        addr.ID,
			Address:   addr.Address,
			Role:      addr.Role,
			Weight:    addr.Weight,
			Enabled:   addr.Enabled,
			CreatedAt: addr.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: addr.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	resp := AddressUpsertResponse{
		OriginGroupID: req.OriginGroupID,
		Items:         items,
	}

	// 触发自动下发
	go h.triggerAutoDispatch(req.OriginGroupID)

	httpx.OK(c, gin.H{"item": resp})
}

// DetailResponse 详情响应
type DetailResponse struct {
	ID          int                      `json:"id"`
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Status      string                   `json:"status"`
	CreatedAt   string                   `json:"createdAt"`
	UpdatedAt   string                   `json:"updatedAt"`
	Addresses   AddressesListResponse    `json:"addresses"`
}

// AddressesListResponse 地址列表响应
type AddressesListResponse struct {
	Items []AddressUpsertItemDTO `json:"items"`
}

// Detail 回源分组详情
// GET /api/v1/origin-groups/detail?id=12
func (h *Handler) Detail(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		httpx.FailErr(c, httpx.ErrParamMissing("id is required"))
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing("invalid id"))
		return
	}

	// 查询分组
	var group model.OriginGroup
	if err := h.db.First(&group, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("origin group not found"))
		} else {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to find origin group", err))
		}
		return
	}

	// 查询地址
	var addresses []model.OriginGroupAddress
	if err := h.db.Where("origin_group_id = ?", id).Find(&addresses).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to find addresses", err))
		return
	}

	// 构造响应
	items := make([]AddressUpsertItemDTO, len(addresses))
	for i, addr := range addresses {
		items[i] = AddressUpsertItemDTO{
			ID:        addr.ID,
			Address:   addr.Address,
			Role:      addr.Role,
			Weight:    addr.Weight,
			Enabled:   addr.Enabled,
			CreatedAt: addr.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: addr.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	resp := DetailResponse{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
		Status:      group.Status,
		CreatedAt:   group.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   group.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Addresses: AddressesListResponse{
			Items: items,
		},
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
	ID                  int    `json:"id"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	Status              string `json:"status"`
	PrimaryCount        int    `json:"primaryCount"`
	BackupCount         int    `json:"backupCount"`
	EnabledPrimaryCount int    `json:"enabledPrimaryCount"`
	CreatedAt           string `json:"createdAt"`
	UpdatedAt           string `json:"updatedAt"`
}

// List 回源分组列表
// GET /api/v1/origin-groups/list?page=1&pageSize=20
func (h *Handler) List(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 构建查询
	query := h.db.Model(&model.OriginGroup{})

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to count origin groups", err))
		return
	}

	// 查询列表
	var groups []model.OriginGroup
	offset := (page - 1) * pageSize
	if err := query.
		Order("id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&groups).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to find origin groups", err))
		return
	}

	// 构造响应
	items := make([]ListItemDTO, len(groups))
	for i, group := range groups {
		// 统计地址数量
		var primaryCount, backupCount, enabledPrimaryCount int64
		h.db.Model(&model.OriginGroupAddress{}).
			Where("origin_group_id = ? AND role = ?", group.ID, model.OriginRolePrimary).
			Count(&primaryCount)
		h.db.Model(&model.OriginGroupAddress{}).
			Where("origin_group_id = ? AND role = ?", group.ID, model.OriginRoleBackup).
			Count(&backupCount)
		h.db.Model(&model.OriginGroupAddress{}).
			Where("origin_group_id = ? AND role = ? AND enabled = ?", group.ID, model.OriginRolePrimary, true).
			Count(&enabledPrimaryCount)

		items[i] = ListItemDTO{
			ID:                  group.ID,
			Name:                group.Name,
			Description:         group.Description,
			Status:              group.Status,
			PrimaryCount:        int(primaryCount),
			BackupCount:         int(backupCount),
			EnabledPrimaryCount: int(enabledPrimaryCount),
			CreatedAt:           group.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:           group.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
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

// triggerAutoDispatch 触发自动下发
func (h *Handler) triggerAutoDispatch(originGroupID int) {
	log.Printf("[Origin Group] Triggering auto dispatch for origin group %d\n", originGroupID)

	// 查询所有enabled=true且agentStatus=online的节点
	var nodes []model.Node
	if err := h.db.Where("enabled = ? AND status = ?", true, "online").Find(&nodes).Error; err != nil {
		log.Printf("[Origin Group] Failed to find online nodes: %v\n", err)
		return
	}

	if len(nodes) == 0 {
		log.Printf("[Origin Group] No online nodes found, skipping dispatch\n")
		return
	}

	// 查询origin group和addresses
	var group model.OriginGroup
	if err := h.db.First(&group, originGroupID).Error; err != nil {
		log.Printf("[Origin Group] Failed to find origin group %d: %v\n", originGroupID, err)
		return
	}

	var addresses []model.OriginGroupAddress
	if err := h.db.Where("origin_group_id = ? AND enabled = ?", originGroupID, true).Find(&addresses).Error; err != nil {
		log.Printf("[Origin Group] Failed to find addresses for origin group %d: %v\n", originGroupID, err)
		return
	}

	// 构造下发payload
	type PrimaryItem struct {
		Address string `json:"address"`
		Weight  int    `json:"weight"`
	}
	type BackupItem struct {
		Address string `json:"address"`
		Weight  int    `json:"weight"`
	}
	type DispatchPayload struct {
		UpstreamName  string        `json:"upstreamName"`
		OriginGroupID int           `json:"originGroupId"`
		Primaries     []PrimaryItem `json:"primaries"`
		Backups       []BackupItem  `json:"backups"`
		RenderVersion int           `json:"renderVersion"`
	}

	primaries := []PrimaryItem{}
	backups := []BackupItem{}

	for _, addr := range addresses {
		if addr.Role == model.OriginRolePrimary {
			primaries = append(primaries, PrimaryItem{
				Address: addr.Address,
				Weight:  addr.Weight,
			})
		} else if addr.Role == model.OriginRoleBackup {
			backups = append(backups, BackupItem{
				Address: addr.Address,
				Weight:  addr.Weight,
			})
		}
	}

	payload := DispatchPayload{
		UpstreamName:  fmt.Sprintf("og-%d", originGroupID),
		OriginGroupID: originGroupID,
		Primaries:     primaries,
		Backups:       backups,
		RenderVersion: 1,
	}

	log.Printf("[Origin Group] Dispatch payload: upstreamName=%s, primaries=%d, backups=%d, nodes=%d\n",
		payload.UpstreamName, len(primaries), len(backups), len(nodes))

	// TODO: 调用既有发布/下发执行器或agent apply_config调用链
	// 本任务暂时只打印日志，实际下发逻辑需要集成B0-01发布体系
	for _, node := range nodes {
		log.Printf("[Origin Group] Would dispatch to node %d (ip=%s)\n", node.ID, node.MainIP)
	}
}

// UpdateRequest 更新回源分组请求
type UpdateRequest struct {
	ID          int    `json:"id" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// Update 更新回源分组
// POST /api/v1/origin-groups/update
func (h *Handler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing("invalid request"))
		return
	}

	// 检查分组是否存在
	var group model.OriginGroup
	if err := h.db.First(&group, req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("origin group not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	// 检查name唯一性（排除自己）
	var existingGroup model.OriginGroup
	if err := h.db.Where("name = ? AND id != ?", req.Name, req.ID).First(&existingGroup).Error; err == nil {
		httpx.FailErr(c, httpx.ErrAlreadyExists("origin group name already exists"))
		return
	}

	// 更新字段
	group.Name = req.Name
	group.Description = req.Description

	if err := h.db.Save(&group).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	// 返回更新后的分组
	resp := CreateResponse{
		ID:          int(group.ID),
		Name:        group.Name,
		Description: group.Description,
		Status:      group.Status,
		CreatedAt:   group.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
		UpdatedAt:   group.UpdatedAt.Format("2006-01-02T15:04:05-07:00"),
	}

	httpx.OK(c, gin.H{"item": resp})
}

// DeleteRequest 删除回源分组请求
type DeleteRequest struct {
	ID int `json:"id" binding:"required"`
}

// Delete 删除回源分组
// POST /api/v1/origin-groups/delete
func (h *Handler) Delete(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing("invalid request"))
		return
	}

	// 检查分组是否存在
	var group model.OriginGroup
	if err := h.db.First(&group, req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("origin group not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	// 开启事务
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除所有关联的地址
	if err := tx.Where("origin_group_id = ?", req.ID).Delete(&model.OriginGroupAddress{}).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	// 删除分组
	if err := tx.Delete(&group).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	httpx.OK(c, nil)
}
