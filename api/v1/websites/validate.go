package websites

import (
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// validateOriginReferences 校验 originGroupId 和 originSetId 的存在性和关联性
func validateOriginReferences(db *gorm.DB, req *CreateRequest) *httpx.AppError {
	switch req.OriginMode {
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
