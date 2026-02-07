package risk

import (
	"fmt"
	"go_cmdb/internal/cert"
	"go_cmdb/internal/domainutil"
	"go_cmdb/internal/model"
	"time"

	"gorm.io/gorm"
)

// RiskRule 风险规则接口
type RiskRule interface {
	// Detect 检测风险并返回风险列表
	Detect(db *gorm.DB) ([]model.CertificateRisk, error)
}

// DomainMismatchRule 域名不匹配风险规则
// 触发条件：
// - website_domains新增/删除
// - website_https.enabled = 1
// - cert_mode = select
// - certificate_domains不再100%覆盖
type DomainMismatchRule struct{}

func (r *DomainMismatchRule) Detect(db *gorm.DB) ([]model.CertificateRisk, error) {
	risks := []model.CertificateRisk{}

	// 查询所有启用HTTPS且使用select模式的网站
	var websites []struct {
		WebsiteID     int
		CertificateID int
	}

	err := db.Raw(`
		SELECT 
			w.id AS website_id,
			wh.certificate_id
		FROM websites w
		JOIN website_https wh ON w.id = wh.website_id
		WHERE wh.enabled = 1
		  AND wh.cert_mode = 'select'
		  AND wh.certificate_id IS NOT NULL
	`).Scan(&websites).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query websites: %w", err)
	}

	// 对每个网站检查证书覆盖
	for _, ws := range websites {
		// 获取证书域名
		var certDomains []string
		err := db.Raw(`
			SELECT domain FROM certificate_domains WHERE certificate_id = ?
		`, ws.CertificateID).Scan(&certDomains).Error

		if err != nil {
			continue
		}

		// 获取网站域名
		var websiteDomains []string
		err = db.Raw(`
			SELECT domain FROM website_domains WHERE website_id = ?
		`, ws.WebsiteID).Scan(&websiteDomains).Error

		if err != nil {
			continue
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

			risk := model.CertificateRisk{
				RiskType:      model.RiskTypeDomainMismatch,
				Level:         model.RiskLevelCritical,
				CertificateID: &ws.CertificateID,
				WebsiteID:     &ws.WebsiteID,
				Detail:        detail,
				Status:        model.RiskStatusActive,
				DetectedAt:    time.Now(),
			}

			risks = append(risks, risk)
		}
	}

	return risks, nil
}

// CertExpiringRule 证书即将过期风险规则
// 触发条件：
// - certificates.expire_at < now + N天（默认15）
// - certificate_bindings.active = 1
// - 绑定网站数量 >= 2（阈值可配置）
type CertExpiringRule struct {
	ExpiringDays         int // 提前多少天预警
	WebsiteThreshold     int // 影响网站数量阈值
}

func NewCertExpiringRule(expiringDays, websiteThreshold int) *CertExpiringRule {
	if expiringDays <= 0 {
		expiringDays = 15 // 默认15天
	}
	if websiteThreshold <= 0 {
		websiteThreshold = 2 // 默认2个网站
	}
	return &CertExpiringRule{
		ExpiringDays:     expiringDays,
		WebsiteThreshold: websiteThreshold,
	}
}

func (r *CertExpiringRule) Detect(db *gorm.DB) ([]model.CertificateRisk, error) {
	risks := []model.CertificateRisk{}

	// 查询即将过期的证书及其影响的网站
	var results []struct {
		CertificateID    int
		ExpireAt         time.Time
		AffectedWebsites string // 逗号分隔的website_id列表
		WebsiteCount     int
	}

	expiringDate := time.Now().AddDate(0, 0, r.ExpiringDays)

	err := db.Raw(`
		SELECT 
			c.id AS certificate_id,
			c.expire_at,
			GROUP_CONCAT(cb.bind_id) AS affected_websites,
			COUNT(DISTINCT cb.bind_id) AS website_count
		FROM certificates c
		JOIN certificate_bindings cb ON c.id = cb.certificate_id
		WHERE c.expire_at < ?
		  AND c.expire_at > NOW()
		  AND cb.bind_type = 'website'
		  AND cb.is_active = 1
		GROUP BY c.id, c.expire_at
		HAVING website_count >= ?
	`, expiringDate, r.WebsiteThreshold).Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query expiring certificates: %w", err)
	}

	// 生成风险
	for _, result := range results {
		daysRemaining := int(time.Until(result.ExpireAt).Hours() / 24)

		detail := model.RiskDetail{
			"message": fmt.Sprintf("Certificate expires in %d days and affects %d websites",
				daysRemaining, result.WebsiteCount),
			"certificate_id":     result.CertificateID,
			"expire_at":          result.ExpireAt.Format(time.RFC3339),
			"days_remaining":     daysRemaining,
			"affected_websites":  result.AffectedWebsites,
			"website_count":      result.WebsiteCount,
		}

		risk := model.CertificateRisk{
			RiskType:      model.RiskTypeCertExpiring,
			Level:         model.RiskLevelWarning,
			CertificateID: &result.CertificateID,
			WebsiteID:     nil, // 影响多个网站，不关联单个website_id
			Detail:        detail,
			Status:        model.RiskStatusActive,
			DetectedAt:    time.Now(),
		}

		risks = append(risks, risk)
	}

	return risks, nil
}

// ACMERenewFailedRule ACME续期失败风险规则
// 触发条件：
// - certificate_requests.status = failed
// - attempts >= max_attempts
// - 对应certificate当前仍被active绑定
type ACMERenewFailedRule struct {
	MaxAttempts int // 最大尝试次数
}

func NewACMERenewFailedRule(maxAttempts int) *ACMERenewFailedRule {
	if maxAttempts <= 0 {
		maxAttempts = 3 // 默认3次
	}
	return &ACMERenewFailedRule{
		MaxAttempts: maxAttempts,
	}
}

func (r *ACMERenewFailedRule) Detect(db *gorm.DB) ([]model.CertificateRisk, error) {
	risks := []model.CertificateRisk{}

	// 查询ACME续期失败的证书请求
	var results []struct {
		CertificateID int
		RequestID     int
		Attempts      int
		LastError     string
	}

	err := db.Raw(`
		SELECT 
			cr.result_certificate_id AS certificate_id,
			cr.id AS request_id,
			cr.attempts,
			cr.last_error
		FROM certificate_requests cr
		WHERE cr.status = 'failed'
		  AND cr.attempts >= ?
		  AND cr.result_certificate_id IS NOT NULL
		  AND EXISTS (
			  SELECT 1 FROM certificate_bindings cb
			  WHERE cb.certificate_id = cr.result_certificate_id
			    AND cb.bind_type = 'website'
			    AND cb.is_active = 1
		  )
	`, r.MaxAttempts).Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query failed ACME renewals: %w", err)
	}

	// 生成风险
	for _, result := range results {
		detail := model.RiskDetail{
			"message": fmt.Sprintf("ACME renewal failed after %d attempts", result.Attempts),
			"certificate_id": result.CertificateID,
			"request_id":     result.RequestID,
			"attempts":       result.Attempts,
			"last_error":     result.LastError,
		}

		risk := model.CertificateRisk{
			RiskType:      model.RiskTypeACMERenewFailed,
			Level:         model.RiskLevelCritical,
			CertificateID: &result.CertificateID,
			WebsiteID:     nil,
			Detail:        detail,
			Status:        model.RiskStatusActive,
			DetectedAt:    time.Now(),
		}

		risks = append(risks, risk)
	}

	return risks, nil
}

// WeakCoverageRule 弱覆盖风险规则
// 定义：
// - wildcard覆盖成立
// - 但website_domains数量 > 1
// - 且包含apex + subdomain混合
//
// 例如：
// - 网站域名：example.com, www.example.com
// - 证书：*.example.com
// 结果：covered（T2-07已允许），但产生warning级风险
type WeakCoverageRule struct{}

func (r *WeakCoverageRule) Detect(db *gorm.DB) ([]model.CertificateRisk, error) {
	risks := []model.CertificateRisk{}

	// 查询所有启用HTTPS且使用select模式的网站
	var websites []struct {
		WebsiteID     int
		CertificateID int
	}

	err := db.Raw(`
		SELECT 
			w.id AS website_id,
			wh.certificate_id
		FROM websites w
		JOIN website_https wh ON w.id = wh.website_id
		WHERE wh.enabled = 1
		  AND wh.cert_mode = 'select'
		  AND wh.certificate_id IS NOT NULL
	`).Scan(&websites).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query websites: %w", err)
	}

	// 对每个网站检查弱覆盖
	for _, ws := range websites {
		// 获取证书域名
		var certDomains []string
		err := db.Raw(`
			SELECT domain FROM certificate_domains WHERE certificate_id = ?
		`, ws.CertificateID).Scan(&certDomains).Error

		if err != nil {
			continue
		}

		// 获取网站域名
		var websiteDomains []string
		err = db.Raw(`
			SELECT domain FROM website_domains WHERE website_id = ?
		`, ws.WebsiteID).Scan(&websiteDomains).Error

		if err != nil {
			continue
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

			risk := model.CertificateRisk{
				RiskType:      model.RiskTypeWeakCoverage,
				Level:         model.RiskLevelWarning,
				CertificateID: &ws.CertificateID,
				WebsiteID:     &ws.WebsiteID,
				Detail:        detail,
				Status:        model.RiskStatusActive,
				DetectedAt:    time.Now(),
			}

			risks = append(risks, risk)
		}
	}

	return risks, nil
}

// isWeakCoverage 检查是否为弱覆盖
// 条件：
// 1. wildcard覆盖成立（完全覆盖）
// 2. website_domains数量 > 1
// 3. 包含apex + subdomain混合
func isWeakCoverage(certDomains, websiteDomains []string) bool {
	// 条件1：必须完全覆盖
	coverage := cert.CalculateCoverage(certDomains, websiteDomains)
	if coverage.Status != cert.CoverageStatusCovered {
		return false
	}

	// 条件2：网站域名数量 > 1
	if len(websiteDomains) <= 1 {
		return false
	}

	// 条件3：检查是否包含apex + subdomain混合
	hasApex := false
	hasSubdomain := false

	for _, domain := range websiteDomains {
		if isApexDomain(domain) {
			hasApex = true
		} else {
			hasSubdomain = true
		}
	}

	// 必须同时包含apex和subdomain
	if !hasApex || !hasSubdomain {
		return false
	}

	// 条件4：证书必须是纯wildcard（不包含apex）
	hasWildcard := false
	hasApexInCert := false

	for _, domain := range certDomains {
		if isWildcard(domain) {
			hasWildcard = true
		}
		baseDomain := extractBaseDomain(websiteDomains)
		if domain == baseDomain {
			hasApexInCert = true
		}
	}

	// 必须有wildcard且没有apex
	return hasWildcard && !hasApexInCert
}

// isWildcard 判断是否为wildcard域名
func isWildcard(domain string) bool {
	return len(domain) > 2 && domain[0] == '*' && domain[1] == '.'
}

// isApexDomain 判断是否为apex域名（不包含子域名）
// 使用 PSL 计算 eTLD+1，如果域名等于其 eTLD+1 则为 apex
func isApexDomain(domain string) bool {
	apex, err := domainutil.EffectiveApex(domain)
	if err != nil {
		return false
	}
	return domain == apex
}

// extractBaseDomain 提取基础域名（eTLD+1）
// 使用 PSL 计算，例如：
//   - www.example.com -> example.com
//   - a.b.example.co.uk -> example.co.uk
func extractBaseDomain(domains []string) string {
	for _, domain := range domains {
		if isApexDomain(domain) {
			return domain
		}
	}

	// 如果没有 apex 域名，从第一个域名提取 eTLD+1
	if len(domains) > 0 {
		apex, err := domainutil.EffectiveApex(domains[0])
		if err != nil {
			return domains[0]
		}
		return apex
	}

	return ""
}
