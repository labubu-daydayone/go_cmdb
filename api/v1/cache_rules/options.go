package cache_rules

import (
	"strconv"

	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
)

// OptionsItemDTO options 接口返回项
type OptionsItemDTO struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	ItemCount int    `json:"itemCount"`
	UpdatedAt string `json:"updatedAt"`
}

// OptionsResponse options 接口响应
type OptionsResponse struct {
	Items []OptionsItemDTO `json:"items"`
}

// Options 缓存规则 options 接口（供下拉选择使用）
// GET /api/v1/cache-rules/options
func (h *Handler) Options(c *gin.Context) {
	// 解析查询参数
	q := c.Query("q")                       // 模糊搜索 name
	status := c.DefaultQuery("status", "active") // 默认 active
	limitStr := c.DefaultQuery("limit", "200")
	limit, _ := strconv.Atoi(limitStr)

	// limit 上限保护
	if limit < 1 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}

	// 构建查询
	query := h.db.Model(&model.CacheRule{})

	// status 筛选（enabled 字段映射到 status）
	// active -> enabled=1, inactive -> enabled=0, all -> 不筛
	if status == "active" {
		query = query.Where("enabled = ?", true)
	} else if status == "inactive" {
		query = query.Where("enabled = ?", false)
	}
	// status=all 时不添加筛选条件

	// 模糊搜索 name
	if q != "" {
		query = query.Where("name LIKE ?", "%"+q+"%")
	}

	// 查询列表
	var rules []model.CacheRule
	if err := query.
		Order("id DESC").
		Limit(limit).
		Find(&rules).Error; err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to list cache rules", err))
		return
	}

	// 查询每个规则组的规则项数量
	ruleIDs := make([]int, len(rules))
	for i, rule := range rules {
		ruleIDs[i] = rule.ID
	}

	type CountResult struct {
		CacheRuleID int `json:"cache_rule_id"`
		Count       int `json:"count"`
	}
	var counts []CountResult
	if len(ruleIDs) > 0 {
		if err := h.db.Model(&model.CacheRuleItem{}).
			Select("cache_rule_id, COUNT(*) as count").
			Where("cache_rule_id IN ?", ruleIDs).
			Group("cache_rule_id").
			Find(&counts).Error; err != nil {
			httpx.FailErr(c, httpx.ErrDatabaseError("failed to count rule items", err))
			return
		}
	}

	// 构建计数映射
	countMap := make(map[int]int)
	for _, c := range counts {
		countMap[c.CacheRuleID] = c.Count
	}

	// 构造响应
	items := make([]OptionsItemDTO, len(rules))
	for i, rule := range rules {
		// enabled 映射到 status
		status := "inactive"
		if rule.Enabled {
			status = "active"
		}

		items[i] = OptionsItemDTO{
			ID:        rule.ID,
			Name:      rule.Name,
			Status:    status,
			ItemCount: countMap[rule.ID],
			UpdatedAt: rule.UpdatedAt.Format("2006-01-02T15:04:05+08:00"),
		}
	}

	resp := OptionsResponse{
		Items: items,
	}

	httpx.OK(c, resp)
}
