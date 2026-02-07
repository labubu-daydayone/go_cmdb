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

// DeleteRequest 删除请求
type DeleteRequest struct {
	IDs []int `json:"ids" binding:"required,min=1"`
}

// DeleteResultItem 删除结果项
type DeleteResultItem struct {
	WebsiteID              int64  `json:"websiteId"`
	ReleaseTaskID          int64  `json:"releaseTaskId"`
	TaskCreated            bool   `json:"taskCreated"`
	DispatchTriggered      bool   `json:"dispatchTriggered"`
	TargetNodeCount        int    `json:"targetNodeCount"`
	CreatedAgentTaskCount  int    `json:"createdAgentTaskCount"`
	SkippedAgentTaskCount  int    `json:"skippedAgentTaskCount"`
	AgentTaskCountAfter    int    `json:"agentTaskCountAfter"`
	SkipReason             string `json:"skipReason"`
}

// Delete 删除网站
func (h *Handler) Delete(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid request body"))
		return
	}

	// 只支持单个网站删除（批量删除逻辑复杂，暂不支持）
	if len(req.IDs) != 1 {
		httpx.FailErr(c, httpx.ErrParamInvalid("only single website deletion is supported"))
		return
	}

	websiteID := int64(req.IDs[0])

	// 1. 删除前读取网站信息（用于创建发布任务）
	var website model.Website
	if err := h.db.First(&website, websiteID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.FailErr(c, httpx.ErrNotFound("website not found"))
		} else {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to query website", err))
		}
		return
	}

	// 读取域名列表
	var domainRecords []model.WebsiteDomain
	h.db.Where("website_id = ?", websiteID).Find(&domainRecords)
	domains := make([]string, 0, len(domainRecords))
	for _, d := range domainRecords {
		domains = append(domains, d.Domain)
	}

	// 2. 在事务中删除
	err := h.db.Transaction(func(tx *gorm.DB) error {
		// 删除 website_https
		if err := tx.Where("website_id = ?", websiteID).Delete(&model.WebsiteHTTPS{}).Error; err != nil {
			return err
		}

		// 删除 website_domains
		if err := tx.Where("website_id = ?", websiteID).Delete(&model.WebsiteDomain{}).Error; err != nil {
			return err
		}

		// 删除 website
		if err := tx.Delete(&model.Website{}, websiteID).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to delete website", err))
		return
	}

	// 3. 创建发布任务
	deleteInfo := &service.WebsiteDeleteInfo{
		WebsiteID:   websiteID,
		LineGroupID: int64(website.LineGroupID),
		Domains:     domains,
	}
	traceID := fmt.Sprintf("website_delete_%d", websiteID)

	releaseTaskID, err := service.CreateWebsiteDeleteReleaseTask(h.db, deleteInfo, traceID)
	if err != nil {
		log.Printf("[Website Delete] Failed to create release_task: %v", err)
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to create release task", err))
		return
	}

	// 4. 派发任务（使用专门的删除派发方法，不查询已删除的 website）
	dispatcher := service.NewAgentTaskDispatcher(h.db)
	dispatchResult, err := dispatcher.EnsureDispatchedForDelete(releaseTaskID, deleteInfo.WebsiteID, deleteInfo.LineGroupID, deleteInfo.Domains, traceID)
	if err != nil {
		log.Printf("[Website Delete] Failed to dispatch tasks: %v", err)
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to dispatch tasks", err))
		return
	}

	// 5. 返回结果
	result := DeleteResultItem{
		WebsiteID:              websiteID,
		ReleaseTaskID:          releaseTaskID,
		TaskCreated:            true,
		DispatchTriggered:      dispatchResult.TargetNodeCount > 0,
		TargetNodeCount:        dispatchResult.TargetNodeCount,
		CreatedAgentTaskCount:  dispatchResult.Created,
		SkippedAgentTaskCount:  dispatchResult.Skipped,
		AgentTaskCountAfter:    dispatchResult.AgentTaskCountAfter,
		SkipReason:             dispatchResult.ErrorMsg,
	}

	httpx.OK(c, gin.H{"item": result})
}
