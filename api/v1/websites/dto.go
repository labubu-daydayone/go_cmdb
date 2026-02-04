package websites

import "time"

// WebsiteListItemDTO 网站列表项 DTO
type WebsiteListItemDTO struct {
	ID                 int       `json:"id"`
	LineGroupID        int       `json:"lineGroupId"`
	LineGroupName      string    `json:"lineGroupName,omitempty"`
	CacheRuleID        int       `json:"cacheRuleId"`
	OriginMode         string    `json:"originMode"`
	OriginGroupID      *int      `json:"originGroupId"`
	OriginGroupName    string    `json:"originGroupName,omitempty"`
	OriginSetID        *int      `json:"originSetId"`
	RedirectURL        string    `json:"redirectUrl,omitempty"`
	RedirectStatusCode int       `json:"redirectStatusCode,omitempty"`
	Status             string    `json:"status"`
	Domains            []string  `json:"domains,omitempty"`
	CNAME              string    `json:"cname,omitempty"`
	HTTPSEnabled       bool      `json:"httpsEnabled"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// WebsiteDTO 网站详情 DTO
type WebsiteDTO struct {
	ID                 int       `json:"id"`
	LineGroupID        int       `json:"lineGroupId"`
	LineGroupName      string    `json:"lineGroupName,omitempty"`
	CacheRuleID        int       `json:"cacheRuleId"`
	OriginMode         string    `json:"originMode"`
	OriginGroupID      *int      `json:"originGroupId"`
	OriginGroupName    string    `json:"originGroupName,omitempty"`
	OriginSetID        *int      `json:"originSetId"`
	RedirectURL        string    `json:"redirectUrl,omitempty"`
	RedirectStatusCode int       `json:"redirectStatusCode,omitempty"`
	Status             string    `json:"status"`
	Domains            []string  `json:"domains,omitempty"`
	CNAME              string    `json:"cname,omitempty"`
	HTTPSEnabled       bool      `json:"httpsEnabled"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}
