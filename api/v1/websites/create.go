package websites

import (
	"database/sql"
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateRequest 创建请求
type CreateRequest struct {
	Text         string `json:"text" binding:"required"`
	LineGroupID  int    `json:"lineGroupId" binding:"required"`
	CacheRuleID  int    `json:"cacheRuleId"`
	OriginMode   string `json:"originMode" binding:"required,oneof=group manual redirect"`
	OriginGroupID *int  `json:"originGroupId"`
	OriginSetID   *int  `json:"originSetId"`
	RedirectURL   *string `json:"redirectUrl"`
	RedirectStatusCode *int `json:"redirectStatusCode"`
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

	// 解析文本
	lines := parseText(req.Text)
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
			case "group":
				if req.OriginGroupID != nil && *req.OriginGroupID > 0 {
					website.OriginGroupID = sql.NullInt32{Int32: int32(*req.OriginGroupID), Valid: true}
				}
				if req.OriginSetID != nil && *req.OriginSetID > 0 {
					website.OriginSetID = sql.NullInt32{Int32: int32(*req.OriginSetID), Valid: true}
				}
			case "redirect":
				if req.RedirectURL != nil {
					website.RedirectURL = *req.RedirectURL
				}
				if req.RedirectStatusCode != nil {
					website.RedirectStatusCode = *req.RedirectStatusCode
				} else {
					website.RedirectStatusCode = 301
				}
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

			result.Created = true
			websiteID := website.ID
			result.WebsiteID = &websiteID
			return nil
		})

		if err != nil {
			errMsg := err.Error()
			result.Error = &errMsg
			result.Created = false
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
	case model.OriginModeRedirect:
		if req.RedirectURL == nil || *req.RedirectURL == "" {
			return httpx.ErrParamMissing("redirectUrl is required for redirect mode")
		}
	}

	return nil
}
