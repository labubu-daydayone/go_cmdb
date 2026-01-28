package line_groups

// LineGroupItemDTO represents a line group item in list or detail response
type LineGroupItemDTO struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	DomainID      int    `json:"domainId"`
	DomainName    string `json:"domainName"`    // For display
	NodeGroupID   int    `json:"nodeGroupId"`
	NodeGroupName string `json:"nodeGroupName"` // For display
	CNAMEPrefix   string `json:"cnamePrefix"`
	CNAME         string `json:"cname"`         // Computed field: cnamePrefix + "." + domainName
	Status        string `json:"status"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

// LineGroupListResponse represents list line groups response
type LineGroupListResponse struct {
	Items    []LineGroupItemDTO `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

// LineGroupDetailResponse represents line group detail response
type LineGroupDetailResponse struct {
	Item LineGroupItemDTO `json:"item"`
}
