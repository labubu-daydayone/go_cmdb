package model

// APIKeyStatus represents API key status
type APIKeyStatus string

const (
	APIKeyStatusActive   APIKeyStatus = "active"
	APIKeyStatusInactive APIKeyStatus = "inactive"
)

// APIKeyProvider represents API key provider
type APIKeyProvider string

const (
	APIKeyProviderCloudflare APIKeyProvider = "cloudflare"
)

// APIKey represents an API key for external services
type APIKey struct {
	BaseModel
	Name      string         `gorm:"type:varchar(128);not null" json:"name"`
	Provider  APIKeyProvider `gorm:"type:varchar(32);not null;index" json:"provider"`
	Account   string         `gorm:"type:varchar(128)" json:"account"`
	APIToken  string         `gorm:"type:varchar(512);not null" json:"-"`
	Status    APIKeyStatus   `gorm:"type:enum('active','inactive');default:'active'" json:"status"`
}

// TableName specifies the table name for APIKey model
func (APIKey) TableName() string {
	return "api_keys"
}

// MaskedToken returns a masked version of the API token
// showing only the last 4 characters (e.g., "****abcd")
func (k *APIKey) MaskedToken() string {
	if len(k.APIToken) <= 4 {
		return "****"
	}
	return "****" + k.APIToken[len(k.APIToken)-4:]
}
