package websites

import (
	"database/sql"
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"
	"go_cmdb/internal/service"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateRequest 创建请求
type CreateRequest struct {
	DomainsText        string  `json:"domainsText" binding:"required"`
	LineGroupID        int     `json:"lineGroupId" binding:"required"`
	CacheRuleID        int     `json:"cacheRuleId"`
	OriginMode         string  `json:"originMode" binding:"required,oneof=group manual redirect"`
	OriginGroupID      *int    `json:"originGroupId"`
	OriginSetID        *int    `json:"originSetId"`
	RedirectURL        *string `json:"redirectUrl"`
	RedirectStatusCode *int    `json:"redirectStatusCode"`
	HTTPSEnabled       *bool   `json:"httpsEnabled"`
	ForceHTTPSRedirect *bool   `json:"forceHttpsRedirect"`
}

// CreateResponse 创建响应
type CreateResponse struct {
	Items []CreateResultItem `json:"items"`
}

// CreateResultItem 单行创建结果
type CreateResultItem struct {
	Line      int      `json:"line"`
	Created   bool     `json:"created"`
	WebsiteID *int     `json:"websiteId"`
	Domains   []string `json:"domains"`
	Error     *string  `json:"error"`
}

// Create 创建网站
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request body"))
		return
	}

	// 参数校验
	if err := validateCreateRequest(&req); err != nil {
		httpx.FailErr(c, err)
		return
	}

	// 校验 originGroupId 和 originSetId 的存在性和关联性
	if err := validateOriginReferences(h.db, &req); err != nil {
		httpx.FailErr(c, err)
		return
	}

	// 解析文本
	lines := parseText(req.DomainsText)
	if len(lines) == 0 {
		httpx.FailErr(c, httpx.ErrParamMissing("domains required"))
		return
	}

	// 检查每行是否至少有 1 个域名
	for i, domains := range lines {
		if len(domains) == 0 {
			httpx.FailErr(c, httpx.ErrParamMissing("domains required"))
			return
		}
		_ = i
	}

	// 逐行处理
	results := make([]CreateResultItem, 0, len(lines))
	for i, domains := range lines {
		lineNum := i + 1
		result := CreateResultItem{
			Line:    lineNum,
			Domains: domains,
		}

		// 在事务中创建
		err := h.db.Transaction(func(tx *gorm.DB) error {
			// 检查域名是否已存在
			for _, domain := range domains {
				var count int64
				if err := tx.Model(&model.WebsiteDomain{}).Where("domain = ?", domain).Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					errMsg := "domain already exists: " + domain
					result.Error = &errMsg
					return nil // 不回滚事务，只是标记失败
				}
			}

			// 创建 website
			website := model.Website{
				LineGroupID: req.LineGroupID,
				CacheRuleID: req.CacheRuleID,
				OriginMode:  req.OriginMode,
				Status:      model.WebsiteStatusActive,
			}

				// 根据 originMode 设置字段
				switch req.OriginMode {
				case model.OriginModeGroup:
					// group 模式：设置 originGroupId 和 originSetId
					website.OriginGroupID = sql.NullInt32{Int32: int32(*req.OriginGroupID), Valid: true}
					website.OriginSetID = sql.NullInt32{Int32: int32(*req.OriginSetID), Valid: true}
					// 确保 redirectUrl 和 redirectStatusCode 为空
					website.RedirectURL = ""
					website.RedirectStatusCode = 0
				case model.OriginModeManual:
					// manual 模式：设置 originSetId
					website.OriginSetID = sql.NullInt32{Int32: int32(*req.OriginSetID), Valid: true}
					// originGroupId 设置为 NULL
					website.OriginGroupID = sql.NullInt32{Valid: false}
					// 确保 redirectUrl 和 redirectStatusCode 为空
					website.RedirectURL = ""
					website.RedirectStatusCode = 0
				case model.OriginModeRedirect:
					// redirect 模式：设置 redirectUrl 和 redirectStatusCode
					website.RedirectURL = *req.RedirectURL
					if req.RedirectStatusCode != nil {
						website.RedirectStatusCode = *req.RedirectStatusCode
					} else {
						website.RedirectStatusCode = 301
					}
					// originGroupId 和 originSetId 设置为 NULL
					website.OriginGroupID = sql.NullInt32{Valid: false}
					website.OriginSetID = sql.NullInt32{Valid: false}
				}

			if err := tx.Create(&website).Error; err != nil {
				return err
			}

				// 创建 website_domains
				for idx, domain := range domains {
					wd := model.WebsiteDomain{
						WebsiteID: website.ID,
						Domain:    domain,
						IsPrimary: idx == 0,
					}
					if err := tx.Create(&wd).Error; err != nil {
						return err
					}
				}

				// 处理 HTTPS 配置
				if err := h.handleHTTPSConfig(tx, website.ID, req.HTTPSEnabled, req.ForceHTTPSRedirect, domains); err != nil {
					return err
				}

			result.Created = true
			websiteID := website.ID
			result.WebsiteID = &websiteID
			return nil
		})

		if err != nil {
			errMsg := err.Error()
			result.Error = &errMsg
			result.Created = false
		} else {
			// 触发 website_release_task（仅 active 且 originSetId 有效）
			if req.OriginMode == model.OriginModeGroup && req.OriginSetID != nil && *req.OriginSetID > 0 {
				h.triggerWebsiteReleaseTask(int64(*result.WebsiteID))
			}
		}

		results = append(results, result)
	}

	httpx.OK(c, CreateResponse{Items: results})
}

// parseText 解析文本为域名列表
func parseText(text string) [][]string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	lines := strings.Split(text, "\n")
	result := make([][]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 按空白分割
		tokens := strings.Fields(line)
		domains := make([]string, 0, len(tokens))
		seen := make(map[string]bool)

		for _, token := range tokens {
			domain := normalizeDomain(token)
			if domain == "" {
				continue
			}
			// 去重
			if !seen[domain] {
				domains = append(domains, domain)
				seen[domain] = true
			}
		}

		if len(domains) > 0 {
			result = append(result, domains)
		}
	}

	return result
}

// normalizeDomain 规范化域名
func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	domain = strings.ToLower(domain)
	domain = strings.TrimSuffix(domain, ".")

	// 禁止包含协议前缀
	if strings.Contains(domain, "http://") || strings.Contains(domain, "https://") || strings.Contains(domain, "/") {
		return ""
	}

	return domain
}

// validateCreateRequest 校验创建请求
func validateCreateRequest(req *CreateRequest) *httpx.AppError {
	switch req.OriginMode {
	case model.OriginModeGroup:
		// group 模式必须有 originGroupId 和 originSetId
		if req.OriginGroupID == nil || *req.OriginGroupID <= 0 {
			return httpx.ErrParamMissing("originGroupId is required for group mode")
		}
		if req.OriginSetID == nil || *req.OriginSetID <= 0 {
			return httpx.ErrParamMissing("originSetId is required for group mode")
		}
		// 禁止 redirectUrl 和 redirectStatusCode
		if req.RedirectURL != nil && *req.RedirectURL != "" {
			return httpx.ErrParamInvalid("redirectUrl must be empty for group mode")
		}
		if req.RedirectStatusCode != nil && *req.RedirectStatusCode != 0 {
			return httpx.ErrParamInvalid("redirectStatusCode must be empty for group mode")
		}
	case model.OriginModeManual:
		// manual 模式必须有 originSetId
		if req.OriginSetID == nil || *req.OriginSetID <= 0 {
			return httpx.ErrParamMissing("originSetId is required for manual mode")
		}
		// 禁止 redirectUrl 和 redirectStatusCode
		if req.RedirectURL != nil && *req.RedirectURL != "" {
			return httpx.ErrParamInvalid("redirectUrl must be empty for manual mode")
		}
		if req.RedirectStatusCode != nil && *req.RedirectStatusCode != 0 {
			return httpx.ErrParamInvalid("redirectStatusCode must be empty for manual mode")
		}
	case model.OriginModeRedirect:
		// redirect 模式必须有 redirectUrl
		if req.RedirectURL == nil || *req.RedirectURL == "" {
			return httpx.ErrParamMissing("redirectUrl is required for redirect mode")
		}
		// redirectStatusCode 只允许 301 或 302
		if req.RedirectStatusCode != nil {
			if *req.RedirectStatusCode != 301 && *req.RedirectStatusCode != 302 {
				return httpx.ErrParamInvalid("redirectStatusCode must be 301 or 302")
			}
		}
		// 禁止 originGroupId 和 originSetId
		if req.OriginGroupID != nil && *req.OriginGroupID != 0 {
			return httpx.ErrParamInvalid("originGroupId must be empty for redirect mode")
		}
		if req.OriginSetID != nil && *req.OriginSetID != 0 {
			return httpx.ErrParamInvalid("originSetId must be empty for redirect mode")
		}
	}

	return nil
}

// handleHTTPSConfig 处理 HTTPS 配置
func (h *Handler) handleHTTPSConfig(tx *gorm.DB, websiteID int, httpsEnabled *bool, forceRedirect *bool, domains []string) error {
	// 如果未传 httpsEnabled，不处理
	if httpsEnabled == nil {
		return nil
	}

	// 查找或创建 website_https 记录
	var websiteHTTPS model.WebsiteHTTPS
	err := tx.Where("website_id = ?", websiteID).First(&websiteHTTPS).Error
	
	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		if !*httpsEnabled {
			// httpsEnabled=false，创建禁用状态的记录
			// 使用 map 确保 NULL 值正确写入
			return tx.Exec(
				"INSERT INTO website_https (website_id, enabled, force_redirect, hsts, cert_mode, certificate_id, acme_provider_id, acme_account_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NULL, NULL, NULL, NOW(), NOW())",
				websiteID, false, false, false, model.CertModeACME,
			).Error
		}

		// httpsEnabled=true，创建启用状态的记录
		forceRedir := false
		if forceRedirect != nil {
			forceRedir = *forceRedirect
		}

		// 获取默认 ACME provider 和 account
		acmeProviderID, acmeAccountID, err := h.getDefaultACME(tx)
		if err != nil {
			return err
		}

		// 使用 Exec 直接执行 SQL，确保 certificate_id 为 NULL
		if err := tx.Exec(
			"INSERT INTO website_https (website_id, enabled, force_redirect, hsts, cert_mode, acme_provider_id, acme_account_id, certificate_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, NULL, NOW(), NOW())",
			websiteID, true, forceRedir, false, model.CertModeACME, acmeProviderID, acmeAccountID,
		).Error; err != nil {
			return err
		}

		// 触发证书申请
		return h.triggerCertificateRequest(tx, websiteID, acmeAccountID, domains)
	} else if err != nil {
		return err
	}

	// 更新现有记录
	updates := make(map[string]interface{})
	
	if !*httpsEnabled {
		// httpsEnabled=false，禁用 HTTPS
		updates["enabled"] = false
		updates["force_redirect"] = false
		updates["certificate_id"] = nil
	} else {
		// httpsEnabled=true，启用 HTTPS
		updates["enabled"] = true
		if forceRedirect != nil {
			updates["force_redirect"] = *forceRedirect
		} else {
			updates["force_redirect"] = false
		}
		updates["hsts"] = false
		updates["cert_mode"] = model.CertModeACME
		updates["certificate_id"] = nil

		// 如果之前没有设置 ACME，设置默认值
		if websiteHTTPS.ACMEProviderID == nil || websiteHTTPS.ACMEAccountID == nil {
			acmeProviderID, acmeAccountID, err := h.getDefaultACME(tx)
			if err != nil {
				return err
			}
			updates["acme_provider_id"] = acmeProviderID
			updates["acme_account_id"] = acmeAccountID
			// 触发证书申请
			if err := h.triggerCertificateRequest(tx, websiteID, acmeAccountID, domains); err != nil {
				return err
			}
		} else {
			// 触发证书申请
			if err := h.triggerCertificateRequest(tx, websiteID, int(*websiteHTTPS.ACMEAccountID), domains); err != nil {
				return err
			}
		}
	}

	return tx.Model(&model.WebsiteHTTPS{}).Where("website_id = ?", websiteID).Updates(updates).Error
}

// getDefaultACME 获取默认的 ACME provider 和 account
func (h *Handler) getDefaultACME(tx *gorm.DB) (int, int, error) {
	// 查找第一个 active 的 provider
	var provider model.AcmeProvider
	if err := tx.Where("status = ?", "active").First(&provider).Error; err != nil {
		return 0, 0, httpx.ErrNotFound("no active ACME provider found")
	}

	// 查找该 provider 下第一个 active 的 account
	var account model.AcmeAccount
	if err := tx.Where("provider_id = ? AND status = ?", provider.ID, "active").First(&account).Error; err != nil {
		return 0, 0, httpx.ErrNotFound("no active ACME account found")
	}

	return int(provider.ID), int(account.ID), nil
}

// triggerCertificateRequest 触发证书申请（幂等）
func (h *Handler) triggerCertificateRequest(tx *gorm.DB, websiteID int, acmeAccountID int, domains []string) error {
	if len(domains) == 0 {
		return nil
	}

	// 构建 domains JSON 字符串（排序后）
	domainsJSON := buildDomainsJSON(domains)

	// 检查是否已存在相同的证书请求（幂等性）
	var existingRequest model.CertificateRequest
	err := tx.Where("acme_account_id = ? AND domains_json = ? AND status IN (?)", 
		acmeAccountID, domainsJSON, []string{"pending", "running"}).
		First(&existingRequest).Error

	if err == nil {
		// 已存在相同的请求，跳过
		return nil
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	// 创建新的证书请求
	certRequest := model.CertificateRequest{
		AccountID:       acmeAccountID,
		Domains:         domainsJSON,
		Status:          model.CertificateRequestStatusPending,
		PollIntervalSec: 40,
		PollMaxAttempts: 10,
		Attempts:        0,
	}

	return tx.Create(&certRequest).Error
}

// buildDomainsJSON 构建 domains JSON 字符串
func buildDomainsJSON(domains []string) string {
	// 简单实现：直接拼接 JSON 数组
	result := "["
	for i, domain := range domains {
		if i > 0 {
			result += ","
		}
		result += "\"" + domain + "\""
	}
	result += "]"
	return result
}

// triggerWebsiteReleaseTask 触发网站发布任务
func (h *Handler) triggerWebsiteReleaseTask(websiteID int64) {
	// 异步触发，不阻塞主流程
	go func() {
		svc := service.NewWebsiteReleaseTaskService(h.db)
		_, _ = svc.CreateOrUpdateTask(websiteID)
	}()
}
