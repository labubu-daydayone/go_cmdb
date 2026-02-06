package websites

import (
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

	// 在事务中删除
	err := h.db.Transaction(func(tx *gorm.DB) error {
		for _, id := range req.IDs {
			// 删除 website_https
			if err := tx.Where("website_id = ?", id).Delete(&model.WebsiteHTTPS{}).Error; err != nil {
				return err
			}

			// 删除 website_domains
			if err := tx.Where("website_id = ?", id).Delete(&model.WebsiteDomain{}).Error; err != nil {
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
