package websites

import (
	"database/sql"
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateNew 创建网站（新版本）
func (h *Handler) CreateNew(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request body"))
		return
	}

	// 参数校验
	if err := h.validateCreateRequestNew(&req); err != nil {
		httpx.FailErr(c, err)
		return
	}

	// 事务处理
	var website model.Website
	err := h.db.Transaction(func(tx *gorm.DB) error {
		// 1. 检查 domain 是否已存在
		var count int64
		if err := tx.Model(&model.Website{}).Where("domain = ?", req.Domain).Count(&count).Error; err != nil {
			return httpx.ErrDatabaseError("failed to check domain", err)
		}
		if count > 0 {
			return httpx.ErrAlreadyExists("domain already exists")
		}

		// 2. 查询 line_group
		var lineGroup model.LineGroup
		if err := tx.First(&lineGroup, req.LineGroupID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return httpx.ErrNotFound("line group not found")
			}
			return httpx.ErrDatabaseError("failed to query line group", err)
		}

		// 3. 创建 website
		website = model.Website{
			Domain:      req.Domain,
			LineGroupID: req.LineGroupID,
			CacheRuleID: req.CacheRuleID,
			OriginMode:  req.OriginMode,
			Status:      model.WebsiteStatusActive,
		}

		// 4. 根据 originMode 设置字段
		switch req.OriginMode {
		case model.OriginModeGroup:
			// group 模式：必须提供 originGroupId 和 originSetId
			if req.OriginGroupID == nil || *req.OriginGroupID <= 0 {
				return httpx.ErrParamMissing("originGroupId is required for group mode")
			}
			if req.OriginSetID == nil || *req.OriginSetID <= 0 {
				return httpx.ErrParamMissing("originSetId is required for group mode")
			}
			website.OriginGroupID = sql.NullInt32{Int32: int32(*req.OriginGroupID), Valid: true}
			website.OriginSetID = sql.NullInt32{Int32: int32(*req.OriginSetID), Valid: true}

		case model.OriginModeManual:
			// manual 模式：originGroupId 和 originSetId 都为 NULL
			// 保持默认值即可

		case model.OriginModeRedirect:
			// redirect 模式：必须提供 redirectUrl
			if req.RedirectURL == nil || *req.RedirectURL == "" {
				return httpx.ErrParamMissing("redirectUrl is required for redirect mode")
			}
			website.RedirectURL = *req.RedirectURL
			if req.RedirectStatusCode != nil {
				website.RedirectStatusCode = *req.RedirectStatusCode
			} else {
				website.RedirectStatusCode = 301 // 默认值
			}
		}

		// 5. 创建记录（使用 Omit 排除未设置的字段）
		var createDB *gorm.DB
		if req.OriginMode == model.OriginModeGroup {
			// group 模式：两个字段都有值，正常创建
			createDB = tx
		} else {
			// manual/redirect 模式：排除 origin_group_id 和 origin_set_id
			createDB = tx.Omit("origin_group_id", "origin_set_id")
		}

		if err := createDB.Create(&website).Error; err != nil {
			// 检查是否是唯一约束冲突
			if err.Error() == "Error 1062" || err.Error() == "UNIQUE constraint failed" {
				return httpx.ErrAlreadyExists("domain already exists")
			}
			return httpx.ErrDatabaseError("failed to create website", err)
		}

		return nil
	})

	if err != nil {
		httpx.FailErr(c, err)
		return
	}

	// 返回创建的网站信息
	httpx.OK(c, gin.H{
		"item": toWebsiteDTO(&website),
	})
}

// validateCreateRequestNew 验证创建请求
func (h *Handler) validateCreateRequestNew(req *CreateRequest) error {
	// domain 必填
	if req.Domain == "" {
		return httpx.ErrParamMissing("domain is required")
	}

	// originMode 必填
	if req.OriginMode == "" {
		return httpx.ErrParamMissing("originMode is required")
	}

	// 根据 originMode 验证参数
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

// toWebsiteDTO 转换为 DTO
func toWebsiteDTO(w *model.Website) map[string]interface{} {
	dto := map[string]interface{}{
		"id":          w.ID,
		"domain":      w.Domain,
		"lineGroupId": w.LineGroupID,
		"cacheRuleId": w.CacheRuleID,
		"originMode":  w.OriginMode,
		"status":      w.Status,
		"createdAt":   w.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"updatedAt":   w.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// 处理可空字段
	if w.OriginGroupID.Valid {
		dto["originGroupId"] = int(w.OriginGroupID.Int32)
	} else {
		dto["originGroupId"] = nil
	}

	if w.OriginSetID.Valid {
		dto["originSetId"] = int(w.OriginSetID.Int32)
	} else {
		dto["originSetId"] = nil
	}

	if w.RedirectURL != "" {
		dto["redirectUrl"] = w.RedirectURL
	} else {
		dto["redirectUrl"] = nil
	}

	if w.RedirectStatusCode != 0 {
		dto["redirectStatusCode"] = w.RedirectStatusCode
	} else {
		dto["redirectStatusCode"] = nil
	}

	return dto
}
