package origin_sets

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"
	"go_cmdb/internal/service"
	"go_cmdb/internal/util"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler Origin Set handler
type Handler struct {
	db *gorm.DB
}

// NewHandler 创建handler实例
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// CreateRequest 创建快照请求
type CreateRequest struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	OriginGroupID int64  `json:"originGroupId" binding:"required"`
}

// CreateResponse 创建快照响应
type CreateResponse struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Status        string `json:"status"`
	OriginGroupID int64  `json:"originGroupId"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

// Create 创建快照
// POST /api/v1/origin-sets/create
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing("invalid request"))
		return
	}

	// 检查 origin group 是否存在
	var originGroup model.OriginGroup
	if err := h.db.First(&originGroup, req.OriginGroupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("origin group not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	// 查询 origin group 的所有地址
	var addresses []model.OriginGroupAddress
	if err := h.db.Where("origin_group_id = ?", req.OriginGroupID).Find(&addresses).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	// 校验：必须至少有一个 enabled 的 primary 地址
	enabledPrimaryCount := 0
	for _, addr := range addresses {
		if addr.Enabled && addr.Role == model.OriginRolePrimary {
			enabledPrimaryCount++
		}
	}
	if enabledPrimaryCount == 0 {
		httpx.FailErr(c, httpx.ErrParamInvalid("origin group must have at least one enabled primary address"))
		return
	}

	// 构建 addresses 数组（用于 snapshot）
	type SnapshotAddress struct {
		Role     string `json:"role"`
		Protocol string `json:"protocol"`
		Address  string `json:"address"`
		Weight   int    `json:"weight"`
		Enabled  bool   `json:"enabled"`
	}
	snapshotAddresses := make([]SnapshotAddress, 0, len(addresses))
	for _, addr := range addresses {
		snapshotAddresses = append(snapshotAddresses, SnapshotAddress{
			Role:     addr.Role,
			Protocol: addr.Protocol,
			Address:  addr.Address,
			Weight:   addr.Weight,
			Enabled:  addr.Enabled,
		})
	}

	// 将地址序列化为 JSON（冻结快照）
	snapshotData := map[string]interface{}{
		"originGroupId": req.OriginGroupID,
		"addresses":     snapshotAddresses,
	}
	snapshotJSON, err := json.Marshal(snapshotData)
	if err != nil {
		httpx.FailErr(c, httpx.ErrInternalError("failed to serialize snapshot", err))
		return
	}

	// 开启事务
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建 origin set
	originSet := model.OriginSet{
		Name:          req.Name,
		Description:   req.Description,
		Status:        "active",
		Source:        model.OriginSetSourceGroup,
		OriginGroupID: req.OriginGroupID,
	}
	if err := tx.Create(&originSet).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	// 创建 origin set item
	originSetItem := model.OriginSetItem{
		OriginSetID:   int64(originSet.ID),
		OriginGroupID: req.OriginGroupID,
		SnapshotJSON:  string(snapshotJSON),
	}
	if err := tx.Create(&originSetItem).Error; err != nil {
		tx.Rollback()
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	if err := tx.Commit().Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	// 返回响应
	resp := CreateResponse{
		ID:            int(originSet.ID),
		Name:          originSet.Name,
		Description:   originSet.Description,
		Status:        originSet.Status,
		OriginGroupID: originSet.OriginGroupID,
		CreatedAt:     originSet.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
		UpdatedAt:     originSet.UpdatedAt.Format("2006-01-02T15:04:05-07:00"),
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

// ListItemDTO 列表项
type ListItemDTO struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Status          string `json:"status"`
	OriginGroupID   int64  `json:"originGroupId"`
	OriginGroupName string `json:"originGroupName"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

// List 列表
// GET /api/v1/origin-sets
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "15"))
	if pageSize > 100 {
		pageSize = 100
	}

	db := h.db.Model(&model.OriginSet{})

	// 过滤
	if originGroupIDStr := c.Query("originGroupId"); originGroupIDStr != "" {
		originGroupID, err := strconv.ParseInt(originGroupIDStr, 10, 64)
		if err == nil {
			db = db.Where("origin_group_id = ?", originGroupID)
		}
	}
	if status := c.Query("status"); status != "" {
		db = db.Where("status = ?", status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	var originSets []struct {
		model.OriginSet
		OriginGroupName string `gorm:"column:origin_group_name"`
	}

	offset := (page - 1) * pageSize
	if err := db.Select("origin_sets.*, origin_groups.name as origin_group_name").
		Joins("left join origin_groups on origin_groups.id = origin_sets.origin_group_id").
		Limit(pageSize).Offset(offset).Find(&originSets).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	items := make([]ListItemDTO, 0, len(originSets))
	for _, set := range originSets {
		items = append(items, ListItemDTO{
			ID:              int(set.ID),
			Name:            set.Name,
			Description:     set.Description,
			Status:          set.Status,
			OriginGroupID:   set.OriginGroupID,
			OriginGroupName: set.OriginGroupName,
			CreatedAt:       set.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
			UpdatedAt:       set.UpdatedAt.Format("2006-01-02T15:04:05-07:00"),
		})
	}

	httpx.OK(c, ListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// DetailResponse 详情响应
type DetailResponse struct {
	ID              int                      `json:"id"`
	Name            string                   `json:"name"`
	Description     string                   `json:"description"`
	Status          string                   `json:"status"`
	OriginGroupID   int64                    `json:"originGroupId"`
	OriginGroupName string                   `json:"originGroupName"`
	Items           DetailItemsWrapper       `json:"items"`
	CreatedAt       string                   `json:"createdAt"`
	UpdatedAt       string                   `json:"updatedAt"`
}

// DetailItemsWrapper 详情项包装器
type DetailItemsWrapper struct {
	Items []DetailItemDTO `json:"items"`
}

// DetailItemDTO 详情项
type DetailItemDTO struct {
	ID            int                    `json:"id"`
	OriginSetID   int64                  `json:"originSetId"`
	OriginGroupID int64                  `json:"originGroupId"`
	Snapshot      map[string]interface{} `json:"snapshot"`
	CreatedAt     string                 `json:"createdAt"`
	UpdatedAt     string                 `json:"updatedAt"`
}

// Detail 详情
// GET /api/v1/origin-sets/:id
func (h *Handler) Detail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid id"))
		return
	}

	// 查询 origin set
	var originSet struct {
		model.OriginSet
		OriginGroupName string `gorm:"column:origin_group_name"`
	}
	if err := h.db.Model(&model.OriginSet{}).Select("origin_sets.*, origin_groups.name as origin_group_name").
		Joins("left join origin_groups on origin_groups.id = origin_sets.origin_group_id").
		First(&originSet, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("origin set not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	// 查询 origin set items
	var items []model.OriginSetItem
	if err := h.db.Where("origin_set_id = ?", id).Find(&items).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	// 构建响应
	detailItems := make([]DetailItemDTO, 0, len(items))
	for _, item := range items {
		var snapshot map[string]interface{}
		if err := json.Unmarshal([]byte(item.SnapshotJSON), &snapshot); err != nil {
			httpx.FailErr(c, httpx.ErrInternalError("failed to parse snapshot", err))
			return
		}

		detailItems = append(detailItems, DetailItemDTO{
			ID:            int(item.ID),
			OriginSetID:   item.OriginSetID,
			OriginGroupID: item.OriginGroupID,
			Snapshot:      snapshot,
			CreatedAt:     item.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
			UpdatedAt:     item.UpdatedAt.Format("2006-01-02T15:04:05-07:00"),
		})
	}

	resp := DetailResponse{
		ID:              int(originSet.ID),
		Name:            originSet.Name,
		Description:     originSet.Description,
		Status:          originSet.Status,
		OriginGroupID:   originSet.OriginGroupID,
		OriginGroupName: originSet.OriginGroupName,
		Items: DetailItemsWrapper{
			Items: detailItems,
		},
		CreatedAt: originSet.CreatedAt.Format("2006-01-02T15:04:05-07:00"),
		UpdatedAt: originSet.UpdatedAt.Format("2006-01-02T15:04:05-07:00"),
	}

	httpx.OK(c, gin.H{"item": resp})
}

// BindWebsiteRequest 绑定网站请求
type BindWebsiteRequest struct {
	WebsiteID   int64 `json:"websiteId" binding:"required"`
	OriginSetID int64 `json:"originSetId" binding:"required"`
}

// BindWebsite 绑定网站
// POST /api/v1/origin-sets/bind-website
func (h *Handler) BindWebsite(c *gin.Context) {
	var req BindWebsiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamMissing("invalid request"))
		return
	}

	// 生成 traceId
	traceID := fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Intn(10000))
	util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=bind_website_enter websiteId=%d originSetId=%d", 
		traceID, req.WebsiteID, req.OriginSetID)

	// 检查 origin set 是否存在
	var originSet model.OriginSet
	if err := h.db.First(&originSet, req.OriginSetID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("origin set not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	// 检查 origin_set_items 是否有有效地址
	var originSetItems []model.OriginSetItem
	if err := h.db.Where("origin_set_id = ?", req.OriginSetID).Find(&originSetItems).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	// 解析 snapshot_json 并检查 addresses
	hasValidAddress := false
	for _, item := range originSetItems {
		var snapshot struct {
			Addresses []struct {
				Address string `json:"address"`
				Enabled bool   `json:"enabled"`
			} `json:"addresses"`
		}
		if err := json.Unmarshal([]byte(item.SnapshotJSON), &snapshot); err == nil {
			if len(snapshot.Addresses) > 0 {
				hasValidAddress = true
				break
			}
		}
	}

	if !hasValidAddress {
		httpx.FailErr(c, httpx.ErrParamInvalid("origin set has no valid addresses"))
		return
	}

	// 检查 website 是否存在
	var website model.Website
	if err := h.db.First(&website, req.WebsiteID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("website not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	// 记录旧的 originSetId
	var oldOriginSetID *int32
	if website.OriginSetID.Valid {
		val := website.OriginSetID.Int32
		oldOriginSetID = &val
	}

	// 更新 website 的 origin_set_id
	newOriginSetID := sql.NullInt32{Int32: int32(req.OriginSetID), Valid: true}
	if err := h.db.Model(&website).Update("origin_set_id", newOriginSetID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

// 同步触发 release_task（仅 group 模式）
		var releaseTaskID int64
		var taskCreated bool
		var skipReason string
		var dispatchTriggered bool
		var targetNodeCount int
		var createdAgentTaskCount int
		var skippedAgentTaskCount int
		var agentTaskCountAfter int

if website.OriginMode == model.OriginModeGroup {
			svc := service.NewReleaseTaskService(h.db)
			util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=create_task_call websiteId=%d originSetId=%d",
				traceID, req.WebsiteID, req.OriginSetID)
			task, dispatchResult, err := svc.CreateWebsiteReleaseTaskWithDispatch(req.WebsiteID, oldOriginSetID, traceID)
			if err != nil {
				util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=create_task_return taskCreated=false taskId=0 skipReason=error(%v)",
					traceID, err)
				httpx.FailErr(c, httpx.ErrInternalError("failed to create release task", err))
				return
			}

			if task != nil {
				releaseTaskID = task.ID
				taskCreated = dispatchResult.TaskCreated // Use result from service
				if !taskCreated {
					skipReason = "content unchanged"
				}
			} else {
				// This case should ideally not be hit if service logic is correct
				taskCreated = false
				skipReason = "content unchanged"
			}

			if dispatchResult != nil {
				dispatchTriggered = dispatchResult.DispatchTriggered
				targetNodeCount = dispatchResult.TargetNodeCount
				createdAgentTaskCount = dispatchResult.Created
				skippedAgentTaskCount = dispatchResult.Skipped
				agentTaskCountAfter = dispatchResult.AgentTaskCountAfter
			}

			util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=create_task_return taskCreated=%v taskId=%d skipReason=%s",
				traceID, taskCreated, releaseTaskID, skipReason)

		} else {
			taskCreated = false
			skipReason = "origin mode is not group"
			dispatchTriggered = false
			util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=create_task_return taskCreated=false taskId=0 skipReason=%s",
				traceID, skipReason)
		}

// 打点：返回响应前
		util.DebugLog("WEB_RELEASE_DEBUG traceId=%s step=response taskCreated=%v taskId=%d skipReason=%s",
			traceID, taskCreated, releaseTaskID, skipReason)

httpx.OK(c, gin.H{
			"item": gin.H{
				"releaseTaskId":         releaseTaskID,
				"taskCreated":           taskCreated,
				"skipReason":            skipReason,
				"dispatchTriggered":     dispatchTriggered,
				"targetNodeCount":       targetNodeCount,
				"createdAgentTaskCount": createdAgentTaskCount,
				"skippedAgentTaskCount": skippedAgentTaskCount,
				"agentTaskCountAfter":   agentTaskCountAfter,
			},
		})
}

// OptionsResponse represents options response
type OptionsResponse struct {
	Items []OriginSetOptionDTO `json:"items"`
}

// OriginSetOptionDTO represents an origin set option for dropdown
type OriginSetOptionDTO struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	OriginGroupID int64  `json:"originGroupId"`
	Description   string `json:"description"`
}

// Options handles GET /api/v1/origin-sets/options
func (h *Handler) Options(c *gin.Context) {
	// originGroupId is required
	originGroupIDStr := c.Query("originGroupId")
	if originGroupIDStr == "" {
		httpx.FailErr(c, httpx.ErrParamMissing("originGroupId is required"))
		return
	}

	originGroupID, err := strconv.ParseInt(originGroupIDStr, 10, 64)
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid originGroupId"))
		return
	}

	// Query only active origin sets for the specified origin group
	var originSets []model.OriginSet
	if err := h.db.
		Where("origin_group_id = ? AND status = ?", originGroupID, "active").
		Order("id DESC").
		Find(&originSets).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to fetch origin sets", err))
		return
	}

	// Convert to DTOs
	items := make([]OriginSetOptionDTO, len(originSets))
	for i, set := range originSets {
		items[i] = OriginSetOptionDTO{
			ID:            int(set.ID),
			Name:          set.Name,
			Status:        set.Status,
			OriginGroupID: set.OriginGroupID,
			Description:   set.Description,
		}
	}

	httpx.OK(c, OptionsResponse{
		Items: items,
	})
}

// triggerWebsiteReleaseTask 触发网站发布任务
func (h *Handler) triggerReleaseTask(websiteID int64, oldOriginSetID sql.NullInt32) {
	// 异步触发，不阻塞主流程
	go func() {
		traceID := fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Intn(10000))
		svc := service.NewReleaseTaskService(h.db)
		_, _ = svc.CreateWebsiteReleaseTask(websiteID, oldOriginSetID, traceID)
	}()
}
