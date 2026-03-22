package directmessage

import (
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"gorm.io/gorm"
)

// Thread is a two-party direct message thread (stable ordering by user IDs).
type Thread struct {
	gorm.Model
	UserASmall  uint      `json:"userASmall" gorm:"uniqueIndex:idx_dm_thread_pair"`
	UserBLarge  uint      `json:"userBLarge" gorm:"uniqueIndex:idx_dm_thread_pair"`
	CreatedByID uint      `json:"createdById"`
	UserA       user.User `json:"-" gorm:"foreignKey:UserASmall"`
	UserB       user.User `json:"-" gorm:"foreignKey:UserBLarge"`
	Creator     user.User `json:"-" gorm:"foreignKey:CreatedByID"`
}

func (Thread) TableName() string { return "direct_message_threads" }

// Message is a single DM in a thread.
type Message struct {
	gorm.Model
	ThreadID uint      `json:"threadId"`
	SenderID uint      `json:"senderId"`
	Body     string    `json:"body"`
	Thread   Thread    `json:"-" gorm:"foreignKey:ThreadID"`
	Sender   user.User `json:"sender" gorm:"foreignKey:SenderID"`
}

func (Message) TableName() string { return "direct_messages" }
