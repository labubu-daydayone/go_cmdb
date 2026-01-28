package dto

import "time"

// NodeIPItem represents a node IP item in list response
type NodeIPItem struct {
	ID        int       `json:"id"`
	NodeID    int       `json:"nodeId"`
	IP        string    `json:"ip"`
	IPRole    string    `json:"ipRole"` // main or sub
	Enabled   bool      `json:"enabled"`
	Status    string    `json:"status"` // active, unreachable, disabled
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
