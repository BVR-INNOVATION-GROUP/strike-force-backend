package delegatedaccess

import (
	organization "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Organization"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"gorm.io/gorm"
)

// DelegatedAccess represents a delegated user who can access the same organization as the delegating admin
type DelegatedAccess struct {
	gorm.Model
	DelegatedUserID uint                      `json:"delegatedUserId" gorm:"not null"`
	DelegatedUser   user.User                 `json:"delegatedUser" gorm:"foreignKey:DelegatedUserID"`
	DelegatorID     uint                      `json:"delegatorId" gorm:"not null"` // The university admin who delegated access
	Delegator       user.User                 `json:"delegator" gorm:"foreignKey:DelegatorID"`
	OrganizationID  uint                      `json:"organizationId" gorm:"not null"`
	Organization    organization.Organization `json:"organization" gorm:"foreignKey:OrganizationID"`
	IsActive        bool                      `json:"isActive" gorm:"default:true"`
}



