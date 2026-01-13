package supervisor

import (
	department "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Department"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"gorm.io/gorm"
)

type Supervisor struct {
	gorm.Model
	UserID       uint                  `json:"userId"`
	User         user.User             `json:"user" gorm:"foreignKey:UserID"`
	DepartmentID uint                  `json:"departmentId"`
	Department   department.Department `json:"department" gorm:"foreignKey:DepartmentID"`
}













