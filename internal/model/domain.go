package model

// DomainPurpose represents domain purpose
type DomainPurpose string

const (
	DomainPurposeCDN     DomainPurpose = "cdn"
	DomainPurposeGeneral DomainPurpose = "general"
)

// DomainStatus represents domain status
type DomainStatus string

const (
	DomainStatusActive   DomainStatus = "active"
	DomainStatusInactive DomainStatus = "inactive"
)

// Domain represents a domain/zone
type Domain struct {
	BaseModel
	Domain  string        `gorm:"type:varchar(255);uniqueIndex;not null" json:"domain"`
	Purpose DomainPurpose `gorm:"type:enum('cdn','general');default:'cdn'" json:"purpose"`
	Status  DomainStatus  `gorm:"type:enum('active','inactive');default:'active'" json:"status"`
}

// TableName specifies the table name for Domain model
func (Domain) TableName() string {
	return "domains"
}
