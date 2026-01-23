package risk

import (
	"fmt"
	"go_cmdb/internal/model"

	"gorm.io/gorm"
)

// Service 风险服务
type Service struct {
	db *gorm.DB
}

// NewService 创建风险服务
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// ListRisksFilter 风险列表过滤条件
type ListRisksFilter struct {
	Level         *model.RiskLevel  `form:"level"`
	RiskType      *model.RiskType   `form:"risk_type"`
	Status        *model.RiskStatus `form:"status"`
	CertificateID *int              `form:"certificate_id"`
	WebsiteID     *int              `form:"website_id"`
	Page          int               `form:"page"`
	PageSize      int               `form:"page_size"`
}

// ListRisksResult 风险列表结果
type ListRisksResult struct {
	Risks      []model.CertificateRisk `json:"risks"`
	Total      int64                   `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"page_size"`
	TotalPages int                     `json:"total_pages"`
}

// ListRisks 查询风险列表（全局）
func (s *Service) ListRisks(filter ListRisksFilter) (*ListRisksResult, error) {
	// 默认值
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	// 构建查询
	query := s.db.Model(&model.CertificateRisk{})

	// 应用过滤条件
	if filter.Level != nil {
		query = query.Where("level = ?", *filter.Level)
	}
	if filter.RiskType != nil {
		query = query.Where("risk_type = ?", *filter.RiskType)
	}
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	if filter.CertificateID != nil {
		query = query.Where("certificate_id = ?", *filter.CertificateID)
	}
	if filter.WebsiteID != nil {
		query = query.Where("website_id = ?", *filter.WebsiteID)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count risks: %w", err)
	}

	// 查询数据
	var risks []model.CertificateRisk
	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Order("detected_at DESC").
		Limit(filter.PageSize).
		Offset(offset).
		Find(&risks).Error; err != nil {
		return nil, fmt.Errorf("failed to query risks: %w", err)
	}

	// 计算总页数
	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	return &ListRisksResult{
		Risks:      risks,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// ListWebsiteRisks 查询网站的风险列表
func (s *Service) ListWebsiteRisks(websiteID int) ([]model.CertificateRisk, error) {
	var risks []model.CertificateRisk
	if err := s.db.Where("website_id = ? AND status = ?", websiteID, model.RiskStatusActive).
		Order("level DESC, detected_at DESC").
		Find(&risks).Error; err != nil {
		return nil, fmt.Errorf("failed to query website risks: %w", err)
	}
	return risks, nil
}

// ListCertificateRisks 查询证书的风险列表
func (s *Service) ListCertificateRisks(certificateID int) ([]model.CertificateRisk, error) {
	var risks []model.CertificateRisk
	if err := s.db.Where("certificate_id = ? AND status = ?", certificateID, model.RiskStatusActive).
		Order("level DESC, detected_at DESC").
		Find(&risks).Error; err != nil {
		return nil, fmt.Errorf("failed to query certificate risks: %w", err)
	}
	return risks, nil
}

// ResolveRisk 解决风险（人工标记）
func (s *Service) ResolveRisk(riskID int) error {
	scanner := NewScanner(s.db, ScannerConfig{})
	return scanner.ResolveRisk(riskID)
}
