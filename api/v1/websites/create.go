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

		// [事务外] Step 1: 域名授权校验（PSL + domains 表 active 检查）
		log.Printf("[Create] Step 1: domain authorization check for %v", domains)
		if err := domainutil.ValidateWebsiteDomains(h.db, domains); err != nil {
			httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
			return
		}

		result := CreateResultItem{
			Line:    lineNum,
			Domains: domains,
		}

		// [事务外] Step 2: 证书决策（查找已有证书 / 判断是否需要 ACME）
		// 注意：这里只做查询和决策，不写入任何数据
		var certDecision *cert.DecisionResult
		httpsRequested := req.HTTPSEnabled != nil && *req.HTTPSEnabled

		if httpsRequested {
			log.Printf("[Create] Step 2: certificate decision for domains %v", domains)
			var err error
			certDecision, err = cert.DecideCertificateReadOnly(h.db, domains)
			if err != nil {
				log.Printf("[Create] Step 2 failed: certificate decision error: %v", err)
				// 证书决策失败不阻塞创建，降级为 HTTPS=false
				certDecision = &cert.DecisionResult{
					Downgraded:      true,
					DowngradeReason: "certificate decision failed: " + err.Error(),
				}
			}
			log.Printf("[Create] Step 2 result: certFound=%v certID=%d acmeNeeded=%v downgraded=%v",
				certDecision.CertFound, certDecision.CertificateID, certDecision.ACMENeeded, certDecision.Downgraded)
		}

		// [事务内] Step 3: 创建 website + domains + website_https（只写 DB）
		log.Printf("[Create] Step 3: begin transaction for website creation")
		var websiteID int
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

			// 写入 website_https 记录（纯 DB 写入，不调用外部服务）
			if req.HTTPSEnabled != nil {
				forceRedir := false
				if req.ForceHTTPSRedirect != nil {
					forceRedir = *req.ForceHTTPSRedirect
				}

				if !*req.HTTPSEnabled || (certDecision != nil && certDecision.Downgraded) {
					// HTTPS 禁用或被降级
					if err := tx.Exec(
						"INSERT INTO website_https (website_id, enabled, force_redirect, hsts, cert_mode, certificate_id, acme_provider_id, acme_account_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NULL, NULL, NULL, NOW(), NOW())",
						website.ID, false, false, false, model.CertModeACME,
					).Error; err != nil {
						return err
					}
				} else if certDecision != nil && certDecision.CertFound {
					// 找到已有证书
					if err := tx.Exec(
						"INSERT INTO website_https (website_id, enabled, force_redirect, hsts, cert_mode, certificate_id, acme_provider_id, acme_account_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NULL, NULL, NOW(), NOW())",
						website.ID, true, forceRedir, false, model.CertModeSelect, certDecision.CertificateID,
					).Error; err != nil {
						return err
					}
					// 创建证书绑定
					binding := model.CertificateBinding{
						CertificateID: certDecision.CertificateID,
						WebsiteID:     website.ID,
						Status:        model.CertificateBindingStatusActive,
					}
					if err := tx.Create(&binding).Error; err != nil {
						return err
					}
				} else if certDecision != nil && certDecision.ACMENeeded {
					// 需要 ACME 申请
					if err := tx.Exec(
						"INSERT INTO website_https (website_id, enabled, force_redirect, hsts, cert_mode, certificate_id, acme_provider_id, acme_account_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NULL, ?, ?, NOW(), NOW())",
						website.ID, true, forceRedir, false, model.CertModeACME, certDecision.ACMEProviderID, certDecision.ACMEAccountID,
					).Error; err != nil {
						return err
					}
				} else {
					// 默认：HTTPS 禁用
					if err := tx.Exec(
						"INSERT INTO website_https (website_id, enabled, force_redirect, hsts, cert_mode, certificate_id, acme_provider_id, acme_account_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NULL, NULL, NULL, NOW(), NOW())",
						website.ID, false, false, false, model.CertModeACME,
					).Error; err != nil {
						return err
					}
				}
			}

			result.Created = true
			websiteID = website.ID
			wid := website.ID
			result.WebsiteID = &wid
			return nil
		})
		log.Printf("[Create] Step 3: transaction completed, websiteID=%d, err=%v", websiteID, err)

		if err != nil {
			errMsg := err.Error()
			result.Error = &errMsg
			result.Created = false
			results = append(results, result)
			continue
		}

		// 如果事务内标记了错误（如域名已存在），跳过后续步骤
		if result.Error != nil {
			results = append(results, result)
			continue
		}

		// 设置证书决策结果到返回值
		if certDecision != nil {
			if certDecision.CertFound {
				result.CertDecision = "existing_cert"
				actualHTTPS := true
				result.HTTPSActual = &actualHTTPS
			} else if certDecision.ACMENeeded {
				result.CertDecision = "acme_triggered"
				actualHTTPS := true
				result.HTTPSActual = &actualHTTPS
			} else if certDecision.Downgraded {
				result.CertDecision = "downgraded"
				actualHTTPS := false
				result.HTTPSActual = &actualHTTPS
				log.Printf("[Create] Website %d: HTTPS explicitly downgraded - %s", websiteID, certDecision.DowngradeReason)
			}
		} else if req.HTTPSEnabled != nil {
			result.CertDecision = "none"
			actualHTTPS := *req.HTTPSEnabled
			result.HTTPSActual = &actualHTTPS
		}

		// [事务外] Step 4: 触发 ACME 申请（如需要）
		if certDecision != nil && certDecision.ACMENeeded {
			log.Printf("[Create] Step 4: triggering ACME request for website %d, accountID=%d", websiteID, certDecision.ACMEAccountID)
			if err := cert.TriggerACMERequest(h.db, websiteID, certDecision.ACMEAccountID, domains); err != nil {
				log.Printf("[Create] Step 4 failed: ACME request trigger error: %v", err)
				// ACME 触发失败不阻塞创建，但记录错误
			} else {
				log.Printf("[Create] Step 4 completed: ACME request created for website %d", websiteID)
			}
		}

		// [事务外] Step 5: 创建 DNS CNAME 记录（无论证书状态如何，只要网站创建成功就补齐 DNS）
		log.Printf("[Create] Step 5: creating DNS CNAME records for website %d, domains=%v, lineGroupId=%d", websiteID, domains, req.LineGroupID)
		if err := dnspkg.EnsureWebsiteDomainCNAMEs(h.db, websiteID, domains, req.LineGroupID); err != nil {
			log.Printf("[Create] Step 5 failed: DNS CNAME creation error: %v", err)
			// DNS 创建失败不阻塞网站创建
		} else {
			log.Printf("[Create] Step 5 completed: DNS CNAME records created for website %d", websiteID)
		}

		// [事务外] Step 6: 触发发布任务
		if result.WebsiteID != nil {
			log.Printf("[Create] Step 6: creating release task for website %d", *result.WebsiteID)
			releaseService := service.NewWebsiteReleaseService(h.db)
			traceID := fmt.Sprintf("website_create_%d", *result.WebsiteID)
			releaseResult, releaseErr := releaseService.CreateWebsiteReleaseTaskWithDispatch(int64(*result.WebsiteID), traceID)
			if releaseErr != nil {
				log.Printf("[Create] Step 6 failed: release task error: %v", releaseErr)
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
				log.Printf("[Create] Step 6 completed: releaseTaskID=%d", releaseResult.ReleaseTaskID)
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
