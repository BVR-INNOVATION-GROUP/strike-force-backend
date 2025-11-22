package organization

import (
	"gorm.io/gorm"
)

func FindById(db *gorm.DB, id uint) uint {

	var organization Organization

	if err := db.Where("user_id = ?", id).First(&organization).Error; err != nil {
		return 0
	}

	return organization.ID

}
