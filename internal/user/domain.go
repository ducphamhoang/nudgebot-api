package user

import (
	"time"

	"nudgebot-api/internal/common"
)

// User represents a user in the system
type User struct {
	ID         common.UserID `gorm:"type:varchar(36);primaryKey" json:"id"`
	TelegramID int64         `gorm:"uniqueIndex;not null" json:"telegram_id"`
	Username   string        `gorm:"type:varchar(255)" json:"username"`
	FirstName  string        `gorm:"type:varchar(255)" json:"first_name,omitempty"`
	LastName   string        `gorm:"type:varchar(255)" json:"last_name,omitempty"`
	CreatedAt  time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for the User model
func (User) TableName() string {
	return "users"
}
