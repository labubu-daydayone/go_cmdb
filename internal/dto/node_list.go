package dto

import "time"

// NodeListItemDTO represents a node in list API responses with ips.items structure
type NodeListItemDTO struct {
	ID              int                `json:"id"`
	Name            string             `json:"name"`
	MainIp          string             `json:"mainIp"`
	AgentPort       int                `json:"agentPort"`
	Enabled         bool               `json:"enabled"`
	AgentStatus     string             `json:"agentStatus"`
	LastSeenAt      *time.Time         `json:"lastSeenAt,omitempty"`
	LastHealthError *string            `json:"lastHealthError,omitempty"`
	HealthFailCount int                `json:"healthFailCount"`
	Ips             NodeIPsContainerDTO `json:"ips"`
	CreatedAt       time.Time          `json:"createdAt"`
	UpdatedAt       time.Time          `json:"updatedAt"`
}

// NodeIPsContainerDTO represents the ips container with items
type NodeIPsContainerDTO struct {
	Items []NodeIPItemDTO `json:"items"`
}

// NodeIPItemDTO represents an IP in the ips.items array
type NodeIPItemDTO struct {
	ID               int    `json:"id"`
	IP               string `json:"ip"`
	IpType           string `json:"ipType"`
	IpEnabled        bool   `json:"ipEnabled"`
	EffectiveEnabled bool   `json:"effectiveEnabled"`
}
