package api_keys

import (
	"context"
	"fmt"
	"go_cmdb/internal/db"
	"go_cmdb/internal/model"
	"strings"
)

// ListParams represents parameters for listing API keys
type ListParams struct {
	Page     int
	PageSize int
	Keyword  string
	Provider string
	Status   string
}

// ListResult represents the result of listing API keys
type ListResult struct {
	Items []APIKeyListItem `json:"items"`
	Total int64            `json:"total"`
	Page  int              `json:"page"`
	PageSize int           `json:"pageSize"`
}

// APIKeyListItem represents an API key in the list response
type APIKeyListItem struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Provider        string `json:"provider"`
	Account         string `json:"account"`
	APITokenMasked  string `json:"apiTokenMasked"`
	Status          string `json:"status"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

// CreateParams represents parameters for creating an API key
type CreateParams struct {
	Name     string
	Provider string
	Account  string
	APIToken string
}

// UpdateParams represents parameters for updating an API key
type UpdateParams struct {
	ID       int64
	Name     *string
	Account  *string
	APIToken *string
	Status   *string
}

// DeleteParams represents parameters for deleting API keys
type DeleteParams struct {
	IDs []int64
}

// ToggleStatusParams represents parameters for toggling API key status
type ToggleStatusParams struct {
	ID     int64
	Status string
}

// List returns a paginated list of API keys
func List(ctx context.Context, params ListParams) (*ListResult, error) {
	db := db.GetDB()

	// Build query
	query := db.Model(&model.APIKey{})

	// Apply filters
	if params.Keyword != "" {
		keyword := "%" + params.Keyword + "%"
		query = query.Where("name LIKE ? OR account LIKE ?", keyword, keyword)
	}

	if params.Provider != "" {
		query = query.Where("provider = ?", params.Provider)
	}

	if params.Status != "" && params.Status != "all" {
		query = query.Where("status = ?", params.Status)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count API keys: %w", err)
	}

	// Apply pagination
	offset := (params.Page - 1) * params.PageSize
	query = query.Offset(offset).Limit(params.PageSize)

	// Fetch records
	var apiKeys []model.APIKey
	if err := query.Order("id DESC").Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch API keys: %w", err)
	}

	// Convert to list items
	items := make([]APIKeyListItem, len(apiKeys))
	for i, key := range apiKeys {
		items[i] = APIKeyListItem{
			ID:             int64(key.ID),
			Name:           key.Name,
			Provider:       string(key.Provider),
			Account:        key.Account,
			APITokenMasked: key.MaskedToken(),
			Status:         string(key.Status),
			CreatedAt:      key.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:      key.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return &ListResult{
		Items:    items,
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

// Create creates a new API key
func Create(ctx context.Context, params CreateParams) error {
	// Validate provider
	if params.Provider != "cloudflare" {
		return fmt.Errorf("invalid provider: %s (only 'cloudflare' is supported)", params.Provider)
	}

	// Validate required fields
	if strings.TrimSpace(params.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(params.APIToken) == "" {
		return fmt.Errorf("apiToken is required")
	}

	db := db.GetDB()

	apiKey := model.APIKey{
		Name:     params.Name,
		Provider: model.APIKeyProvider(params.Provider),
		Account:  params.Account,
		APIToken: params.APIToken,
		Status:   model.APIKeyStatusActive,
	}

	if err := db.Create(&apiKey).Error; err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	return nil
}

// Update updates an existing API key
func Update(ctx context.Context, params UpdateParams) error {
	db := db.GetDB()

	// Check if API key exists
	var apiKey model.APIKey
	if err := db.First(&apiKey, params.ID).Error; err != nil {
		return fmt.Errorf("API key not found: %w", err)
	}

	// Build update map
	updates := make(map[string]interface{})

	if params.Name != nil {
		if strings.TrimSpace(*params.Name) == "" {
			return fmt.Errorf("name cannot be empty")
		}
		updates["name"] = *params.Name
	}

	if params.Account != nil {
		updates["account"] = *params.Account
	}

	if params.APIToken != nil {
		if strings.TrimSpace(*params.APIToken) == "" {
			return fmt.Errorf("apiToken cannot be empty")
		}
		updates["api_token"] = *params.APIToken
	}

	if params.Status != nil {
		if *params.Status != "active" && *params.Status != "inactive" {
			return fmt.Errorf("invalid status: %s", *params.Status)
		}
		// Check dependencies before changing status to inactive
		if *params.Status == "inactive" {
			count, err := CheckDependencies(ctx, params.ID)
			if err != nil {
				return fmt.Errorf("failed to check dependencies: %w", err)
			}
			if count > 0 {
				return fmt.Errorf("API Key is being used by %d domains, cannot disable", count)
			}
		}
		updates["status"] = *params.Status
	}

	// Apply updates
	if len(updates) > 0 {
		if err := db.Model(&apiKey).Updates(updates).Error; err != nil {
			return fmt.Errorf("failed to update API key: %w", err)
		}
	}

	return nil
}

// Delete deletes API keys by IDs
func Delete(ctx context.Context, params DeleteParams) error {
	if len(params.IDs) == 0 {
		return fmt.Errorf("no IDs provided")
	}

	db := db.GetDB()

	// Check dependencies for each ID
	for _, id := range params.IDs {
		count, err := CheckDependencies(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to check dependencies for API key %d: %w", id, err)
		}
		if count > 0 {
			return fmt.Errorf("API Key %d is being used by %d domains, cannot delete", id, count)
		}
	}

	// Delete all keys
	if err := db.Delete(&model.APIKey{}, params.IDs).Error; err != nil {
		return fmt.Errorf("failed to delete API keys: %w", err)
	}

	return nil
}

// ToggleStatus toggles the status of an API key
func ToggleStatus(ctx context.Context, params ToggleStatusParams) error {
	if params.Status != "active" && params.Status != "inactive" {
		return fmt.Errorf("invalid status: %s", params.Status)
	}

	db := db.GetDB()

	// Check if API key exists
	var apiKey model.APIKey
	if err := db.First(&apiKey, params.ID).Error; err != nil {
		return fmt.Errorf("API key not found: %w", err)
	}

	// Check dependencies before changing status to inactive
	if params.Status == "inactive" {
		count, err := CheckDependencies(ctx, params.ID)
		if err != nil {
			return fmt.Errorf("failed to check dependencies: %w", err)
		}
		if count > 0 {
			return fmt.Errorf("API Key is being used by %d domains, cannot disable", count)
		}
	}

	// Update status
	if err := db.Model(&apiKey).Update("status", params.Status).Error; err != nil {
		return fmt.Errorf("failed to update API key status: %w", err)
	}

	return nil
}

// CheckDependencies checks if an API key is being used by domain_dns_providers
func CheckDependencies(ctx context.Context, apiKeyID int64) (int64, error) {
	db := db.GetDB()

	var count int64
	if err := db.Table("domain_dns_providers").
		Where("api_key_id = ?", apiKeyID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count dependencies: %w", err)
	}

	return count, nil
}
