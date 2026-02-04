package upstream

import (
	"fmt"
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"
	"strings"

	"gorm.io/gorm"
)

// Renderer upstream 渲染器
type Renderer struct {
	db *gorm.DB
}

// NewRenderer 创建渲染器
func NewRenderer(db *gorm.DB) *Renderer {
	return &Renderer{db: db}
}

// RenderRequest 渲染请求
type RenderRequest struct {
	OriginSetID int64 `json:"originSetId"`
	WebsiteID   int64 `json:"websiteId"`
}

// RenderResponse 渲染响应
type RenderResponse struct {
	UpstreamKey  string `json:"upstreamKey"`
	UpstreamConf string `json:"upstreamConf"`
}

// Render 渲染 upstream 配置
func (r *Renderer) Render(req *RenderRequest) (*RenderResponse, error) {
	// 1. 查询 origin_set
	var originSet model.OriginSet
	if err := r.db.First(&originSet, req.OriginSetID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, httpx.ErrNotFound("origin set not found")
		}
		return nil, httpx.ErrDatabaseError("failed to query origin set", err)
	}

	// 2. 查询 origin_set_items
	var items []model.OriginSetItem
	if err := r.db.Where("origin_set_id = ? AND enabled = ?", req.OriginSetID, true).
		Order("role ASC, weight DESC").
		Find(&items).Error; err != nil {
		return nil, httpx.ErrDatabaseError("failed to query origin set items", err)
	}

	// 3. 校验至少有一个 primary enabled
	hasPrimary := false
	for _, item := range items {
		if item.Role == "primary" && item.Enabled {
			hasPrimary = true
			break
		}
	}
	if !hasPrimary {
		return nil, httpx.ErrStateConflict("no enabled primary origin")
	}

	// 4. 生成 upstreamKey
	upstreamKey := fmt.Sprintf("up_%d", req.OriginSetID)

	// 5. 渲染配置
	var conf strings.Builder
	conf.WriteString(fmt.Sprintf("upstream %s {\n", upstreamKey))

	for _, item := range items {
		if item.Role == "primary" {
			conf.WriteString(fmt.Sprintf("    server %s weight=%d;\n", item.Address, item.Weight))
		} else if item.Role == "backup" {
			conf.WriteString(fmt.Sprintf("    server %s backup;\n", item.Address))
		}
	}

	conf.WriteString("}\n")

	return &RenderResponse{
		UpstreamKey:  upstreamKey,
		UpstreamConf: conf.String(),
	}, nil
}
