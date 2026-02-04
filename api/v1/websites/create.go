package websites

import (
	"database/sql"
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateRequest 创建请求
type CreateRequest struct {
	Domain      string `json:"domain" binding:"required"` // 域名（必填且唯一）
	LineGroupID int    `json:"lineGroupId" binding:"required"`
	CacheRuleID int    `json:"cacheRuleId"`

	// 回源配置
	OriginMode    string `json:"originMode" binding:"required,oneof=group manual redirect"`
	OriginGroupID *int   `json:"originGroupId"` // group模式时必填
	OriginSetID   *int   `json:"originSetId"`   // group模式时必填

	// redirect配置
	RedirectURL        *string `json:"redirectUrl"`        // redirect模式时必填
	RedirectStatusCode *int    `json:"redirectStatusCode"` // redirect模式时可选，默认301
}

// CreateResponse 创建响应
type CreateResponse struct {
	Item WebsiteDetailItem `json:"item"`
}

// WebsiteDetailItem 网站详情项
type WebsiteDetailItem struct {
	ID                 int     `json:"id"`
	Domain             string  `json:"domain"`
	LineGroupID        int     `json:"lineGroupId"`
	CacheRuleID        int     `json:"cacheRuleId"`
	OriginMode         string  `json:"originMode"`
	OriginGroupID      *int    `json:"originGroupId"`
	OriginSetID        *int    `json:"originSetId"`
	RedirectURL        *string `json:"redirectUrl"`
	RedirectStatusCode *int    `json:"redirectStatusCode"`
	Status             string  `json:"status"`
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

	// 事务处理
	var createdWebsite model.Website
	err := h.db.Transaction(func(tx *gorm.DB) error {
		// 1. 检查 domain 是否已存在
		var existingCount int64
		if err := tx.Model(&model.Website{}).Where("domain = ?", req.Domain).Count(&existingCount).Error; err != nil {
			return httpx.ErrDatabaseError("failed to check domain", err)
		}
		if existingCount > 0 {
			return httpx.ErrAlreadyExists("domain already exists")
		}

		// 2. 创建 website
		website := model.Website{
			Domain:      req.Domain,
			LineGroupID: req.LineGroupID,
			CacheRuleID: req.CacheRuleID,
			OriginMode:  req.OriginMode,
			Status:      model.WebsiteStatusActive,
		}

		// 根据 originMode 设置字段
		switch req.OriginMode {
		case "group":
			// group 模式：设置 originGroupID 和 originSetID
			if req.OriginGroupID != nil && *req.OriginGroupID > 0 {
				website.OriginGroupID = sql.NullInt32{Int32: int32(*req.OriginGroupID), Valid: true}
			}
			if req.OriginSetID != nil && *req.OriginSetID > 0 {
				website.OriginSetID = sql.NullInt32{Int32: int32(*req.OriginSetID), Valid: true}
			}
		case "manual":
			// manual 模式：originGroupID 和 originSetID 保持 NULL
			// 不设置，保持默认值（sql.NullInt32 零值的 Valid=false）
		case "redirect":
			// redirect 模式：设置 redirectUrl
			if req.RedirectURL != nil {
				website.RedirectURL = *req.RedirectURL
			}
			if req.RedirectStatusCode != nil {
				website.RedirectStatusCode = *req.RedirectStatusCode
			} else {
				website.RedirectStatusCode = 301 // 默认301
			}
		}

		// 创建 website（不写入 origin_group_id 和 origin_set_id 如果它们是 NULL）
		if err := tx.Create(&website).Error; err != nil {
			return httpx.ErrDatabaseError("failed to create website", err)
		}

		createdWebsite = website
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

	// 构造返回结果
	response := CreateResponse{
		Item: WebsiteDetailItem{
			ID:                 createdWebsite.ID,
			Domain:             createdWebsite.Domain,
			LineGroupID:        createdWebsite.LineGroupID,
			CacheRuleID:        createdWebsite.CacheRuleID,
			OriginMode:         createdWebsite.OriginMode,
			OriginGroupID:      nullInt32ToIntPtr(createdWebsite.OriginGroupID),
			OriginSetID:        nullInt32ToIntPtr(createdWebsite.OriginSetID),
			RedirectURL:        stringToPtr(createdWebsite.RedirectURL),
			RedirectStatusCode: intToPtr(createdWebsite.RedirectStatusCode),
			Status:             createdWebsite.Status,
		},
	}

	httpx.OK(c, response)
}

// validateCreateRequest 校验创建请求
func validateCreateRequest(req *CreateRequest) *httpx.AppError {
	// 校验 origin_mode
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

// 辅助函数
func nullInt32ToIntPtr(n sql.NullInt32) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int32)
	return &v
}

func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intToPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}
