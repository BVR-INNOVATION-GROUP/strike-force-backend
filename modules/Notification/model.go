package notification

import (
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"gorm.io/gorm"
)

type Notification struct {
	gorm.Model
	Type    string    `json:"type"`
	Title   string    `json:"title"`
	Message string    `json:"message"`
	Seen    bool      `json:"seen"`
	Link    string    `json:"link"`
	UserID  uint      `json:"user_id"`
	User    user.User `json:"user" gorm:"foreignKey:UserID"`
}
