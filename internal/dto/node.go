package dto

import "time"

// NodeDTO represents a node in API responses
type NodeDTO struct {
	ID              int        `json:"id"`
	Name            string     `json:"name"`
	MainIp          string     `json:"mainIp"`
	AgentPort       int        `json:"agentPort"`
	Enabled         bool       `json:"enabled"`
	Status          string     `json:"status"`
	LastSeenAt      *time.Time `json:"lastSeenAt"`
	LastHealthError *string    `json:"lastHealthError"`
	HealthFailCount int        `json:"healthFailCount"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// NodeWithIdentityDTO represents a node with identity summary
type NodeWithIdentityDTO struct {
	ID              int          `json:"id"`
	Name            string       `json:"name"`
	MainIp          string       `json:"mainIp"`
	AgentPort       int          `json:"agentPort"`
	Enabled         bool         `json:"enabled"`
	Status          string       `json:"status"`
	LastSeenAt      *time.Time   `json:"lastSeenAt"`
	LastHealthError *string      `json:"lastHealthError"`
	HealthFailCount int          `json:"healthFailCount"`
	Identity        *IdentityDTO `json:"identity,omitempty"`
	CreatedAt       time.Time    `json:"createdAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
}

// IdentityDTO represents agent identity summary
type IdentityDTO struct {
	ID          int    `json:"id"`
	Fingerprint string `json:"fingerprint"`
}

// SubIpDTO represents a sub IP in API responses
type SubIpDTO struct {
	ID      int    `json:"id"`
	IP      string `json:"ip"`
	Enabled bool   `json:"enabled"`
}

// NodeDetailDTO represents detailed node information with sub IPs
type NodeDetailDTO struct {
	ID              int          `json:"id"`
	Name            string       `json:"name"`
	MainIp          string       `json:"mainIp"`
	AgentPort       int          `json:"agentPort"`
	Enabled         bool         `json:"enabled"`
	Status          string       `json:"status"`
	LastSeenAt      *time.Time   `json:"lastSeenAt"`
	LastHealthError *string      `json:"lastHealthError"`
	HealthFailCount int          `json:"healthFailCount"`
	Identity        *IdentityDTO `json:"identity,omitempty"`
	SubIps          []SubIpDTO   `json:"subIps"`
	CreatedAt       time.Time    `json:"createdAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
}

