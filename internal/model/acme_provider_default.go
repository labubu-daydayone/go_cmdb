package model

import "time"

// ACMEProviderDefault represents the default ACME account for a provider
type ACMEProviderDefault struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ProviderID int64     `gorm:"column:provider_id;not null;uniqueIndex" json:"providerId"`
	AccountID  int64     `gorm:"column:account_id;not null" json:"accountId"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"updatedAt"`
}

// TableName specifies the table name for ACMEProviderDefault
func (ACMEProviderDefault) TableName() string {
	return "acme_provider_defaults"
}
