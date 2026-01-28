package nodeip

import (
	"go_cmdb/internal/dto"
	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// Service handles node IP operations
type Service struct {
	db        *gorm.DB
	DNSLinker *DNSLinker
}

// NewService creates a new node IP service
func NewService(db *gorm.DB) *Service {
	return &Service{
		db:        db,
		DNSLinker: NewDNSLinker(db),
	}
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

// DisableNodeIPs disables node IPs by IDs and updates DNS records
func (s *Service) DisableNodeIPs(ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Disable IPs
	if err := tx.Model(&model.NodeIP{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"enabled": false,
			"status":  model.NodeIPStatusDisabled,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update DNS records to absent for each IP
	linker := NewDNSLinker(tx)
	for _, ipID := range ids {
		if err := linker.ApplyDesiredRecords(ipID, model.DNSRecordDesiredStateAbsent); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// EnableNodeIPs enables node IPs by IDs and updates DNS records
func (s *Service) EnableNodeIPs(ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	// Start transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Enable IPs
	if err := tx.Model(&model.NodeIP{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"enabled": true,
			"status":  model.NodeIPStatusActive,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Update DNS records to present for each IP
	linker := NewDNSLinker(tx)
	for _, ipID := range ids {
		if err := linker.ApplyDesiredRecords(ipID, model.DNSRecordDesiredStatePresent); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
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
