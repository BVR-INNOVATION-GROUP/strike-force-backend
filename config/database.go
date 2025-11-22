package config

import (
	"fmt"
	"os"

	application "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Application"
	chat "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Chat"
	course "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Course"
	department "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Department"
	dispute "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Dispute"
	invitation "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Invitation"
	milestone "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Milestone"
	notification "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Notification"
	organization "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Organization"
	project "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Project"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectToDB() (*gorm.DB, error) {

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		return nil, err
	}

	migrationErr := db.AutoMigrate(&user.User{}, &organization.Organization{}, &course.Course{}, &department.Department{}, &project.Project{}, &milestone.Milestone{}, &application.Application{}, &chat.Message{}, &dispute.Dispute{}, &invitation.Invitation{}, &notification.Notification{})

	if migrationErr != nil {
		fmt.Println("Small migration issue: [DB HAS DATA]")
	}

	fmt.Println("Connected to DB successfully")

	return db, nil
}
