package renderer

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go_cmdb/internal/model"
	"strings"

	"gorm.io/gorm"
)

// WebsiteConfigRenderer 网站配置渲染器
type WebsiteConfigRenderer struct {
	db *gorm.DB
}

// NewWebsiteConfigRenderer 创建网站配置渲染器
func NewWebsiteConfigRenderer(db *gorm.DB) *WebsiteConfigRenderer {
	return &WebsiteConfigRenderer{db: db}
}

// RenderConfig 渲染网站配置
func (r *WebsiteConfigRenderer) RenderConfig(websiteID int) (*model.ReleaseTaskPayload, string, error) {
	// 查询网站信息
	var website model.Website
	if err := r.db.First(&website, websiteID).Error; err != nil {
		return nil, "", fmt.Errorf("failed to query website: %w", err)
	}

	// 查询域名列表
	var domains []model.WebsiteDomain
	if err := r.db.Where("website_id = ?", websiteID).Find(&domains).Error; err != nil {
		return nil, "", fmt.Errorf("failed to query domains: %w", err)
	}

	// 渲染 upstream 配置
	upstreamContent, err := r.renderUpstream(website)
	if err != nil {
		return nil, "", fmt.Errorf("failed to render upstream: %w", err)
	}

	// 渲染 server 配置
	serverContent, err := r.renderServer(website, domains)
	if err != nil {
		return nil, "", fmt.Errorf("failed to render server: %w", err)
	}

	// 计算 content hash
	contentHash := r.calculateContentHash(upstreamContent, serverContent)

	// 构造 payload
	payload := &model.ReleaseTaskPayload{
		WebsiteID: websiteID,
		Files: map[string]model.ReleaseTaskFileInfo{
			"upstream": {
				Path:    fmt.Sprintf("/data/vhost/upstream/ws-%d.conf", websiteID),
				Content: upstreamContent,
			},
			"server": {
				Path:    fmt.Sprintf("/data/vhost/server/ws-%d.conf", websiteID),
				Content: serverContent,
			},
		},
	}

	return payload, contentHash, nil
}

// renderUpstream 渲染 upstream 配置
func (r *WebsiteConfigRenderer) renderUpstream(website model.Website) (string, error) {
	// 如果没有 originSetId，返回空配置
	if !website.OriginSetID.Valid || website.OriginSetID.Int32 == 0 {
		return "", nil
	}

	// 查询 origin_set
	var originSet model.OriginSet
	if err := r.db.First(&originSet, website.OriginSetID.Int32).Error; err != nil {
		return "", fmt.Errorf("failed to query origin_set: %w", err)
	}

	// 查询 origin_set_items
	var items []model.OriginSetItem
	if err := r.db.Where("origin_set_id = ?", originSet.ID).Find(&items).Error; err != nil {
		return "", fmt.Errorf("failed to query origin_set_items: %w", err)
	}

	// 生成 upstream 配置
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("upstream ws_%d {\n", website.ID))

	// 从 snapshot_json 中解析地址
	for _, item := range items {
		var snapshot map[string]interface{}
		if err := json.Unmarshal([]byte(item.SnapshotJSON), &snapshot); err != nil {
			continue
		}
		if addresses, ok := snapshot["addresses"].([]interface{}); ok {
			for _, addr := range addresses {
				if addrMap, ok := addr.(map[string]interface{}); ok {
					ip := addrMap["ip"].(string)
					port := int(addrMap["port"].(float64))
					weight := 1
					if w, ok := addrMap["weight"].(float64); ok && w > 0 {
						weight = int(w)
					}
					sb.WriteString(fmt.Sprintf("    server %s:%d weight=%d max_fails=3 fail_timeout=30s;\n",
						ip, port, weight))
				}
			}
		}
	}

	sb.WriteString("    keepalive 32;\n")
	sb.WriteString("}\n")

	return sb.String(), nil
}

// renderServer 渲染 server 配置
func (r *WebsiteConfigRenderer) renderServer(website model.Website, domains []model.WebsiteDomain) (string, error) {
	// 生成 server_name 列表
	serverNames := make([]string, 0, len(domains))
	for _, d := range domains {
		serverNames = append(serverNames, d.Domain)
	}

	// 生成 server 配置
	var sb strings.Builder
	sb.WriteString("server {\n")
	sb.WriteString("    listen 80;\n")
	sb.WriteString(fmt.Sprintf("    server_name %s;\n", strings.Join(serverNames, " ")))
	sb.WriteString("\n")
	sb.WriteString("    location / {\n")

	// 根据 originMode 生成不同的配置
	switch website.OriginMode {
	case model.OriginModeGroup, model.OriginModeManual:
		if website.OriginSetID.Valid && website.OriginSetID.Int32 > 0 {
			sb.WriteString(fmt.Sprintf("        proxy_pass http://ws_%d;\n", website.ID))
			sb.WriteString("        proxy_set_header Host $host;\n")
			sb.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
			sb.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
		}
	case model.OriginModeRedirect:
		if website.RedirectURL != "" {
			statusCode := 301
			if website.RedirectStatusCode > 0 {
				statusCode = website.RedirectStatusCode
			}
			sb.WriteString(fmt.Sprintf("        return %d %s;\n", statusCode, website.RedirectURL))
		}
	}

	sb.WriteString("    }\n")
	sb.WriteString("}\n")

	return sb.String(), nil
}

// calculateContentHash 计算内容哈希
func (r *WebsiteConfigRenderer) calculateContentHash(upstreamContent, serverContent string) string {
	content := upstreamContent + "\n---\n" + serverContent
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}
