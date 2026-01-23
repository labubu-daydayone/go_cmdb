package risk

import (
	"fmt"
	"log"
	"go_cmdb/internal/model"
	"time"

	"gorm.io/gorm"
)

// ScannerConfig 风险扫描器配置
type ScannerConfig struct {
	Enabled              bool // 是否启用
	IntervalSec          int  // 扫描间隔（秒）
	CertExpiringDays     int  // 证书过期预警天数
	CertExpiringThreshold int // 证书过期影响网站数量阈值
	ACMEMaxAttempts      int  // ACME最大尝试次数
}

// Scanner 风险扫描器
type Scanner struct {
	db     *gorm.DB
	config ScannerConfig
	rules  []RiskRule
	stopCh chan struct{}
}

// NewScanner 创建风险扫描器
func NewScanner(db *gorm.DB, config ScannerConfig) *Scanner {
	// 初始化规则
	rules := []RiskRule{
		&DomainMismatchRule{},
		NewCertExpiringRule(config.CertExpiringDays, config.CertExpiringThreshold),
		NewACMERenewFailedRule(config.ACMEMaxAttempts),
		&WeakCoverageRule{},
	}

	return &Scanner{
		db:     db,
		config: config,
		rules:  rules,
		stopCh: make(chan struct{}),
	}
}

// Start 启动风险扫描器
func (s *Scanner) Start() {
	if !s.config.Enabled {
		log.Println("[RiskScanner] Disabled, not starting")
		return
	}

	log.Printf("[RiskScanner] Starting with interval %d seconds", s.config.IntervalSec)

	// 立即执行一次扫描
	go func() {
		log.Println("[RiskScanner] Running initial scan...")
		if err := s.Scan(); err != nil {
			log.Printf("[RiskScanner] Initial scan failed: %v", err)
		}
	}()

	// 启动定时扫描
	ticker := time.NewTicker(time.Duration(s.config.IntervalSec) * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Println("[RiskScanner] Running scheduled scan...")
				if err := s.Scan(); err != nil {
					log.Printf("[RiskScanner] Scan failed: %v", err)
				}
			case <-s.stopCh:
				ticker.Stop()
				log.Println("[RiskScanner] Stopped")
				return
			}
		}
	}()
}

// Stop 停止风险扫描器
func (s *Scanner) Stop() {
	close(s.stopCh)
}

// Scan 执行一次风险扫描
func (s *Scanner) Scan() error {
	log.Println("[RiskScanner] Starting scan...")

	totalRisks := 0
	newRisks := 0
	updatedRisks := 0

	// 对每个规则执行检测
	for _, rule := range s.rules {
		risks, err := rule.Detect(s.db)
		if err != nil {
			log.Printf("[RiskScanner] Rule detection failed: %v", err)
			continue
		}

		totalRisks += len(risks)

		// 幂等生成风险
		for _, risk := range risks {
			inserted, err := s.upsertRisk(risk)
			if err != nil {
				log.Printf("[RiskScanner] Failed to upsert risk: %v", err)
				continue
			}

			if inserted {
				newRisks++
			} else {
				updatedRisks++
			}
		}
	}

	log.Printf("[RiskScanner] Scan completed: %d risks detected, %d new, %d updated",
		totalRisks, newRisks, updatedRisks)

	return nil
}

// upsertRisk 幂等生成风险
// 使用UNIQUE KEY约束：(risk_type, certificate_id, website_id, status)
// 返回：(是否为新插入, error)
func (s *Scanner) upsertRisk(risk model.CertificateRisk) (bool, error) {
	// 检查是否已存在active状态的相同风险
	var existing model.CertificateRisk
	query := s.db.Where("risk_type = ? AND status = ?", risk.RiskType, model.RiskStatusActive)

	if risk.CertificateID != nil {
		query = query.Where("certificate_id = ?", *risk.CertificateID)
	} else {
		query = query.Where("certificate_id IS NULL")
	}

	if risk.WebsiteID != nil {
		query = query.Where("website_id = ?", *risk.WebsiteID)
	} else {
		query = query.Where("website_id IS NULL")
	}

	err := query.First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 不存在，插入新记录
		if err := s.db.Create(&risk).Error; err != nil {
			return false, fmt.Errorf("failed to create risk: %w", err)
		}
		log.Printf("[RiskScanner] New risk created: type=%s, cert_id=%v, website_id=%v",
			risk.RiskType, risk.CertificateID, risk.WebsiteID)
		return true, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to query existing risk: %w", err)
	}

	// 已存在，更新detected_at和detail
	updates := map[string]interface{}{
		"detected_at": risk.DetectedAt,
		"detail":      risk.Detail,
		"level":       risk.Level, // 更新level（可能会变化）
	}

	if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
		return false, fmt.Errorf("failed to update risk: %w", err)
	}

	return false, nil
}

// ResolveRisk 解决风险（人工标记）
func (s *Scanner) ResolveRisk(riskID int) error {
	// 查询风险
	var risk model.CertificateRisk
	if err := s.db.First(&risk, riskID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("risk not found")
		}
		return fmt.Errorf("failed to query risk: %w", err)
	}

	// 检查状态
	if risk.Status == model.RiskStatusResolved {
		return fmt.Errorf("risk already resolved")
	}

	// 更新状态
	now := time.Now()
	updates := map[string]interface{}{
		"status":      model.RiskStatusResolved,
		"resolved_at": now,
	}

	if err := s.db.Model(&risk).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to resolve risk: %w", err)
	}

	log.Printf("[RiskScanner] Risk resolved: id=%d, type=%s", riskID, risk.RiskType)
	return nil
}
