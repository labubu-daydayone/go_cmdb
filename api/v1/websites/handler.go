package websites

import (
	"go_cmdb/internal/httpx"
	"strconv"

	"go_cmdb/internal/cert"
	"go_cmdb/internal/model"
	"go_cmdb/internal/upstream"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler 网站管理handler
type Handler struct {
	db          *gorm.DB
	certService *cert.Service
}

// NewHandler 创建handler
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		db:          db,
		certService: cert.NewService(db),
	}
}

// ListRequest 列表请求
type ListRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"pageSize" form:"pageSize"`
	Domain   string `json:"domain" form:"domain"` // 域名搜索
	Status   string `json:"status" form:"status"` // 状态筛选
}

// ListResponse 列表响应
type ListResponse struct {
	Items    []WebsiteListItemDTO `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
}

// List 网站列表
func (h *Handler) List(c *gin.Context) {
	var req ListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid query parameters"))
		return
	}

	// 默认分页
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 15
	}

	// 构建查询
	query := h.db.Model(&model.Website{})

	// 域名搜索（通过website_domains表）
	if req.Domain != "" {
		query = query.Where("id IN (?)", h.db.Model(&model.WebsiteDomain{}).
			Select("website_id").
			Where("domain LIKE ?", "%"+req.Domain+"%"))
	}

	// 状态筛选
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to count websites", err))
		return
	}

	// 查询列表
	var websites []model.Website
	offset := (req.Page - 1) * req.PageSize
	if err := query.
		Preload("LineGroup").
		Preload("OriginGroup").
		Preload("Domains").
		Preload("HTTPS").
		Offset(offset).
		Limit(req.PageSize).
		Order("id DESC").
		Find(&websites).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query websites", err))
		return
	}

	// 转换为响应格式
	list := make([]WebsiteListItemDTO, 0, len(websites))
	for _, w := range websites {
		item := WebsiteListItemDTO{
			ID:                 w.ID,
			LineGroupID:        w.LineGroupID,
			CacheRuleID:        w.CacheRuleID,
			OriginMode:         w.OriginMode,
			RedirectURL:        w.RedirectURL,
			RedirectStatusCode: w.RedirectStatusCode,
			Status:             w.Status,
			CreatedAt:          w.CreatedAt,
			UpdatedAt:          w.UpdatedAt,
		}

		// OriginGroupID 和 OriginSetID 处理
		if w.OriginGroupID.Valid {
			val := int(w.OriginGroupID.Int32)
			item.OriginGroupID = &val
		}
		if w.OriginSetID.Valid {
			val := int(w.OriginSetID.Int32)
			item.OriginSetID = &val
		}

		// LineGroup名称和CNAME
		if w.LineGroup != nil {
			item.LineGroupName = w.LineGroup.Name
			// 加载Domain信息以计算CNAME
			var domain model.Domain
			if err := h.db.First(&domain, w.LineGroup.DomainID).Error; err == nil {
				item.CNAME = w.LineGroup.CNAMEPrefix + "." + domain.Domain
			}
		}

		// OriginGroup名称
		if w.OriginGroup != nil {
			item.OriginGroupName = w.OriginGroup.Name
		}

		// 域名列表
		domains := make([]string, 0, len(w.Domains))
		for _, d := range w.Domains {
			domains = append(domains, d.Domain)
		}
		item.Domains = domains

		// HTTPS状态
		if w.HTTPS != nil {
			item.HTTPSEnabled = w.HTTPS.Enabled
		}

		list = append(list, item)
	}

	httpx.OK(c, ListResponse{
		Items:    list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

type GetByIDRequest struct {
	ID string `uri:"id" binding:"required"`
}

// GetByID 根据ID查询网站详情
func (h *Handler) GetByID(c *gin.Context) {
	var req GetByIDRequest
	if err := c.ShouldBindUri(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid id"))
		return
	}

	id, err := strconv.Atoi(req.ID)
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid id format"))
		return
	}

	var website model.Website
	if err := h.db.
		Preload("LineGroup").
		Preload("OriginGroup").
		Preload("Domains").
		Preload("HTTPS").
		First(&website, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("website not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query website", err))
		return
	}

	// 转换为 DTO
	item := WebsiteDTO{
		ID:                 website.ID,
		LineGroupID:        website.LineGroupID,
		CacheRuleID:        website.CacheRuleID,
		OriginMode:         website.OriginMode,
		RedirectURL:        website.RedirectURL,
		RedirectStatusCode: website.RedirectStatusCode,
		Status:             website.Status,
		CreatedAt:          website.CreatedAt,
		UpdatedAt:          website.UpdatedAt,
	}

	// OriginGroupID 和 OriginSetID 处理
	if website.OriginGroupID.Valid {
		val := int(website.OriginGroupID.Int32)
		item.OriginGroupID = &val
	}
	if website.OriginSetID.Valid {
		val := int(website.OriginSetID.Int32)
		item.OriginSetID = &val
	}

	// LineGroup名称和CNAME
	if website.LineGroup != nil {
		item.LineGroupName = website.LineGroup.Name
		var domain model.Domain
		if err := h.db.First(&domain, website.LineGroup.DomainID).Error; err == nil {
			item.CNAME = website.LineGroup.CNAMEPrefix + "." + domain.Domain
		}
	}

	// OriginGroup名称
	if website.OriginGroup != nil {
		item.OriginGroupName = website.OriginGroup.Name
	}

	// 域名列表
	domains := make([]string, 0, len(website.Domains))
	for _, d := range website.Domains {
		domains = append(domains, d.Domain)
	}
	item.Domains = domains

	// HTTPS状态
	if website.HTTPS != nil {
		item.HTTPSEnabled = website.HTTPS.Enabled
	}

	httpx.OK(c, gin.H{
		"item": item,
	})
}

// validateCertificateCoverage validates if a certificate covers all website domains
// Returns AppError if coverage is not complete (T2-07)
func (h *Handler) validateCertificateCoverage(tx *gorm.DB, certificateID int, websiteID int) *httpx.AppError {
	// Check if certificate exists
	var certExists bool
	if err := tx.Raw("SELECT EXISTS(SELECT 1 FROM certificates WHERE id = ?)", certificateID).Scan(&certExists).Error; err != nil {
		return httpx.ErrDatabaseError("failed to check certificate", err)
	}

	if !certExists {
		return httpx.ErrNotFound("certificate not found")
	}

	// Get certificate domains
	certDomains, err := h.certService.GetCertificateDomains(certificateID)
	if err != nil {
		return httpx.ErrDatabaseError("failed to get certificate domains", err)
	}

	// Get website domains
	websiteDomains, err := h.certService.GetWebsiteDomains(websiteID)
	if err != nil {
		return httpx.ErrDatabaseError("failed to get website domains", err)
	}

	// Calculate coverage
	coverage := cert.CalculateCoverage(certDomains, websiteDomains)

	// Only allow if fully covered
	if coverage.Status != cert.CoverageStatusCovered {
		return httpx.ErrStateConflict("Certificate does not cover all website domains").WithData(gin.H{
			"certificateDomains": certDomains,
			"websiteDomains":     websiteDomains,
			"missingDomains":     coverage.MissingDomains,
			"coverageStatus":     coverage.Status,
		})
	}

	return nil
}

// BindOriginSetRequest 绑定 Origin Set 请求
type BindOriginSetRequest struct {
	WebsiteID   int64 `json:"websiteId" binding:"required"`
	OriginSetID int64 `json:"originSetId" binding:"required"`
}

// BindOriginSetResponse 绑定 Origin Set 响应
type BindOriginSetResponse struct {
	Item BindOriginSetItem `json:"item"`
}

// BindOriginSetItem 绑定 Origin Set 响应项
type BindOriginSetItem struct {
	ReleaseID int64 `json:"releaseId"`
}

// BindOriginSet 绑定 Origin Set 并触发发布
func (h *Handler) BindOriginSet(c *gin.Context) {
	var req BindOriginSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request body"))
		return
	}

	// 1. 更新 website 的 origin_set_id
	err := h.db.Model(&model.Website{}).
		Where("id = ?", req.WebsiteID).
		Updates(map[string]interface{}{
			"origin_set_id": req.OriginSetID,
			"origin_mode":   "group",
		}).Error
	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to update website", err))
		return
	}

	// 2. 触发发布（这里需要调用 upstream publisher）
	// 为了避免循环依赖，我们在这里直接调用 publisher
	publisher := upstream.NewPublisher(h.db)
	publishResp, err := publisher.Publish(&upstream.PublishRequest{
		WebsiteID:   req.WebsiteID,
		OriginSetID: req.OriginSetID,
	})
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.FailErr(c, appErr)
		} else {
			httpx.FailErr(c, httpx.ErrInternalError("failed to publish", err))
		}
		return
	}

	httpx.OK(c, BindOriginSetResponse{
		Item: BindOriginSetItem{
			ReleaseID: publishResp.ReleaseID,
		},
	})
}

// UpdateRequest 更新请求
type UpdateRequest struct {
	ID                 int     `json:"id" binding:"required"`
	LineGroupID        *int    `json:"lineGroupId"`
	CacheRuleID        *int    `json:"cacheRuleId"`
	OriginMode         *string `json:"originMode"`
	OriginGroupID      *int    `json:"originGroupId"`
	OriginSetID        *int    `json:"originSetId"`
	RedirectURL        *string `json:"redirectUrl"`
	RedirectStatusCode *int    `json:"redirectStatusCode"`
	Status             *string `json:"status"`
}

// Update 更新网站
func (h *Handler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request body"))
		return
	}

	var website model.Website
	if err := h.db.First(&website, req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("website not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query website", err))
		return
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.LineGroupID != nil {
		updates["line_group_id"] = *req.LineGroupID
	}
	if req.CacheRuleID != nil {
		updates["cache_rule_id"] = *req.CacheRuleID
	}
	if req.OriginMode != nil {
		updates["origin_mode"] = *req.OriginMode
	}
	if req.OriginGroupID != nil {
		if *req.OriginGroupID > 0 {
			updates["origin_group_id"] = *req.OriginGroupID
		} else {
			updates["origin_group_id"] = nil
		}
	}
	if req.OriginSetID != nil {
		if *req.OriginSetID > 0 {
			updates["origin_set_id"] = *req.OriginSetID
		} else {
			updates["origin_set_id"] = nil
		}
	}
	if req.RedirectURL != nil {
		updates["redirect_url"] = *req.RedirectURL
	}
	if req.RedirectStatusCode != nil {
		updates["redirect_status_code"] = *req.RedirectStatusCode
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}

	if err := h.db.Model(&website).Updates(updates).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to update website", err))
		return
	}

	// 重新查询
	if err := h.db.
		Preload("LineGroup").
		Preload("OriginGroup").
		Preload("Domains").
		Preload("HTTPS").
		First(&website, req.ID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query updated website", err))
		return
	}

	// 转换为 DTO
	item := WebsiteDTO{
		ID:                 website.ID,
		LineGroupID:        website.LineGroupID,
		CacheRuleID:        website.CacheRuleID,
		OriginMode:         website.OriginMode,
		RedirectURL:        website.RedirectURL,
		RedirectStatusCode: website.RedirectStatusCode,
		Status:             website.Status,
		CreatedAt:          website.CreatedAt,
		UpdatedAt:          website.UpdatedAt,
	}

	// OriginGroupID 和 OriginSetID 处理
	if website.OriginGroupID.Valid {
		val := int(website.OriginGroupID.Int32)
		item.OriginGroupID = &val
	}
	if website.OriginSetID.Valid {
		val := int(website.OriginSetID.Int32)
		item.OriginSetID = &val
	}

	// LineGroup名称和CNAME
	if website.LineGroup != nil {
		item.LineGroupName = website.LineGroup.Name
		var domain model.Domain
		if err := h.db.First(&domain, website.LineGroup.DomainID).Error; err == nil {
			item.CNAME = website.LineGroup.CNAMEPrefix + "." + domain.Domain
		}
	}

	// OriginGroup名称
	if website.OriginGroup != nil {
		item.OriginGroupName = website.OriginGroup.Name
	}

	// 域名列表
	domains := make([]string, 0, len(website.Domains))
	for _, d := range website.Domains {
		domains = append(domains, d.Domain)
	}
	item.Domains = domains

	// HTTPS状态
	if website.HTTPS != nil {
		item.HTTPSEnabled = website.HTTPS.Enabled
	}

	httpx.OK(c, item)
}

// DeleteRequest 删除请求
type DeleteRequest struct {
	IDs []int `json:"ids" binding:"required"`
}

// Delete 删除网站
func (h *Handler) Delete(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request body"))
		return
	}

	if len(req.IDs) == 0 {
		httpx.FailErr(c, httpx.ErrParamInvalid("ids cannot be empty"))
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		for _, id := range req.IDs {
			// 删除 website_domains
			if err := tx.Where("website_id = ?", id).Delete(&model.WebsiteDomain{}).Error; err != nil {
				return err
			}

			// 删除 website_https（如果存在）
			if err := tx.Where("website_id = ?", id).Delete(&model.WebsiteHTTPS{}).Error; err != nil {
				return err
			}

			// 删除 website
			if err := tx.Delete(&model.Website{}, id).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to delete websites", err))
		return
	}

	httpx.OK(c, nil)
}
