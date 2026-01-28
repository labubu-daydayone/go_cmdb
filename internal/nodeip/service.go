package nodeip

import (
	"go_cmdb/internal/dto"
	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// Service handles node IP operations
type Service struct {
	db *gorm.DB
}

// NewService creates a new node IP service
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// ListNodeIPs retrieves node IPs with optional filtering by nodeId
func (s *Service) ListNodeIPs(nodeID *int) ([]dto.NodeIPItem, error) {
	var nodeIPs []model.NodeIP
	query := s.db.Model(&model.NodeIP{})

	if nodeID != nil {
		query = query.Where("node_id = ?", *nodeID)
	}

	if err := query.Order("id ASC").Find(&nodeIPs).Error; err != nil {
		return nil, err
	}

	// Convert to DTO
	items := make([]dto.NodeIPItem, len(nodeIPs))
	for i, ip := range nodeIPs {
		items[i] = dto.NodeIPItem{
			ID:        ip.ID,
			NodeID:    ip.NodeID,
			IP:        ip.IP,
			IPRole:    string(ip.IPType),
			Enabled:   ip.Enabled,
			Status:    string(ip.Status),
			CreatedAt: ip.CreatedAt,
			UpdatedAt: ip.UpdatedAt,
		}
	}

	return items, nil
}

// DisableNodeIPs disables node IPs by IDs
func (s *Service) DisableNodeIPs(ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	return s.db.Model(&model.NodeIP{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"enabled": false,
			"status":  model.NodeIPStatusDisabled,
		}).Error
}

// EnableNodeIPs enables node IPs by IDs
func (s *Service) EnableNodeIPs(ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	return s.db.Model(&model.NodeIP{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"enabled": true,
			"status":  model.NodeIPStatusActive,
		}).Error
}

// CheckIPsExist checks if all IDs exist
func (s *Service) CheckIPsExist(ids []int) ([]int, error) {
	var existingIDs []int
	if err := s.db.Model(&model.NodeIP{}).
		Where("id IN ?", ids).
		Pluck("id", &existingIDs).Error; err != nil {
		return nil, err
	}
	return existingIDs, nil
}
