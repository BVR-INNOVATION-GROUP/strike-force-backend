package config

import (
	"fmt"
	"os"

	application "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Application"
	auth "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Auth"
	branch "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Branch"
	chat "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Chat"
	college "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/College"
	course "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Course"
	delegatedaccess "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/DelegatedAccess"
	department "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Department"
	dispute "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Dispute"
	invitation "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Invitation"
	milestone "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Milestone"
	notification "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Notification"
	organization "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Organization"
	portfolio "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Portfolio"
	project "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Project"
	student "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Student"
	supervisor "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Supervisor"
	supervisorrequest "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/SupervisorRequest"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectToDB() (*gorm.DB, error) {

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
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

	migrationErr := db.AutoMigrate(&user.User{}, &organization.Organization{}, &branch.Branch{}, &college.College{}, &course.Course{}, &department.Department{}, &project.Project{}, &milestone.Milestone{}, &application.Application{}, &chat.Message{}, &dispute.Dispute{}, &invitation.Invitation{}, &notification.Notification{}, &student.Student{}, &supervisor.Supervisor{}, &supervisorrequest.SupervisorRequest{}, &portfolio.PortfolioItem{}, &auth.PasswordResetToken{}, &delegatedaccess.DelegatedAccess{})

	if migrationErr != nil {
		fmt.Println("Small migration issue: [DB HAS DATA]")
	}

	fmt.Println("Connected to DB successfully")

	return db, nil
}
