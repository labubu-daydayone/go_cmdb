package model

// UserStatus represents user status
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
)

// User represents a user in the system
type User struct {
	BaseModel
	Username     string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"`
	Role         string     `gorm:"type:varchar(32);default:'admin'" json:"role"`
	Status       UserStatus `gorm:"type:enum('active','inactive');default:'active'" json:"status"`
}

// TableName specifies the table name for User model
func (User) TableName() string {
	return "users"
}
