package domains

import "time"

// DomainOptionDTO represents a domain option for dropdown selection
type DomainOptionDTO struct {
	ID        int64     `json:"id"`
	Domain    string    `json:"domain"`
	Status    string    `json:"status"`
	Purpose   string    `json:"purpose"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
