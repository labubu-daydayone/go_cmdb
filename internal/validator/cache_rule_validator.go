package validator

import (
	"fmt"
	"regexp"
	"strings"

	"go_cmdb/internal/model"
)

// CacheRuleItemValidator 缓存规则项校验器
type CacheRuleItemValidator struct{}

// NewCacheRuleItemValidator 创建校验器实例
func NewCacheRuleItemValidator() *CacheRuleItemValidator {
	return &CacheRuleItemValidator{}
}

// Validate 校验缓存规则项
func (v *CacheRuleItemValidator) Validate(matchType, matchValue string, ttlSeconds int) error {
	// 去除首尾空格
	matchValue = strings.TrimSpace(matchValue)

	// 通用校验：matchValue 不允许为空
	if matchValue == "" {
		return fmt.Errorf("matchValue cannot be empty")
	}

	// 通用校验：ttlSeconds 必须为正整数
	if ttlSeconds <= 0 {
		return fmt.Errorf("ttlSeconds must be greater than 0")
	}

	// 根据 matchType 进行具体校验
	switch matchType {
	case model.MatchTypePath:
		return v.validatePath(matchValue)
	case model.MatchTypeSuffix:
		return v.validateSuffix(matchValue)
	case model.MatchTypeExact:
		return v.validateExact(matchValue)
	default:
		return fmt.Errorf("invalid matchType: %s", matchType)
	}
}

// validatePath 校验 path 类型规则
// 规则：
// - 必须以 / 开头
// - 必须以 / 结尾
// - 不允许空字符串
func (v *CacheRuleItemValidator) validatePath(matchValue string) error {
	if !strings.HasPrefix(matchValue, "/") {
		return fmt.Errorf("path matchValue must start with /")
	}

	if !strings.HasSuffix(matchValue, "/") {
		return fmt.Errorf("path matchValue must end with /")
	}

	if matchValue == "" {
		return fmt.Errorf("path matchValue cannot be empty")
	}

	return nil
}

// validateSuffix 校验 suffix 类型规则
// 规则：
// - 必须以 . 开头
// - 不得包含 /
// - 不得以 / 结尾
// - 仅允许 [a-zA-Z0-9._-]
func (v *CacheRuleItemValidator) validateSuffix(matchValue string) error {
	if !strings.HasPrefix(matchValue, ".") {
		return fmt.Errorf("suffix matchValue must start with .")
	}

	if strings.Contains(matchValue, "/") {
		return fmt.Errorf("suffix matchValue cannot contain /")
	}

	if strings.HasSuffix(matchValue, "/") {
		return fmt.Errorf("suffix matchValue cannot end with /")
	}

	// 仅允许 [a-zA-Z0-9._-]
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._-]+$`, matchValue)
	if !matched {
		return fmt.Errorf("suffix matchValue can only contain [a-zA-Z0-9._-]")
	}

	return nil
}

// validateExact 校验 exact 类型规则
// 规则：
// - 必须以 / 开头
// - 后缀是否以 / 结尾不做限制
func (v *CacheRuleItemValidator) validateExact(matchValue string) error {
	if !strings.HasPrefix(matchValue, "/") {
		return fmt.Errorf("exact matchValue must start with /")
	}

	return nil
}
