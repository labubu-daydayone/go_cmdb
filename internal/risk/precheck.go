package risk

import (
	"fmt"
	"go_cmdb/internal/cert"
	"go_cmdb/internal/model"
	"time"
)

// PrecheckRequest 预检请求
type PrecheckRequest struct {
	CertMode      string `json:"cert_mode" binding:"required"`
	CertificateID *int   `json:"certificate_id"`
}

// PrecheckRisk 预检风险
type PrecheckRisk struct {
	Type   model.RiskType  `json:"type"`
	Level  model.RiskLevel `json:"level"`
	Detail model.RiskDetail `json:"detail"`
}

// PrecheckResult 预检结果
type PrecheckResult struct {
	OK    bool           `json:"ok"`
	Risks []PrecheckRisk `json:"risks"`
}

// PrecheckHTTPS 前置风险预检
// 在用户启用HTTPS之前，检查可能的风险
func (s *Service) PrecheckHTTPS(websiteID int, req PrecheckRequest) (*PrecheckResult, error) {
	risks := []PrecheckRisk{}

	// 如果是select模式，需要检查证书覆盖
	if req.CertMode == "select" {
		if req.CertificateID == nil {
			return nil, fmt.Errorf("certificate_id is required for select mode")
		}

		// 检查domain_mismatch风险
		domainMismatchRisk, err := s.checkDomainMismatch(websiteID, *req.CertificateID)
		if err != nil {
			return nil, err
		}
		if domainMismatchRisk != nil {
			risks = append(risks, *domainMismatchRisk)
		}

		// 检查weak_coverage风险
		weakCoverageRisk, err := s.checkWeakCoverage(websiteID, *req.CertificateID)
		if err != nil {
			return nil, err
		}
		if weakCoverageRisk != nil {
			risks = append(risks, *weakCoverageRisk)
		}

		// 检查cert_expiring风险
		certExpiringRisk, err := s.checkCertExpiring(*req.CertificateID)
		if err != nil {
			return nil, err
		}
		if certExpiringRisk != nil {
			risks = append(risks, *certExpiringRisk)
		}
	}

	// 判断是否可以启用HTTPS
	// 有critical级别的风险时，ok=false
	ok := true
	for _, r := range risks {
		if r.Level == model.RiskLevelCritical {
			ok = false
			break
		}
	}

	return &PrecheckResult{
		OK:    ok,
		Risks: risks,
	}, nil
}

// checkDomainMismatch 检查域名不匹配风险
func (s *Service) checkDomainMismatch(websiteID, certificateID int) (*PrecheckRisk, error) {
	// 获取证书域名
	var certDomains []string
	if err := s.db.Raw(`
		SELECT domain FROM certificate_domains WHERE certificate_id = ?
	`, certificateID).Scan(&certDomains).Error; err != nil {
		return nil, fmt.Errorf("failed to query certificate domains: %w", err)
	}

	// 获取网站域名
	var websiteDomains []string
	if err := s.db.Raw(`
		SELECT domain FROM website_domains WHERE website_id = ?
	`, websiteID).Scan(&websiteDomains).Error; err != nil {
		return nil, fmt.Errorf("failed to query website domains: %w", err)
	}

	// 计算覆盖状态
	coverage := cert.CalculateCoverage(certDomains, websiteDomains)

	// 如果不是完全覆盖，生成风险
	if coverage.Status != cert.CoverageStatusCovered {
		detail := model.RiskDetail{
			"message":             "Certificate does not cover all website domains",
			"certificate_domains": certDomains,
			"website_domains":     websiteDomains,
			"missing_domains":     coverage.MissingDomains,
			"coverage_status":     coverage.Status,
		}

		return &PrecheckRisk{
			Type:   model.RiskTypeDomainMismatch,
			Level:  model.RiskLevelCritical,
			Detail: detail,
		}, nil
	}

	return nil, nil
}

// checkWeakCoverage 检查弱覆盖风险
func (s *Service) checkWeakCoverage(websiteID, certificateID int) (*PrecheckRisk, error) {
	// 获取证书域名
	var certDomains []string
	if err := s.db.Raw(`
		SELECT domain FROM certificate_domains WHERE certificate_id = ?
	`, certificateID).Scan(&certDomains).Error; err != nil {
		return nil, fmt.Errorf("failed to query certificate domains: %w", err)
	}

	// 获取网站域名
	var websiteDomains []string
	if err := s.db.Raw(`
		SELECT domain FROM website_domains WHERE website_id = ?
	`, websiteID).Scan(&websiteDomains).Error; err != nil {
		return nil, fmt.Errorf("failed to query website domains: %w", err)
	}

	// 检查是否为弱覆盖
	if isWeakCoverage(certDomains, websiteDomains) {
		// 找出基础域名（用于推荐）
		baseDomain := extractBaseDomain(websiteDomains)

		detail := model.RiskDetail{
			"message": "Wildcard certificate covers mixed apex and subdomain",
			"certificate_domains": certDomains,
			"website_domains":     websiteDomains,
			"recommendation": fmt.Sprintf("Use certificate with both %s and *.%s",
				baseDomain, baseDomain),
		}

		return &PrecheckRisk{
			Type:   model.RiskTypeWeakCoverage,
			Level:  model.RiskLevelWarning,
			Detail: detail,
		}, nil
	}

	return nil, nil
}

// checkCertExpiring 检查证书即将过期风险
func (s *Service) checkCertExpiring(certificateID int) (*PrecheckRisk, error) {
	// 查询证书过期时间
	var expireAt time.Time
	if err := s.db.Raw(`
		SELECT expire_at FROM certificates WHERE id = ?
	`, certificateID).Scan(&expireAt).Error; err != nil {
		return nil, fmt.Errorf("failed to query certificate expire_at: %w", err)
	}

	// 检查是否即将过期（15天内）
	expiringDate := time.Now().AddDate(0, 0, 15)
	if expireAt.Before(expiringDate) && expireAt.After(time.Now()) {
		daysRemaining := int(time.Until(expireAt).Hours() / 24)

		detail := model.RiskDetail{
			"message": fmt.Sprintf("Certificate expires in %d days", daysRemaining),
			"certificate_id": certificateID,
			"expire_at":      expireAt.Format(time.RFC3339),
			"days_remaining": daysRemaining,
		}

		return &PrecheckRisk{
			Type:   model.RiskTypeCertExpiring,
			Level:  model.RiskLevelWarning,
			Detail: detail,
		}, nil
	}

	return nil, nil
}
