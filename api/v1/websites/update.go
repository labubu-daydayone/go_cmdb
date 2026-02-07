package websites

import (
	"fmt"
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"
	"go_cmdb/internal/service"
	"log"

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
	ReleaseTaskID          int    `json:"releaseTaskId"`
	TaskCreated            bool   `json:"taskCreated"`
	SkipReason             string `json:"skipReason"`
	DispatchTriggered      bool   `json:"dispatchTriggered"`
	TargetNodeCount        int    `json:"targetNodeCount"`
	CreatedAgentTaskCount  int    `json:"createdAgentTaskCount"`
	SkippedAgentTaskCount  int    `json:"skippedAgentTaskCount"`
	AgentTaskCountAfter    int    `json:"agentTaskCountAfter"`
	PayloadValid           bool   `json:"payloadValid"`
	PayloadInvalidReason   string `json:"payloadInvalidReason"`
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

	// 在事务中更新
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
			// 校验 originMode
			if *req.OriginMode != model.OriginModeGroup && *req.OriginMode != model.OriginModeManual && *req.OriginMode != model.OriginModeRedirect {
				return httpx.ErrParamInvalid("originMode must be group, manual or redirect")
			}

			// 校验字段组合
			if err := validateUpdateOriginMode(&req); err != nil {
				return err
			}

			// 校验存在性和关联性
			if err := validateUpdateOriginReferences(tx, &req); err != nil {
				return err
			}

			updates["origin_mode"] = *req.OriginMode

			// 根据 originMode 设置字段
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
			lines := parseText(*req.DomainsText)
			if len(lines) == 0 {
				return httpx.ErrParamMissing("domains required")
			}
			if len(lines) > 1 {
				return httpx.ErrParamInvalid("update only supports single website")
			}

			domains := lines[0]

			// 检查域名是否已被其他网站使用
			for _, domain := range domains {
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
		}

		// 处理 HTTPS 配置
		if req.HTTPSEnabled != nil || req.ForceHTTPSRedirect != nil {
			// 获取当前域名列表
			var currentDomains []string
			if req.DomainsText != nil {
				lines := parseText(*req.DomainsText)
				if len(lines) > 0 {
					currentDomains = lines[0]
				}
			} else {
				for _, d := range website.Domains {
					currentDomains = append(currentDomains, d.Domain)
				}
			}

			if err := h.handleHTTPSConfigUpdate(tx, website.ID, req.HTTPSEnabled, req.ForceHTTPSRedirect, currentDomains); err != nil {
				return err
			}
		}

		return nil
	})

		if err != nil {
			if appErr, ok := err.(*httpx.AppError); ok {
				httpx.FailErr(c, appErr)
			} else {
				httpx.FailErr(c, httpx.ErrDatabaseError("failed to update website", err))
			}
			return
		}

		// 更新成功后触发发布任务
		releaseService := service.NewWebsiteReleaseService(h.db)
		traceID := fmt.Sprintf("website_update_%d", req.ID)
		releaseResult, releaseErr := releaseService.CreateWebsiteReleaseTaskWithDispatch(int64(req.ID), traceID)
		if releaseErr != nil {
			log.Printf("[Update] Failed to create release task for website %d: %v", req.ID, releaseErr)
			// 发布任务创建失败，返回错误
			httpx.FailErr(c, httpx.ErrInternalError("failed to create release task", releaseErr))
			return
		}

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
		// 加载Domain信息以计算CNAME
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
		// 校验 originGroupId 存在
		var originGroup model.OriginGroup
		if err := db.First(&originGroup, *req.OriginGroupID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return httpx.ErrParamInvalid("originGroupId does not exist")
			}
			return httpx.ErrDatabaseError("failed to query origin group", err)
		}

		// 校验 originSetId 存在
		var originSet model.OriginSet
		if err := db.First(&originSet, *req.OriginSetID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return httpx.ErrParamInvalid("originSetId does not exist")
			}
			return httpx.ErrDatabaseError("failed to query origin set", err)
		}

		// 校验 originSetId 属于 originGroupId
		if int(originSet.OriginGroupID) != *req.OriginGroupID {
			return httpx.ErrParamInvalid("originSetId does not belong to originGroupId")
		}

	case model.OriginModeManual:
		// 校验 originSetId 存在
		var originSet model.OriginSet
		if err := db.First(&originSet, *req.OriginSetID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return httpx.ErrParamInvalid("originSetId does not exist")
			}
			return httpx.ErrDatabaseError("failed to query origin set", err)
		}

		// 如果传了 originGroupId，也要校验存在性
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
