package nodeip

import (
	"fmt"
	"go_cmdb/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DNSLinker handles DNS record linkage for Node IP availability changes
type DNSLinker struct {
	db *gorm.DB
}

// NewDNSLinker creates a new DNS linker
func NewDNSLinker(db *gorm.DB) *DNSLinker {
	return &DNSLinker{db: db}
}

// DNSRecordPreview represents a preview of DNS records that would be affected
type DNSRecordPreview struct {
	DomainID      int    `json:"domainId"`
	Domain        string `json:"domain"`
	NodeGroupID   int    `json:"nodeGroupId"`
	NodeGroupName string `json:"nodeGroupName"`
	RecordType    string `json:"recordType"`
	RecordName    string `json:"recordName"`
	RecordValue   string `json:"recordValue"`
	DesiredState  string `json:"desiredState"`
}

// BuildDesiredRecords builds the list of DNS records that should be created/updated for an IP
func (l *DNSLinker) BuildDesiredRecords(ipID int, desiredState model.DNSRecordDesiredState) ([]DNSRecordPreview, error) {
	// Get the IP
	var nodeIP model.NodeIP
	if err := l.db.First(&nodeIP, ipID).Error; err != nil {
		return nil, fmt.Errorf("failed to find node IP: %w", err)
	}

	// Find all node groups that contain this IP
	var nodeGroupIPs []model.NodeGroupIP
	if err := l.db.Where("ip_id = ?", ipID).Find(&nodeGroupIPs).Error; err != nil {
		return nil, fmt.Errorf("failed to find node group IPs: %w", err)
	}

	if len(nodeGroupIPs) == 0 {
		// IP not in any node group, no DNS records to manage
		return []DNSRecordPreview{}, nil
	}

	// Get all node group IDs
	nodeGroupIDs := make([]int, len(nodeGroupIPs))
	for i, ngIP := range nodeGroupIPs {
		nodeGroupIDs[i] = ngIP.NodeGroupID
	}

	// Get all node groups
	var nodeGroups []model.NodeGroup
	if err := l.db.Where("id IN ?", nodeGroupIDs).Find(&nodeGroups).Error; err != nil {
		return nil, fmt.Errorf("failed to find node groups: %w", err)
	}

	// Get all CDN domains
	var cdnDomains []model.Domain
	if err := l.db.Where("purpose = ? AND status = ?", "cdn", "active").Find(&cdnDomains).Error; err != nil {
		return nil, fmt.Errorf("failed to find CDN domains: %w", err)
	}

	if len(cdnDomains) == 0 {
		// No CDN domains, no DNS records to manage
		return []DNSRecordPreview{}, nil
	}

	// Build preview list
	previews := []DNSRecordPreview{}
	for _, ng := range nodeGroups {
		for _, domain := range cdnDomains {
			previews = append(previews, DNSRecordPreview{
				DomainID:      domain.ID,
				Domain:        domain.Domain,
				NodeGroupID:   ng.ID,
				NodeGroupName: ng.Name,
				RecordType:    "A",
				RecordName:    ng.CNAMEPrefix,
				RecordValue:   nodeIP.IP,
				DesiredState:  string(desiredState),
			})
		}
	}

	return previews, nil
}

// ApplyDesiredRecords applies the desired state to all DNS records for an IP
func (l *DNSLinker) ApplyDesiredRecords(ipID int, desiredState model.DNSRecordDesiredState) error {
	// Get the IP
	var nodeIP model.NodeIP
	if err := l.db.First(&nodeIP, ipID).Error; err != nil {
		return fmt.Errorf("failed to find node IP: %w", err)
	}

	// Find all node groups that contain this IP
	var nodeGroupIPs []model.NodeGroupIP
	if err := l.db.Where("ip_id = ?", ipID).Find(&nodeGroupIPs).Error; err != nil {
		return fmt.Errorf("failed to find node group IPs: %w", err)
	}

	if len(nodeGroupIPs) == 0 {
		// IP not in any node group, no DNS records to manage
		return nil
	}

	// Get all node group IDs
	nodeGroupIDs := make([]int, len(nodeGroupIPs))
	for i, ngIP := range nodeGroupIPs {
		nodeGroupIDs[i] = ngIP.NodeGroupID
	}

	// Get all node groups
	var nodeGroups []model.NodeGroup
	if err := l.db.Where("id IN ?", nodeGroupIDs).Find(&nodeGroups).Error; err != nil {
		return fmt.Errorf("failed to find node groups: %w", err)
	}

	// Get all CDN domains
	var cdnDomains []model.Domain
	if err := l.db.Where("purpose = ? AND status = ?", "cdn", "active").Find(&cdnDomains).Error; err != nil {
		return fmt.Errorf("failed to find CDN domains: %w", err)
	}

	if len(cdnDomains) == 0 {
		// No CDN domains, no DNS records to manage
		return nil
	}

	// Upsert DNS records for each node group × each CDN domain
	for _, ng := range nodeGroups {
		for _, domain := range cdnDomains {
			record := model.DomainDNSRecord{
				DomainID:     domain.ID,
				Type:         model.DNSRecordTypeA,
				Name:         ng.CNAMEPrefix,
				Value:        nodeIP.IP,
				TTL:          120,
				OwnerType:    "node_group",
				OwnerID:      ng.ID,
				Status:       model.DNSRecordStatusPending,
				DesiredState: desiredState,
			}

			// Use ON DUPLICATE KEY UPDATE to ensure idempotency
			// The unique constraint is: (domain_id, type, name, value, owner_type, owner_id)
			if err := l.db.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "domain_id"},
					{Name: "type"},
					{Name: "name"},
					{Name: "value"},
					{Name: "owner_type"},
					{Name: "owner_id"},
				},
				DoUpdates: clause.AssignmentColumns([]string{
					"desired_state",
					"status",
					"updated_at",
				}),
			}).Create(&record).Error; err != nil {
				return fmt.Errorf("failed to upsert DNS record for domain %s: %w", domain.Domain, err)
			}
		}
	}

	return nil
}
