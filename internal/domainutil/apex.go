package domainutil

import (
	"fmt"
	"net"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/publicsuffix"
	"gorm.io/gorm"
)

// Normalize 对域名进行规范化处理
// 规则：
//   - 小写
//   - trim 空格
//   - 去掉末尾 .
//   - 去掉端口（如 example.com:443）
//   - 拒绝 IP（IPv4/IPv6）
//   - 拒绝空字符串/非法字符
func Normalize(host string) (string, error) {
	// trim 空格
	host = strings.TrimSpace(host)

	// 拒绝空字符串
	if host == "" {
		return "", fmt.Errorf("domain must not be empty")
	}

	// 小写
	host = strings.ToLower(host)

	// 去掉末尾 .
	host = strings.TrimSuffix(host, ".")

	// 去掉端口
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	// 再次检查空字符串（去掉端口后可能为空）
	if host == "" {
		return "", fmt.Errorf("domain must not be empty after normalization")
	}

	// 拒绝 IPv4
	if net.ParseIP(host) != nil {
		return "", fmt.Errorf("IP address is not allowed as domain: %s", host)
	}

	// 拒绝 IPv6（带方括号的形式）
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		inner := host[1 : len(host)-1]
		if net.ParseIP(inner) != nil {
			return "", fmt.Errorf("IP address is not allowed as domain: %s", host)
		}
	}

	// 校验域名合法性：只允许 a-z 0-9 . -
	for i := 0; i < len(host); {
		r, size := utf8.DecodeRuneInString(host[i:])
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '*') {
			return "", fmt.Errorf("domain contains invalid character: %c in %s", r, host)
		}
		i += size
	}

	// 域名不能以 . 或 - 开头
	if strings.HasPrefix(host, ".") || strings.HasPrefix(host, "-") {
		return "", fmt.Errorf("domain must not start with '.' or '-': %s", host)
	}

	// 域名必须包含至少一个 .
	if !strings.Contains(host, ".") {
		return "", fmt.Errorf("domain must contain at least one dot: %s", host)
	}

	return host, nil
}

// EffectiveApex 使用 PSL 计算 eTLD+1（注册域名/授权根）
// 例如：
//   - www.example.com -> example.com
//   - a.b.example.co.uk -> example.co.uk
//   - example.com -> example.com
//
// 项目内任何地方不得自行 split 计算 apex，必须调用此函数
func EffectiveApex(domain string) (string, error) {
	// 先规范化
	normalized, err := Normalize(domain)
	if err != nil {
		return "", fmt.Errorf("normalize failed for %s: %w", domain, err)
	}

	// 处理通配符域名：去掉 *. 前缀后计算
	if strings.HasPrefix(normalized, "*.") {
		normalized = normalized[2:]
	}

	// 使用 PSL 计算 eTLD+1
	apex, err := publicsuffix.EffectiveTLDPlusOne(normalized)
	if err != nil {
		return "", fmt.Errorf("PSL lookup failed for %s: %w", domain, err)
	}

	return apex, nil
}

// ValidateWebsiteDomains 校验网站域名列表
// 对每个域名：
//  1. Normalize
//  2. 用 PSL 算 apex
//  3. 在 domains 表中查找 apex 且 status='active'
//
// 任一域名不通过则返回错误，不进行后续流程
func ValidateWebsiteDomains(db *gorm.DB, domains []string) error {
	if len(domains) == 0 {
		return fmt.Errorf("domains list must not be empty")
	}

	for _, domain := range domains {
		// 规范化
		normalized, err := Normalize(domain)
		if err != nil {
			return fmt.Errorf("domain not allowed: %s, %s", domain, err.Error())
		}

		// 计算 apex
		apex, err := EffectiveApex(normalized)
		if err != nil {
			return fmt.Errorf("domain not allowed: %s, %s", domain, err.Error())
		}

		// 在 domains 表中查找 apex 且 status='active'
		var count int64
		if err := db.Table("domains").
			Where("domain = ? AND status = ?", apex, "active").
			Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check apex for %s: %w", domain, err)
		}

		if count == 0 {
			return fmt.Errorf("domain not allowed: %s, apex=%s not active in domains", domain, apex)
		}
	}

	return nil
}
