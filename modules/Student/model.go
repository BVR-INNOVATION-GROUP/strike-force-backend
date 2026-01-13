package student

import (
	"time"

	branch "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Branch"
	course "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Course"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Student struct {
	gorm.Model
	UserID           uint          `json:"userId"`
	User             user.User     `json:"user" gorm:"foreignKey:UserID"`
	StudentID        *string       `json:"studentId,omitempty" gorm:"type:varchar(50);uniqueIndex;default:null"`
	CourseID         uint          `json:"courseId"`
	Course           course.Course `json:"course" gorm:"foreignKey:CourseID"`
	BranchID         *uint         `json:"branchId,omitempty"`
	Branch           *branch.Branch `json:"branch,omitempty" gorm:"foreignKey:BranchID"`
	Gender           string         `json:"gender" gorm:"type:varchar(50)"`
	District         string         `json:"district" gorm:"type:varchar(100)"`
	UniversityBranch string         `json:"universityBranch" gorm:"type:varchar(100)"` // Kept for backward compatibility
	BirthYear        int            `json:"birthYear" gorm:"type:integer"`
	EnrollmentYear   int            `json:"enrollmentYear" gorm:"type:integer"`
	// DNA Snapshot fields
	HasCompletedDNASnapshot bool            `json:"hasCompletedDnaSnapshot" gorm:"default:false"`
	DNASnapshotResponses    *datatypes.JSON `json:"dnaSnapshotResponses,omitempty" gorm:"type:json"`
	DNAArchetype            *string         `json:"dnaArchetype,omitempty" gorm:"type:varchar(100)"`
	DNASnapshotCompletedAt  *time.Time     `json:"dnaSnapshotCompletedAt,omitempty"`
}
