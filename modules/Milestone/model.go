package milestone

import (
	project "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Project"
	"gorm.io/gorm"
)

type Milestone struct {
	gorm.Model
	ProjectID          int             `json:"project_id"`
	Project            project.Project `json:"project" gorm:"foreignKey:ProjectID"`
	Title              string          `json:"title"`
	Scope              string          `json:"scope"`
	AcceptanceCriteria string          `json:"acceptance_criteria"`
	DueDate            string          `json:"due_date"`
	Amount             int             `json:"amount"`
	Currency           string          `json:"currency"`
	Status             string          `json:"status"`
}
