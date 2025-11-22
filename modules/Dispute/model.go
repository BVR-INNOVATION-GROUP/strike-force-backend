package dispute

import (
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Dispute struct {
	gorm.Model
	SubjectType string         `json:"subject_type"`
	Reason      string         `json:"reason"`
	Description string         `json:"description"`
	Evidence    datatypes.JSON `json:"evidence" gorm:"type:json"`
	Status      string         `json:"status" gorm:"default:'pending'"`
	Level       string         `json:"level"`
	IssuerID    uint           `json:"issuer_id"`
	Issuer      user.User      `json:"issuer" gorm:"foreignKey:IssuerID"`
	DefendantID uint           `json:"defendant_id"`
	Defendant   user.User      `json:"defendant" gorm:"foreignKey:DefendantID"`
	Resolution  string         `json:"resolution"`
	ResolvedAt  string         `json:"resolved_at"`
}
