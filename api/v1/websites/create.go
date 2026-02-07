package websites

import (
	"database/sql"
	"fmt"
	"go_cmdb/internal/cert"
	dnspkg "go_cmdb/internal/dns"
	"go_cmdb/internal/domainutil"
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"
	"go_cmdb/internal/service"
	"log"

	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateRequest 创建请求
type CreateRequest struct {
	DomainsText        string  `json:"domainsText" binding:"required"`
	LineGroupID        int     `json:"lineGroupId" binding:"required"`
	CacheRuleID        *int    `json:"cacheRuleId"`
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
	Line                  int      `json:"line"`
	Created               bool     `json:"created"`
	WebsiteID             *int     `json:"websiteId"`
	Domains               []string `json:"domains"`
	Error                 *string  `json:"error"`
	ReleaseTaskID         int      `json:"releaseTaskId"`
	TaskCreated           bool     `json:"taskCreated"`
	SkipReason            string   `json:"skipReason"`
	DispatchTriggered     bool     `json:"dispatchTriggered"`
	TargetNodeCount       int      `json:"targetNodeCount"`
	CreatedAgentTaskCount int      `json:"createdAgentTaskCount"`
	SkippedAgentTaskCount int      `json:"skippedAgentTaskCount"`
	AgentTaskCountAfter   int      `json:"agentTaskCountAfter"`
	PayloadValid          bool     `json:"payloadValid"`
	PayloadInvalidReason  string   `json:"payloadInvalidReason"`
	// 证书决策结果
	CertDecision string `json:"certDecision"` // existing_cert / acme_triggered / downgraded / none
	HTTPSActual  *bool  `json:"httpsActual"`  // 实际 HTTPS 状态（可能被降级）
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

		// 对每行域名进行规范化和 apex 校验
		normalizedDomains := make([]string, 0, len(domains))
		for _, d := range domains {
			nd, err := domainutil.Normalize(d)
			if err != nil {
				httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
				return
			}
			normalizedDomains = append(normalizedDomains, nd)
		}
		domains = normalizedDomains

		// 使用 PSL 计算 apex 并校验 domains 表中是否存在 active 记录
		if err := domainutil.ValidateWebsiteDomains(h.db, domains); err != nil {
			httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
			return
		}

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
				OriginMode:  req.OriginMode,
				Status:      model.WebsiteStatusActive,
			}

			// 处理 CacheRuleID
			if req.CacheRuleID != nil && *req.CacheRuleID > 0 {
				website.CacheRuleID = sql.NullInt32{Int32: int32(*req.CacheRuleID), Valid: true}
			} else {
				website.CacheRuleID = sql.NullInt32{Valid: false}
			}

			// 根据 originMode 设置字段
			switch req.OriginMode {
			case model.OriginModeGroup:
				website.OriginGroupID = sql.NullInt32{Int32: int32(*req.OriginGroupID), Valid: true}
				website.OriginSetID = sql.NullInt32{Int32: int32(*req.OriginSetID), Valid: true}
				website.RedirectURL = ""
				website.RedirectStatusCode = 0
			case model.OriginModeManual:
				website.OriginSetID = sql.NullInt32{Int32: int32(*req.OriginSetID), Valid: true}
				website.OriginGroupID = sql.NullInt32{Valid: false}
				website.RedirectURL = ""
				website.RedirectStatusCode = 0
			case model.OriginModeRedirect:
				website.RedirectURL = *req.RedirectURL
				if req.RedirectStatusCode != nil {
					website.RedirectStatusCode = *req.RedirectStatusCode
				} else {
					website.RedirectStatusCode = 301
				}
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

			// 处理 HTTPS 配置（含证书决策和降级逻辑）
			certDecisionStr, httpsActual, err := h.handleHTTPSConfig(tx, website.ID, req.HTTPSEnabled, req.ForceHTTPSRedirect, domains)
			if err != nil {
				return err
			}
			result.CertDecision = certDecisionStr
			result.HTTPSActual = httpsActual

			// 创建 DNS CNAME 记录（无论 HTTPS 状态如何）
			if err := dnspkg.EnsureWebsiteDomainCNAMEs(tx, website.ID, domains, req.LineGroupID); err != nil {
				log.Printf("[Create] Failed to create DNS CNAME records for website %d: %v", website.ID, err)
				// DNS 创建失败不阻塞网站创建
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
			// 创建成功后触发发布任务
			if result.WebsiteID != nil {
				releaseService := service.NewWebsiteReleaseService(h.db)
				traceID := fmt.Sprintf("website_create_%d", *result.WebsiteID)
				releaseResult, releaseErr := releaseService.CreateWebsiteReleaseTaskWithDispatch(int64(*result.WebsiteID), traceID)
				if releaseErr != nil {
					log.Printf("[Create] Failed to create release task for website %d: %v", *result.WebsiteID, releaseErr)
				} else {
					result.ReleaseTaskID = int(releaseResult.ReleaseTaskID)
					result.TaskCreated = releaseResult.TaskCreated
					result.SkipReason = releaseResult.SkipReason
					result.DispatchTriggered = releaseResult.DispatchTriggered
					result.TargetNodeCount = releaseResult.TargetNodeCount
					result.CreatedAgentTaskCount = releaseResult.CreatedAgentTaskCount
					result.SkippedAgentTaskCount = releaseResult.SkippedAgentTaskCount
					result.AgentTaskCountAfter = releaseResult.AgentTaskCountAfter
					result.PayloadValid = releaseResult.PayloadValid
					result.PayloadInvalidReason = releaseResult.PayloadInvalidReason
				}
			}
		}

		results = append(results, result)
	}

	httpx.OK(c, CreateResponse{Items: results})
}

// handleHTTPSConfig 处理 HTTPS 配置
// 返回: (certDecision, httpsActual, error)
// certDecision: "existing_cert" / "acme_triggered" / "downgraded" / "none"
// httpsActual: 实际 HTTPS 状态（可能被降级为 false）
func (h *Handler) handleHTTPSConfig(tx *gorm.DB, websiteID int, httpsEnabled *bool, forceRedirect *bool, domains []string) (string, *bool, error) {
	// 如果未传 httpsEnabled，不处理
	if httpsEnabled == nil {
		return "none", nil, nil
	}

	actualHTTPS := *httpsEnabled

	if !*httpsEnabled {
		// httpsEnabled=false，创建禁用状态的记录
		if err := tx.Exec(
			"INSERT INTO website_https (website_id, enabled, force_redirect, hsts, cert_mode, certificate_id, acme_provider_id, acme_account_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NULL, NULL, NULL, NOW(), NOW())",
			websiteID, false, false, false, model.CertModeACME,
		).Error; err != nil {
			return "", nil, err
		}
		return "none", &actualHTTPS, nil
	}

	// httpsEnabled=true，执行证书决策流程
	forceRedir := false
	if forceRedirect != nil {
		forceRedir = *forceRedirect
	}

	// 调用证书决策逻辑
	decision, err := cert.DecideCertificate(tx, websiteID, domains)
	if err != nil {
		return "", nil, err
	}

	certDecisionStr := "none"

	if decision.CertFound {
		// 找到已有证书，直接绑定
		certDecisionStr = "existing_cert"
		log.Printf("[Create] Website %d: found existing certificate %d", websiteID, decision.CertificateID)

		// 创建 website_https 记录（cert_mode=select）
		if err := tx.Exec(
			"INSERT INTO website_https (website_id, enabled, force_redirect, hsts, cert_mode, certificate_id, acme_provider_id, acme_account_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NULL, NULL, NOW(), NOW())",
			websiteID, true, forceRedir, false, model.CertModeSelect, decision.CertificateID,
		).Error; err != nil {
			return "", nil, err
		}

		// 创建证书绑定
		binding := model.CertificateBinding{
			CertificateID: decision.CertificateID,
			WebsiteID:     websiteID,
			Status:        model.CertificateBindingStatusActive,
		}
		if err := tx.Create(&binding).Error; err != nil {
			return "", nil, err
		}

	} else if decision.ACMETriggered {
		// 触发了 ACME 申请
		certDecisionStr = "acme_triggered"
		log.Printf("[Create] Website %d: ACME certificate request triggered", websiteID)

		// 创建 website_https 记录（cert_mode=acme，ACME provider/account 已由 decision 设置）
		// 先查询 ACME provider/account
		acmeProviderID, acmeAccountID, _ := getDefaultACMEIDs(tx)
		if err := tx.Exec(
			"INSERT INTO website_https (website_id, enabled, force_redirect, hsts, cert_mode, certificate_id, acme_provider_id, acme_account_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NULL, ?, ?, NOW(), NOW())",
			websiteID, true, forceRedir, false, model.CertModeACME, acmeProviderID, acmeAccountID,
		).Error; err != nil {
			return "", nil, err
		}

	} else if decision.Downgraded {
		// 降级：无证书、无 ACME
		certDecisionStr = "downgraded"
		actualHTTPS = false
		log.Printf("[Create] Website %d: HTTPS downgraded - %s", websiteID, decision.DowngradeReason)

		// 创建 website_https 记录（enabled=false）
		if err := tx.Exec(
			"INSERT INTO website_https (website_id, enabled, force_redirect, hsts, cert_mode, certificate_id, acme_provider_id, acme_account_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NULL, NULL, NULL, NOW(), NOW())",
			websiteID, false, false, false, model.CertModeACME,
		).Error; err != nil {
			return "", nil, err
		}
	}

	return certDecisionStr, &actualHTTPS, nil
}

// getDefaultACMEIDs 获取默认的 ACME provider 和 account ID
func getDefaultACMEIDs(tx *gorm.DB) (int, int, error) {
	var provider model.AcmeProvider
	if err := tx.Where("status = ?", "active").First(&provider).Error; err != nil {
		return 0, 0, err
	}
	var account model.AcmeAccount
	if err := tx.Where("provider_id = ? AND status = ?", provider.ID, "active").First(&account).Error; err != nil {
		return 0, 0, err
	}
	return int(provider.ID), int(account.ID), nil
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
		if req.OriginGroupID == nil || *req.OriginGroupID <= 0 {
			return httpx.ErrParamMissing("originGroupId is required for group mode")
		}
		if req.OriginSetID == nil || *req.OriginSetID <= 0 {
			return httpx.ErrParamMissing("originSetId is required for group mode")
		}
		if req.RedirectURL != nil && *req.RedirectURL != "" {
			return httpx.ErrParamInvalid("redirectUrl must be empty for group mode")
		}
		if req.RedirectStatusCode != nil && *req.RedirectStatusCode != 0 {
			return httpx.ErrParamInvalid("redirectStatusCode must be empty for group mode")
		}
	case model.OriginModeManual:
		if req.OriginSetID == nil || *req.OriginSetID <= 0 {
			return httpx.ErrParamMissing("originSetId is required for manual mode")
		}
		if req.RedirectURL != nil && *req.RedirectURL != "" {
			return httpx.ErrParamInvalid("redirectUrl must be empty for manual mode")
		}
		if req.RedirectStatusCode != nil && *req.RedirectStatusCode != 0 {
			return httpx.ErrParamInvalid("redirectStatusCode must be empty for manual mode")
		}
	case model.OriginModeRedirect:
		if req.RedirectURL == nil || *req.RedirectURL == "" {
			return httpx.ErrParamMissing("redirectUrl is required for redirect mode")
		}
		if req.RedirectStatusCode != nil {
			if *req.RedirectStatusCode != 301 && *req.RedirectStatusCode != 302 {
				return httpx.ErrParamInvalid("redirectStatusCode must be 301 or 302")
			}
		}
		if req.OriginGroupID != nil && *req.OriginGroupID != 0 {
			return httpx.ErrParamInvalid("originGroupId must be empty for redirect mode")
		}
		if req.OriginSetID != nil && *req.OriginSetID != 0 {
			return httpx.ErrParamInvalid("originSetId must be empty for redirect mode")
		}
	}

	return nil
}
