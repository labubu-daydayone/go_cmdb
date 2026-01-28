package origin_sets

import (
	"encoding/json"
	"strconv"

	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

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

	// 将地址序列化为 JSON（冻结快照）
	snapshotData := map[string]interface{}{
		"originGroupId": req.OriginGroupID,
		"addresses":     addresses,
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

	// 更新 website 的 origin_set_id
	if err := h.db.Model(&website).Update("origin_set_id", req.OriginSetID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("database error", err))
		return
	}

	httpx.OK(c, nil)
}
