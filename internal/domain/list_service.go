package domain

import (
	"context"
	"encoding/json"
	"fmt"

	"go_cmdb/internal/db"
)

// DomainListItem represents a domain item in the list
type DomainListItem struct {
	ID          int64       `json:"id"`
	Domain      string      `json:"domain"`
	Purpose     string      `json:"purpose"`
	Status      string      `json:"status"`
	Provider    *string     `json:"provider"`
	APIKey      *APIKeyInfo `json:"apiKey"`
	NameServers []string    `json:"nameServers"`
	LastSyncAt  *string     `json:"lastSyncAt"`
	CreatedAt   string      `json:"createdAt"`
}

// APIKeyInfo represents API key information
type APIKeyInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// DomainListResult represents the result of domain list query
type DomainListResult struct {
	Items    []DomainListItem `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
}

// ListDomainsParams represents the parameters for listing domains
type ListDomainsParams struct {
	Page      int
	PageSize  int
	Keyword   string
	Purpose   string
	Provider  string
	APIKeyID  *int64
	Status    string
}

// ListDomains queries domains with aggregated information
func ListDomains(ctx context.Context, params ListDomainsParams) (*DomainListResult, error) {
	// Validate and set defaults
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	// Build query
	query := db.DB.Table("domains d").
		Select(`
			d.id,
			d.domain,
			d.purpose,
			d.status,
			p.provider,
			ak.id as api_key_id,
			ak.name as api_key_name,
			zm.name_servers_json,
			zm.last_sync_at,
			d.created_at
		`).
		Joins("LEFT JOIN domain_dns_providers p ON p.domain_id = d.id").
		Joins("LEFT JOIN api_keys ak ON ak.id = p.api_key_id").
		Joins("LEFT JOIN domain_dns_zone_meta zm ON zm.domain_id = d.id")

	// Apply filters
	if params.Keyword != "" {
		query = query.Where("d.domain LIKE ?", "%"+params.Keyword+"%")
	}
	if params.Purpose != "" {
		query = query.Where("d.purpose = ?", params.Purpose)
	}
	if params.Provider != "" {
		query = query.Where("p.provider = ?", params.Provider)
	}
	if params.APIKeyID != nil {
		query = query.Where("p.api_key_id = ?", *params.APIKeyID)
	}
	if params.Status != "" {
		query = query.Where("d.status = ?", params.Status)
	}

	// Count total
	var total int64
	countQuery := db.DB.Table("domains d")
	if params.Keyword != "" {
		countQuery = countQuery.Where("d.domain LIKE ?", "%"+params.Keyword+"%")
	}
	if params.Purpose != "" {
		countQuery = countQuery.Where("d.purpose = ?", params.Purpose)
	}
	if params.Status != "" {
		countQuery = countQuery.Where("d.status = ?", params.Status)
	}
	// For provider and apiKeyId filters, need to join
	if params.Provider != "" || params.APIKeyID != nil {
		countQuery = countQuery.Joins("LEFT JOIN domain_dns_providers p ON p.domain_id = d.id")
		if params.Provider != "" {
			countQuery = countQuery.Where("p.provider = ?", params.Provider)
		}
		if params.APIKeyID != nil {
			countQuery = countQuery.Where("p.api_key_id = ?", *params.APIKeyID)
		}
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count domains: %w", err)
	}

	// Query with pagination
	offset := (params.Page - 1) * params.PageSize
	query = query.Order("d.id DESC").Limit(params.PageSize).Offset(offset)

	type QueryRow struct {
		ID              int64   `gorm:"column:id"`
		Domain          string  `gorm:"column:domain"`
		Purpose         string  `gorm:"column:purpose"`
		Status          string  `gorm:"column:status"`
		Provider        *string `gorm:"column:provider"`
		APIKeyID        *int64  `gorm:"column:api_key_id"`
		APIKeyName      *string `gorm:"column:api_key_name"`
		NameServersJSON *string `gorm:"column:name_servers_json"`
		LastSyncAt      *string `gorm:"column:last_sync_at"`
		CreatedAt       string  `gorm:"column:created_at"`
	}

	var rows []QueryRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to query domains: %w", err)
	}

	// Build result
	list := make([]DomainListItem, 0, len(rows))
	for _, row := range rows {
		item := DomainListItem{
			ID:         row.ID,
			Domain:     row.Domain,
			Purpose:    row.Purpose,
			Status:     row.Status,
			Provider:   row.Provider,
			LastSyncAt: row.LastSyncAt,
			CreatedAt:  row.CreatedAt,
		}

		// Parse API Key
		if row.APIKeyID != nil && row.APIKeyName != nil {
			item.APIKey = &APIKeyInfo{
				ID:   *row.APIKeyID,
				Name: *row.APIKeyName,
			}
		}

		// Parse Name Servers
		if row.NameServersJSON != nil && *row.NameServersJSON != "" {
			var ns []string
			if err := json.Unmarshal([]byte(*row.NameServersJSON), &ns); err == nil {
				item.NameServers = ns
			} else {
				// If unmarshal fails, set empty array
				item.NameServers = []string{}
			}
		} else {
			item.NameServers = []string{}
		}

		list = append(list, item)
	}

	return &DomainListResult{
		Items:    list,
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}
