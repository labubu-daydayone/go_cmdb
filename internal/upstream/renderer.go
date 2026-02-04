package upstream

import (
	"encoding/json"
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

// OriginAddress 回源地址（从 snapshot_json 解析）
type OriginAddress struct {
	ID            int    `json:"id"`
	Role          string `json:"role"`
	Weight        int    `json:"weight"`
	Address       string `json:"address"`
	Enabled       bool   `json:"enabled"`
	Protocol      string `json:"protocol"`
	OriginGroupID int    `json:"origin_group_id"`
}

// SnapshotData 快照数据结构
type SnapshotData struct {
	Addresses     []OriginAddress `json:"addresses"`
	OriginGroupID int             `json:"originGroupId"`
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
	if err := r.db.Where("origin_set_id = ?", req.OriginSetID).
		Find(&items).Error; err != nil {
		return nil, httpx.ErrDatabaseError("failed to query origin set items", err)
	}

	if len(items) == 0 {
		return nil, httpx.ErrStateConflict("origin set has no items")
	}

	// 3. 解析 snapshot_json（只取第一个 item，因为一个 origin_set 对应一个 origin_group）
	var snapshot SnapshotData
	if err := json.Unmarshal([]byte(items[0].SnapshotJSON), &snapshot); err != nil {
		return nil, httpx.ErrInternalError("failed to parse snapshot json", err)
	}

	// 4. 过滤出 enabled=true 的地址
	var enabledAddresses []OriginAddress
	for _, addr := range snapshot.Addresses {
		if addr.Enabled {
			enabledAddresses = append(enabledAddresses, addr)
		}
	}

	// 5. 校验至少有一个 primary enabled
	hasPrimary := false
	for _, addr := range enabledAddresses {
		if addr.Role == "primary" {
			hasPrimary = true
			break
		}
	}
	if !hasPrimary {
		return nil, httpx.ErrStateConflict("no enabled primary origin")
	}

	// 6. 生成 upstreamKey
	upstreamKey := fmt.Sprintf("up_%d", req.OriginSetID)

	// 7. 渲染配置
	var conf strings.Builder
	conf.WriteString(fmt.Sprintf("upstream %s {\n", upstreamKey))

	for _, addr := range enabledAddresses {
		if addr.Role == "primary" {
			conf.WriteString(fmt.Sprintf("    server %s weight=%d;\n", addr.Address, addr.Weight))
		} else if addr.Role == "backup" {
			conf.WriteString(fmt.Sprintf("    server %s backup;\n", addr.Address))
		}
	}

	conf.WriteString("}\n")

	return &RenderResponse{
		UpstreamKey:  upstreamKey,
		UpstreamConf: conf.String(),
	}, nil
}
