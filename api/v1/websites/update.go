package websites

import (
	"database/sql"
	"fmt"
	"log"

	"go_cmdb/internal/cert"
	dnspkg "go_cmdb/internal/dns"
	"go_cmdb/internal/domainutil"
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"
	"go_cmdb/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// UpdateRequest 更新请求
type UpdateRequest struct {
	ID                 int     `json:"id" binding:"required"`
	DomainsText        *string `json:"domainsText"`
	LineGroupID        *int    `json:"lineGroupId"`
	CacheRuleID        *int    `json:"cacheRuleId"`
	OriginMode         *string `json:"originMode"`
	OriginGroupID      *int    `json:"originGroupId"`
	OriginSetID        *int    `json:"originSetId"`
	RedirectURL        *string `json:"redirectUrl"`
	RedirectStatusCode *int    `json:"redirectStatusCode"`
	HTTPSEnabled       *bool   `json:"httpsEnabled"`
	ForceHTTPSRedirect *bool   `json:"forceHttpsRedirect"`
}

// UpdateResultItem 更新结果
type UpdateResultItem struct {
	WebsiteDTO
	ReleaseTaskID         int    `json:"releaseTaskId"`
	TaskCreated           bool   `json:"taskCreated"`
	SkipReason            string `json:"skipReason"`
	DispatchTriggered     bool   `json:"dispatchTriggered"`
	TargetNodeCount       int    `json:"targetNodeCount"`
	CreatedAgentTaskCount int    `json:"createdAgentTaskCount"`
	SkippedAgentTaskCount int    `json:"skippedAgentTaskCount"`
	AgentTaskCountAfter   int    `json:"agentTaskCountAfter"`
	PayloadValid          bool   `json:"payloadValid"`
	PayloadInvalidReason  string `json:"payloadInvalidReason"`
}

// Update 更新网站
func (h *Handler) Update(c *gin.Context) {
	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request body"))
		return
	}

	// 查询现有网站
	var website model.Website
	if err := h.db.Preload("Domains").First(&website, req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("website not found"))
			return
		}
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query website", err))
		return
	}

	// 计算最终域名列表（用于证书决策和 DNS）
	var finalDomains []string
	if req.DomainsText != nil {
		lines := parseText(*req.DomainsText)
		if len(lines) == 0 {
			httpx.FailErr(c, httpx.ErrParamMissing("domains required"))
			return
		}
		if len(lines) > 1 {
			httpx.FailErr(c, httpx.ErrParamInvalid("update only supports single website"))
			return
		}
		// 规范化域名
		for _, d := range lines[0] {
			nd, err := domainutil.Normalize(d)
			if err != nil {
				httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
				return
			}
			finalDomains = append(finalDomains, nd)
		}
		// PSL + domains 表 active 校验
		if err := domainutil.ValidateWebsiteDomains(h.db, finalDomains); err != nil {
			httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
			return
		}
	} else {
		for _, d := range website.Domains {
			finalDomains = append(finalDomains, d.Domain)
		}
	}

	// 计算最终 lineGroupID
	finalLineGroupID := website.LineGroupID
	if req.LineGroupID != nil {
		finalLineGroupID = *req.LineGroupID
	}

	// [事务外] Step 1: 证书决策（如需要）
	var certDecision *cert.DecisionResult
	httpsRequested := req.HTTPSEnabled != nil && *req.HTTPSEnabled

	if httpsRequested && len(finalDomains) > 0 {
		log.Printf("[Update] Step 1: certificate decision for website %d, domains %v", req.ID, finalDomains)
		var err error
		certDecision, err = cert.DecideCertificateReadOnly(h.db, finalDomains)
		if err != nil {
			log.Printf("[Update] Step 1 failed: certificate decision error: %v", err)
			certDecision = &cert.DecisionResult{
				Downgraded:      true,
				DowngradeReason: "certificate decision failed: " + err.Error(),
			}
		}
		log.Printf("[Update] Step 1 result: certFound=%v certID=%d acmeNeeded=%v downgraded=%v",
			certDecision.CertFound, certDecision.CertificateID, certDecision.ACMENeeded, certDecision.Downgraded)
	}

	// [事务内] Step 2: 更新 website + domains + website_https（只写 DB）
	log.Printf("[Update] Step 2: begin transaction for website %d update", req.ID)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		updates := make(map[string]interface{})

		// 更新 lineGroupId
		if req.LineGroupID != nil {
			updates["line_group_id"] = *req.LineGroupID
		}

		// 更新 cacheRuleId
		if req.CacheRuleID != nil {
			if *req.CacheRuleID > 0 {
				updates["cache_rule_id"] = sql.NullInt32{Int32: int32(*req.CacheRuleID), Valid: true}
			} else {
				updates["cache_rule_id"] = sql.NullInt32{Valid: false}
			}
		}

		// 更新 redirectUrl 和 redirectStatusCode（允许单独更新）
		if req.RedirectURL != nil {
			updates["redirect_url"] = *req.RedirectURL
		}
		if req.RedirectStatusCode != nil {
			updates["redirect_status_code"] = *req.RedirectStatusCode
		}

		// 更新 originMode 及相关字段
		if req.OriginMode != nil {
			if *req.OriginMode != model.OriginModeGroup && *req.OriginMode != model.OriginModeManual && *req.OriginMode != model.OriginModeRedirect {
				return httpx.ErrParamInvalid("originMode must be group, manual or redirect")
			}
			if err := validateUpdateOriginMode(&req); err != nil {
				return err
			}
			if err := validateUpdateOriginReferences(tx, &req); err != nil {
				return err
			}

			updates["origin_mode"] = *req.OriginMode
			switch *req.OriginMode {
			case model.OriginModeGroup:
				updates["origin_group_id"] = *req.OriginGroupID
				updates["origin_set_id"] = *req.OriginSetID
				updates["redirect_url"] = ""
				updates["redirect_status_code"] = 0
			case model.OriginModeManual:
				updates["origin_set_id"] = *req.OriginSetID
				updates["origin_group_id"] = nil
				updates["redirect_url"] = ""
				updates["redirect_status_code"] = 0
			case model.OriginModeRedirect:
				updates["redirect_url"] = *req.RedirectURL
				if req.RedirectStatusCode != nil {
					updates["redirect_status_code"] = *req.RedirectStatusCode
				} else {
					updates["redirect_status_code"] = 301
				}
				updates["origin_group_id"] = nil
				updates["origin_set_id"] = nil
			}
		}

		// 更新 website
		if len(updates) > 0 {
			if err := tx.Model(&website).Updates(updates).Error; err != nil {
				return err
			}
		}

		// 更新域名
		if req.DomainsText != nil {
			// 检查域名是否已被其他网站使用
			for _, domain := range finalDomains {
				var count int64
				if err := tx.Model(&model.WebsiteDomain{}).
					Where("domain = ? AND website_id != ?", domain, website.ID).
					Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					return httpx.ErrParamInvalid("domain already exists: " + domain)
				}
			}

			// 删除旧域名
			if err := tx.Where("website_id = ?", website.ID).Delete(&model.WebsiteDomain{}).Error; err != nil {
				return err
			}

			// 创建新域名
			for idx, domain := range finalDomains {
				wd := model.WebsiteDomain{
					WebsiteID: website.ID,
					Domain:    domain,
					IsPrimary: idx == 0,
				}
				if err := tx.Create(&wd).Error; err != nil {
					return err
				}
			}
		}

		// 写入 website_https 记录（纯 DB 写入）
		if req.HTTPSEnabled != nil || req.ForceHTTPSRedirect != nil {
			var websiteHTTPS model.WebsiteHTTPS
			existErr := tx.Where("website_id = ?", website.ID).First(&websiteHTTPS).Error

			forceRedir := false
			if req.ForceHTTPSRedirect != nil {
				forceRedir = *req.ForceHTTPSRedirect
			}

			if req.HTTPSEnabled != nil && !*req.HTTPSEnabled {
				// HTTPS 禁用
				if existErr == gorm.ErrRecordNotFound {
					if err := tx.Exec(
						"INSERT INTO website_https (website_id, enabled, force_redirect, hsts, cert_mode, certificate_id, acme_provider_id, acme_account_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NULL, NULL, NULL, NOW(), NOW())",
						website.ID, false, false, false, model.CertModeACME,
					).Error; err != nil {
						return err
					}
				} else if existErr == nil {
					if err := tx.Model(&model.WebsiteHTTPS{}).Where("website_id = ?", website.ID).Updates(map[string]interface{}{
						"enabled":        false,
						"force_redirect": false,
						"certificate_id": nil,
					}).Error; err != nil {
						return err
					}
				} else {
					return existErr
				}
			} else if certDecision != nil && certDecision.Downgraded {
				// HTTPS 被降级
				log.Printf("[Update] Website %d: HTTPS explicitly downgraded - %s", website.ID, certDecision.DowngradeReason)
				if existErr == gorm.ErrRecordNotFound {
					if err := tx.Exec(
						"INSERT INTO website_https (website_id, enabled, force_redirect, hsts, cert_mode, certificate_id, acme_provider_id, acme_account_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NULL, NULL, NULL, NOW(), NOW())",
						website.ID, false, false, false, model.CertModeACME,
					).Error; err != nil {
						return err
					}
				} else if existErr == nil {
					if err := tx.Model(&model.WebsiteHTTPS{}).Where("website_id = ?", website.ID).Updates(map[string]interface{}{
						"enabled":        false,
						"force_redirect": false,
						"certificate_id": nil,
					}).Error; err != nil {
						return err
					}
				} else {
					return existErr
				}
			} else if certDecision != nil && certDecision.CertFound {
				// 找到已有证书
				if existErr == gorm.ErrRecordNotFound {
					if err := tx.Exec(
						"INSERT INTO website_https (website_id, enabled, force_redirect, hsts, cert_mode, certificate_id, acme_provider_id, acme_account_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, NULL, NULL, NOW(), NOW())",
						website.ID, true, forceRedir, false, model.CertModeSelect, certDecision.CertificateID,
					).Error; err != nil {
						return err
					}
				} else if existErr == nil {
					if err := tx.Model(&model.WebsiteHTTPS{}).Where("website_id = ?", website.ID).Updates(map[string]interface{}{
						"enabled":        true,
						"force_redirect": forceRedir,
						"cert_mode":      model.CertModeSelect,
						"certificate_id": certDecision.CertificateID,
					}).Error; err != nil {
						return err
					}
				} else {
					return existErr
				}
				// 创建/更新证书绑定
				var existingBinding model.CertificateBinding
				if err := tx.Where("website_id = ?", website.ID).First(&existingBinding).Error; err == gorm.ErrRecordNotFound {
					binding := model.CertificateBinding{
						CertificateID: certDecision.CertificateID,
						WebsiteID:     website.ID,
						Status:        model.CertificateBindingStatusActive,
					}
					if err := tx.Create(&binding).Error; err != nil {
						return err
					}
				} else if err == nil {
					if err := tx.Model(&existingBinding).Updates(map[string]interface{}{
						"certificate_id": certDecision.CertificateID,
						"status":         model.CertificateBindingStatusActive,
					}).Error; err != nil {
						return err
					}
				}
			} else if certDecision != nil && certDecision.ACMENeeded {
				// 需要 ACME 申请
				if existErr == gorm.ErrRecordNotFound {
					if err := tx.Exec(
						"INSERT INTO website_https (website_id, enabled, force_redirect, hsts, cert_mode, certificate_id, acme_provider_id, acme_account_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NULL, ?, ?, NOW(), NOW())",
						website.ID, true, forceRedir, false, model.CertModeACME, certDecision.ACMEProviderID, certDecision.ACMEAccountID,
					).Error; err != nil {
						return err
					}
				} else if existErr == nil {
					if err := tx.Model(&model.WebsiteHTTPS{}).Where("website_id = ?", website.ID).Updates(map[string]interface{}{
						"enabled":          true,
						"force_redirect":   forceRedir,
						"cert_mode":        model.CertModeACME,
						"certificate_id":   nil,
						"acme_provider_id": certDecision.ACMEProviderID,
						"acme_account_id":  certDecision.ACMEAccountID,
					}).Error; err != nil {
						return err
					}
				} else {
					return existErr
				}
			} else if req.ForceHTTPSRedirect != nil && req.HTTPSEnabled == nil {
				// 只更新 forceRedirect
				if existErr == nil {
					if err := tx.Model(&model.WebsiteHTTPS{}).Where("website_id = ?", website.ID).Updates(map[string]interface{}{
						"force_redirect": forceRedir,
					}).Error; err != nil {
						return err
					}
				}
			}
		}

		return nil
	})
	log.Printf("[Update] Step 2: transaction completed for website %d, err=%v", req.ID, err)

	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.FailErr(c, appErr)
		} else {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to update website", err))
		}
		return
	}

	// [事务外] Step 3: 触发 ACME 申请（如需要）
	if certDecision != nil && certDecision.ACMENeeded {
		log.Printf("[Update] Step 3: triggering ACME request for website %d, accountID=%d", req.ID, certDecision.ACMEAccountID)
		if err := cert.TriggerACMERequest(h.db, req.ID, certDecision.ACMEAccountID, finalDomains); err != nil {
			log.Printf("[Update] Step 3 failed: ACME request trigger error: %v", err)
		} else {
			log.Printf("[Update] Step 3 completed: ACME request created for website %d", req.ID)
		}
	}

	// [事务外] Step 4: DNS CNAME 补齐（无论证书状态如何）
	if len(finalDomains) > 0 && finalLineGroupID > 0 {
		log.Printf("[Update] Step 4: creating DNS CNAME records for website %d, domains=%v, lineGroupId=%d", req.ID, finalDomains, finalLineGroupID)
		if err := dnspkg.EnsureWebsiteDomainCNAMEs(h.db, req.ID, finalDomains, finalLineGroupID); err != nil {
			log.Printf("[Update] Step 4 failed: DNS CNAME creation error: %v", err)
		} else {
			log.Printf("[Update] Step 4 completed: DNS CNAME records created for website %d", req.ID)
		}
	}

	// [事务外] Step 5: 触发发布任务
	log.Printf("[Update] Step 5: creating release task for website %d", req.ID)
	releaseService := service.NewWebsiteReleaseService(h.db)
	traceID := fmt.Sprintf("website_update_%d", req.ID)
	releaseResult, releaseErr := releaseService.CreateWebsiteReleaseTaskWithDispatch(int64(req.ID), traceID)
	if releaseErr != nil {
		log.Printf("[Update] Step 5 failed: release task error: %v", releaseErr)
		httpx.FailErr(c, httpx.ErrInternalError("failed to create release task", releaseErr))
		return
	}
	log.Printf("[Update] Step 5 completed: releaseTaskID=%d", releaseResult.ReleaseTaskID)

	// 重新查询返回
	if err := h.db.
		Preload("LineGroup").
		Preload("OriginGroup").
		Preload("Domains").
		Preload("HTTPS").
		First(&website, req.ID).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to query website", err))
		return
	}

	// 转换为 DTO
	item := WebsiteDTO{
		ID:                 website.ID,
		LineGroupID:        website.LineGroupID,
		OriginMode:         website.OriginMode,
		RedirectURL:        website.RedirectURL,
		RedirectStatusCode: website.RedirectStatusCode,
		Status:             website.Status,
		CreatedAt:          website.CreatedAt,
		UpdatedAt:          website.UpdatedAt,
	}

	// CacheRuleID 处理
	if website.CacheRuleID.Valid {
		val := int(website.CacheRuleID.Int32)
		item.CacheRuleID = &val
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

	// 构造返回结果
	result := UpdateResultItem{
		WebsiteDTO: item,
	}

	// 填充发布任务信息
	if releaseResult != nil {
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

	httpx.OK(c, result)
}

// validateUpdateOriginMode 校验更新请求中的 originMode 字段组合
func validateUpdateOriginMode(req *UpdateRequest) *httpx.AppError {
	if req.OriginMode == nil {
		return nil
	}

	switch *req.OriginMode {
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

// validateUpdateOriginReferences 校验更新请求中的 originGroupId 和 originSetId 的存在性和关联性
func validateUpdateOriginReferences(db *gorm.DB, req *UpdateRequest) *httpx.AppError {
	if req.OriginMode == nil {
		return nil
	}

	switch *req.OriginMode {
	case model.OriginModeGroup:
		var originGroup model.OriginGroup
		if err := db.First(&originGroup, *req.OriginGroupID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return httpx.ErrParamInvalid("originGroupId does not exist")
			}
			return httpx.ErrDatabaseError("failed to query origin group", err)
		}

		var originSet model.OriginSet
		if err := db.First(&originSet, *req.OriginSetID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return httpx.ErrParamInvalid("originSetId does not exist")
			}
			return httpx.ErrDatabaseError("failed to query origin set", err)
		}

		if int(originSet.OriginGroupID) != *req.OriginGroupID {
			return httpx.ErrParamInvalid("originSetId does not belong to originGroupId")
		}

	case model.OriginModeManual:
		var originSet model.OriginSet
		if err := db.First(&originSet, *req.OriginSetID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return httpx.ErrParamInvalid("originSetId does not exist")
			}
			return httpx.ErrDatabaseError("failed to query origin set", err)
		}

		if req.OriginGroupID != nil && *req.OriginGroupID > 0 {
			var originGroup model.OriginGroup
			if err := db.First(&originGroup, *req.OriginGroupID).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return httpx.ErrParamInvalid("originGroupId does not exist")
				}
				return httpx.ErrDatabaseError("failed to query origin group", err)
			}
		}
	}

	return nil
}
