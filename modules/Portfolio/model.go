package portfolio

import (
	project "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Project"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// PortfolioItem represents a verified portfolio entry from completed work
type PortfolioItem struct {
	gorm.Model
	UserID          uint                `json:"userId"`
	User            user.User           `json:"user" gorm:"foreignKey:UserID"`
	ProjectID       uint                `json:"projectId"`
	Project         project.Project     `json:"project" gorm:"foreignKey:ProjectID"`
	MilestoneID     *uint               `json:"milestoneId,omitempty"` // Optional - for milestone-based entries
	Role            string              `json:"role"`                  // Student's role in the project
	Scope           string              `json:"scope"`                // Description of work done
	Proof           datatypes.JSON      `json:"proof" gorm:"type:json"` // Array of file paths/URLs
	Rating          *float64            `json:"rating,omitempty"`       // Partner rating (1-5)
	Complexity      string              `json:"complexity"`             // LOW, MEDIUM, HIGH
	AmountDelivered float64             `json:"amountDelivered"`       // Amount earned
	Currency        string              `json:"currency" gorm:"default:'USD'"`
	OnTime          bool                `json:"onTime"`                // Was delivered on time
	VerifiedAt      datatypes.Date      `json:"verifiedAt"`            // Auto-created timestamp
}














