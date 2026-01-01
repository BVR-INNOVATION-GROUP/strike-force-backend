package college

import (
	organization "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Organization"
	"gorm.io/gorm"
)

// College represents a college/faculty within a university organization.
// Benchmarked from the Branch model for structure and relationships.
type College struct {
	gorm.Model
	Name           string                    `json:"name"`
	OrganizationID uint                      `json:"organizationId"`
	Organization   organization.Organization `json:"organization" gorm:"foreignKey:OrganizationID"`
}






