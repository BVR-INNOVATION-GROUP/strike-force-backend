package supervisorrequest

import (
	project "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Project"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"gorm.io/gorm"
)

// SupervisorRequest represents a request from a student/group to a supervisor
type SupervisorRequest struct {
	gorm.Model
	ProjectID        uint                `json:"projectId"`
	Project          project.Project      `json:"project" gorm:"foreignKey:ProjectID"`
	StudentOrGroupID uint                `json:"studentOrGroupId"` // User ID (for individual) or Group ID
	SupervisorID     uint                `json:"supervisorId"`     // User ID of supervisor
	Supervisor       user.User           `json:"supervisor" gorm:"foreignKey:SupervisorID"`
	Status           string              `json:"status" gorm:"default:'PENDING'"` // PENDING, APPROVED, DENIED
	Message          string              `json:"message" gorm:"type:text"`
}














