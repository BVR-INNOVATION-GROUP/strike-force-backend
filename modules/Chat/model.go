package chat

import (
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"gorm.io/gorm"
)

type Message struct {
	gorm.Model
	SenderID int        `json:"sender_id"`
	Sender   user.User  `json:"sender" gorm:"foreignKey:SenderID"`
	Body     string     `json:"body"`
	GroupID  uint       `json:"group_id"`
	Group    user.Group `json:"group" gorm:"foreignKey:GroupID"`
}
