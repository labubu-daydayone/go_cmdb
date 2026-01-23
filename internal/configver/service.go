package configver

import (
	"fmt"
	"time"

	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// Service handles config version operations
type Service struct {
	db *gorm.DB
}

// NewService creates a new config version service
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// CreateVersion creates a new config version with database-generated version number
// Version is guaranteed to be globally unique and incrementing by using the database auto-increment ID
func (s *Service) CreateVersion(nodeID int, payload string, reason string) (*model.ConfigVersion, error) {
	// Step 1: Insert a record with version=0 (placeholder) to get auto-increment ID
	configVersion := &model.ConfigVersion{
		Version:   0, // Placeholder, will be updated to ID
		NodeID:    nodeID,
		Payload:   payload,
		Reason:    reason,
		Status:    model.ConfigVersionStatusPending,
		CreatedAt: time.Now(),
	}

	if err := s.db.Create(configVersion).Error; err != nil {
		return nil, fmt.Errorf("failed to create config version: %w", err)
	}

	// Step 2: Update version to be equal to the auto-increment ID
	// This guarantees global uniqueness and incrementing order
	if err := s.db.Model(configVersion).Update("version", configVersion.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to update version to ID: %w", err)
	}

	// Step 3: Reload to get the updated version value
	configVersion.Version = configVersion.ID

	return configVersion, nil
}

// GetLatestVersion gets the latest config version for a node
func (s *Service) GetLatestVersion(nodeID int) (*model.ConfigVersion, error) {
	var configVersion model.ConfigVersion
	if err := s.db.Where("node_id = ?", nodeID).
		Order("version DESC").
		First(&configVersion).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}

	return &configVersion, nil
}

// GetByVersion gets a config version by version number
func (s *Service) GetByVersion(version int64) (*model.ConfigVersion, error) {
	var configVersion model.ConfigVersion
	if err := s.db.Where("version = ?", version).First(&configVersion).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	return &configVersion, nil
}

// UpdateStatus updates the status of a config version
func (s *Service) UpdateStatus(version int64, status string, lastError ...string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == model.ConfigVersionStatusApplied {
		now := time.Now()
		updates["applied_at"] = &now
	}

	if status == model.ConfigVersionStatusFailed && len(lastError) > 0 {
		updates["last_error"] = lastError[0]
	}

	if err := s.db.Model(&model.ConfigVersion{}).
		Where("version = ?", version).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// ListVersions lists config versions with pagination
func (s *Service) ListVersions(nodeID *int, page, pageSize int) ([]model.ConfigVersion, int64, error) {
	query := s.db.Model(&model.ConfigVersion{})

	if nodeID != nil {
		query = query.Where("node_id = ?", *nodeID)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count versions: %w", err)
	}

	// Get list
	var versions []model.ConfigVersion
	offset := (page - 1) * pageSize
	if err := query.Order("version DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&versions).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list versions: %w", err)
	}

	return versions, total, nil
}
