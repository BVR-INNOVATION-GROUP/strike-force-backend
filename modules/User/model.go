package user

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Profile struct {
	Avatar   string         `json:"avatar"`
	Bio      string         `json:"bio"`
	Skills   datatypes.JSON `json:"skills" gorm:"type:json"` // JSON field
	Phone    string         `json:"phone"`
	Location string         `json:"location"`
}

type User struct {
	gorm.Model // includes: ID uint, CreatedAt, UpdatedAt, DeletedAt

	Role         string  `json:"role"`
	Email        string  `json:"email" gorm:"unique"`
	Name         string  `json:"name"`
	Password     string  `json:"-"` // Never return password in JSON responses
	Profile      Profile `json:"profile" gorm:"embedded;embeddedPrefix:profile_"`
	Groups       []Group `json:"groups" gorm:"many2many:user_groups"`
	CourseID     uint    `json:"courseId"`
	OrgID        *uint   `json:"orgId,omitempty" gorm:"-"`
	IsBlocked    bool    `json:"isBlocked" gorm:"default:false"`
	LastLoginAt  *time.Time `json:"lastLoginAt,omitempty"`
}
type Group struct {
	gorm.Model
	UserID   uint   `json:"userId"`
	User     User   `json:"user" gorm:"foreignKey:UserID"`
	Members  []User `json:"members" gorm:"many2many:user_groups"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
}
