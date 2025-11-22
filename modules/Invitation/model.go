package invitation

import (
	organization "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Organization"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"gorm.io/gorm"
)

type Invitation struct {
	gorm.Model
	OrganizationID uint                      `json:"org_id"`
	Organization   organization.Organization `json:"org" gorm:"foreignKey:OrganizationID"`
	UserID         uint                      `json:"user_id"`
	User           user.User                 `json:"user" gorm:"foreignKey:UserID"`
}
