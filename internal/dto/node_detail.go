package dto

import "time"

// NodeDetailItemDTO represents a node detail in API responses with ips.items structure
type NodeDetailItemDTO struct {
	ID              int                `json:"id"`
	Name            string             `json:"name"`
	MainIp          string             `json:"mainIp"`
	AgentPort       int                `json:"agentPort"`
	NodeEnabled     bool               `json:"nodeEnabled"`
	AgentStatus     string             `json:"agentStatus"`
	LastSeenAt      *time.Time         `json:"lastSeenAt,omitempty"`
	LastHealthError *string            `json:"lastHealthError,omitempty"`
	HealthFailCount int                `json:"healthFailCount"`
	Identity        *IdentityDTO       `json:"identity,omitempty"`
	Ips             NodeIPsContainerDTO `json:"ips"`
	CreatedAt       time.Time          `json:"createdAt"`
	UpdatedAt       time.Time          `json:"updatedAt"`
}
