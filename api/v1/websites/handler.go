package websites

import (
	"go_cmdb/internal/httpx"
	"log"
	"strconv"

	"go_cmdb/internal/cert"
	"go_cmdb/internal/model"
	"go_cmdb/internal/upstream"
	"go_cmdb/internal/ws"

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
	Items    []WebsiteItem `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

// WebsiteItem 网站列表项
type WebsiteItem struct {
	ID                 int      `json:"id"`
	LineGroupID        int      `json:"line_group_id"`
	LineGroupName      string   `json:"line_group_name"`
	CacheRuleID        int      `json:"cache_rule_id"`
	OriginMode         string   `json:"origin_mode"`
	OriginGroupID      int      `json:"origin_group_id"`
	OriginGroupName    string   `json:"origin_group_name"`
	OriginSetID        int      `json:"origin_set_id"`
	RedirectURL        string   `json:"redirect_url"`
	RedirectStatusCode int      `json:"redirect_status_code"`
	Status             string   `json:"status"`
	Domains            []string `json:"domains"`
	PrimaryDomain      string   `json:"primary_domain"`
	CNAME              string   `json:"cname"`
	HTTPSEnabled       bool     `json:"https_enabled"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
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
	list := make([]WebsiteItem, 0, len(websites))
	for _, w := range websites {
		item := WebsiteItem{
			ID:                 w.ID,
			LineGroupID:        w.LineGroupID,
			CacheRuleID:        w.CacheRuleID,
			OriginMode:         w.OriginMode,
			OriginGroupID:      w.OriginGroupID,
			OriginSetID:        w.OriginSetID,
			RedirectURL:        w.RedirectURL,
			RedirectStatusCode: w.RedirectStatusCode,
			Status:             w.Status,
			CreatedAt:          w.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:          w.UpdatedAt.Format("2006-01-02 15:04:05"),
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
			if d.IsPrimary {
				item.PrimaryDomain = d.Domain
			}
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

// CreateRequest 创建请求
type CreateRequest struct {
	LineGroupID int      `json:"line_group_id" binding:"required"`
	CacheRuleID int      `json:"cache_rule_id"`
	Domains     []string `json:"domains" binding:"required,min=1"` // 至少一个域名

	// 回源配置
	OriginMode         string                   `json:"origin_mode" binding:"required,oneof=group manual redirect"`
	OriginGroupID      int                      `json:"origin_group_id"`      // group模式时必填
	OriginAddresses    []OriginAddressInput     `json:"origin_addresses"`     // manual模式时必填
	RedirectURL        string                   `json:"redirect_url"`         // redirect模式时必填
	RedirectStatusCode int                      `json:"redirect_status_code"` // redirect模式时必填

	// HTTPS配置（可选）
	HTTPS *HTTPSInput `json:"https"`
}

// OriginAddressInput 回源地址输入
type OriginAddressInput struct {
	Role     string `json:"role" binding:"required,oneof=primary backup"`
	Protocol string `json:"protocol" binding:"required,oneof=http https"`
	Address  string `json:"address" binding:"required"`
	Weight   int    `json:"weight"`
	Enabled  bool   `json:"enabled"`
}

// HTTPSInput HTTPS配置输入
type HTTPSInput struct {
	Enabled       bool   `json:"enabled"`
	ForceRedirect bool   `json:"force_redirect"`
	HSTS          bool   `json:"hsts"`
	CertMode      string `json:"cert_mode" binding:"required,oneof=select acme"`
	CertificateID int    `json:"certificate_id"`   // select模式时必填
	ACMEProviderID int   `json:"acme_provider_id"` // acme模式时必填
	ACMEAccountID  int   `json:"acme_account_id"`  // acme模式时必填
}

// Create 创建网站
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request body"))
		return
	}

	// 参数校验
	if err := h.validateCreateRequest(&req); err != nil {
		httpx.FailErr(c, err)
		return
	}

	// 事务处理
	var websiteID int
	err := h.db.Transaction(func(tx *gorm.DB) error {
		// 1. 查询line_group
		var lineGroup model.LineGroup
		if err := tx.First(&lineGroup, req.LineGroupID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return httpx.ErrNotFound("line group not found")
			}
			return httpx.ErrDatabaseError("failed to query line group", err)
		}

		// 加载Domain信息以计算CNAME
		var domain model.Domain
		if err := tx.First(&domain, lineGroup.DomainID).Error; err != nil {
			return httpx.ErrDatabaseError("failed to query domain", err)
		}
		cname := lineGroup.CNAMEPrefix + "." + domain.Domain
		// 2. 创建 website
		website := model.Website{
			LineGroupID:        req.LineGroupID,
			CacheRuleID:        req.CacheRuleID,
			OriginMode:         req.OriginMode,
			RedirectURL:        req.RedirectURL,
			RedirectStatusCode: req.RedirectStatusCode,
			Status:             model.WebsiteStatusActive,
		}

		// 根据 originMode 设置 origin 字段
		switch req.OriginMode {
		case "group":
			// group 模式：只设置 originGroupID
			website.OriginGroupID = &req.OriginGroupID
			// originSetID 保持 NULL（默认值）
		case "manual":
			// manual 模式：originGroupID 和 originSetID 都为 NULL
			// 不设置任何值，保持默认 NULL
		case "redirect":
			// redirect 模式：originGroupID 和 originSetID 都为 NULL
			// 不设置任何值，保持默认 NULL
		}

		if err := tx.Create(&website).Error; err != nil {
			return httpx.ErrDatabaseError("failed to create website", err)
		}
		websiteID = website.ID

		// 3. 创建website_domains
		for i, domain := range req.Domains {
			// 检查域名是否已存在
			var count int64
			if err := tx.Model(&model.WebsiteDomain{}).Where("domain = ?", domain).Count(&count).Error; err != nil {
				return httpx.ErrDatabaseError("failed to check domain", err)
			}
			if count > 0 {
				return httpx.ErrAlreadyExists("domain already exists: " + domain)
			}

			// 创建域名记录
			wd := model.WebsiteDomain{
				WebsiteID: website.ID,
				Domain:    domain,
				IsPrimary: i == 0, // 第一个域名为主域名
			}
			if err := tx.Create(&wd).Error; err != nil {
				return httpx.ErrDatabaseError("failed to create website domain", err)
			}

			// 4. 生成DNS记录（CNAME）
			dnsRecord := model.DomainDNSRecord{
				DomainID:  int(lineGroup.DomainID),
				OwnerType: "website_domain",
				OwnerID:   wd.ID,
				Type:      "CNAME",
				Name:      domain, // 完整域名
				Value:     cname,
				TTL:       600,
				Status:    "pending",
			}
			if err := tx.Create(&dnsRecord).Error; err != nil {
				return httpx.ErrDatabaseError("failed to create DNS record", err)
			}
		}

		// 5. 根据origin_mode创建origin_set
		if req.OriginMode == model.OriginModeGroup {
			// group模式：从origin_group复制
			if err := h.createOriginSetFromGroup(tx, website.ID, req.OriginGroupID); err != nil {
				return err
			}
		} else if req.OriginMode == model.OriginModeManual {
			// manual模式：手动创建
			if err := h.createOriginSetManual(tx, website.ID, req.OriginAddresses); err != nil {
				return err
			}
		}

			// 6. 创建website_https（如果enabled）
			if req.HTTPS != nil && req.HTTPS.Enabled {
				// select模式：校验证书覆盖
				if req.HTTPS.CertMode == model.CertModeSelect {
					if err := h.validateCertificateCoverage(tx, req.HTTPS.CertificateID, website.ID); err != nil {
						return err
					}
				}

				https := model.WebsiteHTTPS{
					WebsiteID:      website.ID,
					Enabled:        true,
					ForceRedirect:  req.HTTPS.ForceRedirect,
					HSTS:           req.HTTPS.HSTS,
					CertMode:       req.HTTPS.CertMode,
					CertificateID:  req.HTTPS.CertificateID,
					ACMEProviderID: req.HTTPS.ACMEProviderID,
					ACMEAccountID:  req.HTTPS.ACMEAccountID,
				}
				if err := tx.Create(&https).Error; err != nil {
					return httpx.ErrDatabaseError("failed to create website https", err)
				}
			}

		return nil
	})

		if err != nil {
			if appErr, ok := err.(*httpx.AppError); ok {
				httpx.FailErr(c, appErr)
			} else {
				httpx.FailErr(c, httpx.ErrDatabaseError("transaction failed", err))
			}
			return
		}

		// Publish website event (after transaction success)
		// Note: Broadcast failure should not affect the main flow
		if err := ws.PublishWebsiteEvent("add", gin.H{"id": websiteID}); err != nil {
			log.Printf("[WebSocket] Failed to publish website event: %v", err)
		}

		httpx.OK(c, gin.H{"id": websiteID})
}

// validateCreateRequest 校验创建请求
func (h *Handler) validateCreateRequest(req *CreateRequest) *httpx.AppError {
	// 校验origin_mode
	switch req.OriginMode {
	case model.OriginModeGroup:
		if req.OriginGroupID <= 0 {
			return httpx.ErrParamMissing("origin_group_id is required for group mode")
		}
	case model.OriginModeManual:
		if len(req.OriginAddresses) == 0 {
			return httpx.ErrParamMissing("origin_addresses is required for manual mode")
		}
	case model.OriginModeRedirect:
		if req.RedirectURL == "" {
			return httpx.ErrParamMissing("redirect_url is required for redirect mode")
		}
		if req.RedirectStatusCode == 0 {
			req.RedirectStatusCode = 301 // 默认301
		}
	}

	// 校验HTTPS配置
	if req.HTTPS != nil && req.HTTPS.Enabled {
		if req.HTTPS.CertMode == model.CertModeSelect {
			if req.HTTPS.CertificateID <= 0 {
				return httpx.ErrParamMissing("certificate_id is required for select mode")
			}
		} else if req.HTTPS.CertMode == model.CertModeACME {
			if req.HTTPS.ACMEProviderID <= 0 && req.HTTPS.ACMEAccountID <= 0 {
				return httpx.ErrParamMissing("acme_provider_id or acme_account_id is required for acme mode")
			}
			// ACME模式：简单校验域名非空和合法性
			if len(req.Domains) == 0 {
				return httpx.ErrParamMissing("domains is required for acme mode")
			}
			// 简单域名合法性校验（可选）
			for _, domain := range req.Domains {
				if domain == "" {
					return httpx.ErrParamInvalid("domain cannot be empty")
				}
			}
		}
	}

	return nil
}

// createOriginSetFromGroup 从origin_group创建origin_set
func (h *Handler) createOriginSetFromGroup(tx *gorm.DB, websiteID, originGroupID int) *httpx.AppError {
	// 查询origin_group
	var originGroup model.OriginGroup
	if err := tx.Preload("Addresses").First(&originGroup, originGroupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.ErrNotFound("origin group not found")
		}
		return httpx.ErrDatabaseError("failed to query origin group", err)
	}

	// 创建origin_set
	originSet := model.OriginSet{
		Source:        "group",
		OriginGroupID: int64(originGroupID),
	}
	if err := tx.Create(&originSet).Error; err != nil {
		return httpx.ErrDatabaseError("failed to create origin set", err)
	}

	// 复制addresses
	for _, addr := range originGroup.Addresses {
		originAddr := model.OriginAddress{
			OriginSetID: originSet.ID,
			Role:        addr.Role,
			Protocol:    addr.Protocol,
			Address:     addr.Address,
			Weight:      addr.Weight,
			Enabled:     addr.Enabled,
		}
		if err := tx.Create(&originAddr).Error; err != nil {
			return httpx.ErrDatabaseError("failed to create origin address", err)
		}
	}

	// 更新website的origin_set_id
	if err := tx.Model(&model.Website{}).Where("id = ?", websiteID).Update("origin_set_id", originSet.ID).Error; err != nil {
		return httpx.ErrDatabaseError("failed to update website", err)
	}

	return nil
}

// createOriginSetManual 手动创建origin_set
func (h *Handler) createOriginSetManual(tx *gorm.DB, websiteID int, addresses []OriginAddressInput) *httpx.AppError {
	// 创建origin_set
	originSet := model.OriginSet{
		Source:        "manual",
		OriginGroupID: 0,
	}
	if err := tx.Create(&originSet).Error; err != nil {
		return httpx.ErrDatabaseError("failed to create origin set", err)
	}

	// 创建addresses
	for _, addr := range addresses {
		originAddr := model.OriginAddress{
			OriginSetID: originSet.ID,
			Role:        addr.Role,
			Protocol:    addr.Protocol,
			Address:     addr.Address,
			Weight:      addr.Weight,
			Enabled:     addr.Enabled,
		}
		if err := tx.Create(&originAddr).Error; err != nil {
			return httpx.ErrDatabaseError("failed to create origin address", err)
		}
	}

	// 更新website的origin_set_id
	if err := tx.Model(&model.Website{}).Where("id = ?", websiteID).Update("origin_set_id", originSet.ID).Error; err != nil {
		return httpx.ErrDatabaseError("failed to update website", err)
	}

	return nil
}

// UpdateRequest 更新请求
type UpdateRequest struct {
	ID          int    `json:"id" binding:"required"`
	LineGroupID int    `json:"line_group_id"`
	CacheRuleID int    `json:"cache_rule_id"`
	Status      string `json:"status" binding:"omitempty,oneof=active inactive"`

	// 回源配置更新（可选）
	OriginMode         *string                  `json:"origin_mode"`
	OriginGroupID      *int                     `json:"origin_group_id"`
	OriginAddresses    []OriginAddressInput     `json:"origin_addresses"`
	RedirectURL        *string                  `json:"redirect_url"`
	RedirectStatusCode *int                     `json:"redirect_status_code"`

	// HTTPS配置更新（可选）
	HTTPS *HTTPSInput `json:"https"`
}

// Update 更新网站
func (h *Handler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request body"))
		return
	}

	// 事务处理
	err := h.db.Transaction(func(tx *gorm.DB) error {
		// 查询website
		var website model.Website
		if err := tx.Preload("Domains").First(&website, req.ID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return httpx.ErrNotFound("website not found")
			}
			return httpx.ErrDatabaseError("failed to query website", err)
		}

		// 更新基础字段
		updates := make(map[string]interface{})
		if req.LineGroupID > 0 && req.LineGroupID != website.LineGroupID {
			// 切换line_group
			var lineGroup model.LineGroup
			if err := tx.First(&lineGroup, req.LineGroupID).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return httpx.ErrNotFound("line group not found")
				}
				return httpx.ErrDatabaseError("failed to query line group", err)
			}

			// 加载Domain信息以计算CNAME
			var domain model.Domain
			if err := tx.First(&domain, lineGroup.DomainID).Error; err != nil {
				return httpx.ErrDatabaseError("failed to query domain", err)
			}
			cname := lineGroup.CNAMEPrefix + "." + domain.Domain

			updates["line_group_id"] = req.LineGroupID

			// 标记旧DNS记录为error
			if err := tx.Model(&model.DomainDNSRecord{}).
				Where("owner_type = ? AND owner_id IN (?)",
					"website_domain",
					tx.Model(&model.WebsiteDomain{}).Select("id").Where("website_id = ?", website.ID)).
				Updates(map[string]interface{}{
					"status":     "error",
					"last_error": "line group changed",
				}).Error; err != nil {
				return httpx.ErrDatabaseError("failed to update DNS records", err)
			}

			// 生成新DNS记录
			for _, domain := range website.Domains {
				dnsRecord := model.DomainDNSRecord{
					DomainID:  int(lineGroup.DomainID),
					OwnerType: "website_domain",
					OwnerID:   domain.ID,
					Type:      "CNAME",
					Name:      domain.Domain,
					Value:     cname,
					TTL:       600,
					Status:    "pending",
				}
				if err := tx.Create(&dnsRecord).Error; err != nil {
					return httpx.ErrDatabaseError("failed to create DNS record", err)
				}
			}
		}

		if req.CacheRuleID >= 0 {
			updates["cache_rule_id"] = req.CacheRuleID
		}

		if req.Status != "" {
			updates["status"] = req.Status
		}

		// 更新origin_mode
		if req.OriginMode != nil {
			oldMode := website.OriginMode
			newMode := *req.OriginMode

			// 状态机校验（简化版，实际可能需要更复杂的逻辑）
			if oldMode != newMode {
				// 删除旧origin_set
				if website.OriginSetID > 0 {
					if err := tx.Delete(&model.OriginAddress{}, "origin_set_id = ?", website.OriginSetID).Error; err != nil {
						return httpx.ErrDatabaseError("failed to delete origin addresses", err)
					}
					if err := tx.Delete(&model.OriginSet{}, website.OriginSetID).Error; err != nil {
						return httpx.ErrDatabaseError("failed to delete origin set", err)
					}
				}

				// 创建新origin_set
				if newMode == model.OriginModeGroup {
					if req.OriginGroupID == nil || *req.OriginGroupID <= 0 {
						return httpx.ErrParamMissing("origin_group_id is required for group mode")
					}
					if err := h.createOriginSetFromGroup(tx, website.ID, *req.OriginGroupID); err != nil {
						return err
					}
					updates["origin_mode"] = newMode
					updates["origin_group_id"] = *req.OriginGroupID
					updates["redirect_url"] = ""
					updates["redirect_status_code"] = 0
				} else if newMode == model.OriginModeManual {
					if len(req.OriginAddresses) == 0 {
						return httpx.ErrParamMissing("origin_addresses is required for manual mode")
					}
					if err := h.createOriginSetManual(tx, website.ID, req.OriginAddresses); err != nil {
						return err
					}
					updates["origin_mode"] = newMode
					updates["origin_group_id"] = 0
					updates["redirect_url"] = ""
					updates["redirect_status_code"] = 0
				} else if newMode == model.OriginModeRedirect {
					if req.RedirectURL == nil || *req.RedirectURL == "" {
						return httpx.ErrParamMissing("redirect_url is required for redirect mode")
					}
					updates["origin_mode"] = newMode
					updates["origin_group_id"] = 0
					updates["origin_set_id"] = 0
					updates["redirect_url"] = *req.RedirectURL
					if req.RedirectStatusCode != nil {
						updates["redirect_status_code"] = *req.RedirectStatusCode
					} else {
						updates["redirect_status_code"] = 301
					}
				}
			}
		}

		// 应用更新
		if len(updates) > 0 {
			if err := tx.Model(&website).Updates(updates).Error; err != nil {
				return httpx.ErrDatabaseError("failed to update website", err)
			}
		}

		// 更新HTTPS配置
			if req.HTTPS != nil {
				// select模式且enabled：校验证书覆盖
				if req.HTTPS.Enabled && req.HTTPS.CertMode == model.CertModeSelect {
					if err := h.validateCertificateCoverage(tx, req.HTTPS.CertificateID, website.ID); err != nil {
						return err
					}
				}

				var https model.WebsiteHTTPS
				err := tx.Where("website_id = ?", website.ID).First(&https).Error
				if err == gorm.ErrRecordNotFound {
					// 创建新记录
					https = model.WebsiteHTTPS{
						WebsiteID:      website.ID,
						Enabled:        req.HTTPS.Enabled,
						ForceRedirect:  req.HTTPS.ForceRedirect,
						HSTS:           req.HTTPS.HSTS,
						CertMode:       req.HTTPS.CertMode,
						CertificateID:  req.HTTPS.CertificateID,
						ACMEProviderID: req.HTTPS.ACMEProviderID,
						ACMEAccountID:  req.HTTPS.ACMEAccountID,
					}
					if err := tx.Create(&https).Error; err != nil {
						return httpx.ErrDatabaseError("failed to create website https", err)
					}
				} else if err != nil {
					return httpx.ErrDatabaseError("failed to query website https", err)
				} else {
					// 更新现有记录
					httpsUpdates := map[string]interface{}{
						"enabled":         req.HTTPS.Enabled,
						"force_redirect":  req.HTTPS.ForceRedirect,
						"hsts":            req.HTTPS.HSTS,
						"cert_mode":       req.HTTPS.CertMode,
						"certificate_id":  req.HTTPS.CertificateID,
						"acme_provider_id": req.HTTPS.ACMEProviderID,
						"acme_account_id":  req.HTTPS.ACMEAccountID,
					}
					if err := tx.Model(&https).Updates(httpsUpdates).Error; err != nil {
						return httpx.ErrDatabaseError("failed to update website https", err)
					}
				}
			}

		return nil
	})

		if err != nil {
			if appErr, ok := err.(*httpx.AppError); ok {
				httpx.FailErr(c, appErr)
			} else {
				httpx.FailErr(c, httpx.ErrDatabaseError("transaction failed", err))
			}
			return
		}

		// Publish website event (after transaction success)
		if err := ws.PublishWebsiteEvent("update", gin.H{"id": req.ID}); err != nil {
			log.Printf("[WebSocket] Failed to publish website event: %v", err)
		}

		httpx.OK(c, gin.H{"success": true})
	}

// DeleteRequest 删除请求
type DeleteRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}

// Delete 删除网站
func (h *Handler) Delete(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request body"))
		return
	}

	// 事务处理
	err := h.db.Transaction(func(tx *gorm.DB) error {
		for _, id := range req.IDs {
			// 查询website
			var website model.Website
			if err := tx.First(&website, id).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					continue // 跳过不存在的记录
				}
				return httpx.ErrDatabaseError("failed to query website", err)
			}

			// 删除certificate_bindings
			if err := tx.Where("owner_type = ? AND owner_id = ?", "website", id).Delete(&model.CertificateBinding{}).Error; err != nil {
				return httpx.ErrDatabaseError("failed to delete certificate bindings", err)
			}

			// 标记DNS记录为error
			if err := tx.Model(&model.DomainDNSRecord{}).
				Where("owner_type = ? AND owner_id IN (?)",
					"website_domain",
					tx.Model(&model.WebsiteDomain{}).Select("id").Where("website_id = ?", id)).
				Updates(map[string]interface{}{
					"status":     "error",
					"last_error": "website deleted",
				}).Error; err != nil {
				return httpx.ErrDatabaseError("failed to update DNS records", err)
			}

			// 删除website_domains
			if err := tx.Where("website_id = ?", id).Delete(&model.WebsiteDomain{}).Error; err != nil {
				return httpx.ErrDatabaseError("failed to delete website domains", err)
			}

			// 删除website_https
			if err := tx.Where("website_id = ?", id).Delete(&model.WebsiteHTTPS{}).Error; err != nil {
				return httpx.ErrDatabaseError("failed to delete website https", err)
			}

			// 删除website
			if err := tx.Delete(&website).Error; err != nil {
				return httpx.ErrDatabaseError("failed to delete website", err)
			}
		}

		return nil
	})

		if err != nil {
			if appErr, ok := err.(*httpx.AppError); ok {
				httpx.FailErr(c, appErr)
			} else {
				httpx.FailErr(c, httpx.ErrDatabaseError("transaction failed", err))
			}
			return
		}

		// Publish website events (after transaction success)
		for _, id := range req.IDs {
			if err := ws.PublishWebsiteEvent("delete", gin.H{"id": id}); err != nil {
				log.Printf("[WebSocket] Failed to publish website event: %v", err)
			}
		}

		httpx.OK(c, nil)
}

// GetByIDRequest 根据ID查询请求
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
		Preload("OriginSet.Addresses").
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

	httpx.OK(c, website)
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
