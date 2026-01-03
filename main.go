package main

import (
	"fmt"
	"log"

	"github.com/BVR-INNOVATION-GROUP/strike-force-backend/config"
	analytics "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Analytics"
	application "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Application"
	auth "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Auth"
	branch "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Branch"
	chat "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Chat"
	college "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/College"
	course "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Course"
	delegatedaccess "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/DelegatedAccess"
	department "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Department"
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
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("Starting application...")

	app := fiber.New()

	app.Use(cors.New())

	// Load .env file if it exists (for local development)
	// In production (Railway), environment variables are set directly
	envErr := godotenv.Load()
	if envErr != nil {
		log.Println("Note: .env file not found. Using environment variables from system (production mode)")
	}

	log.Println("Connecting to database...")
	DB, DBError := config.ConnectToDB()

	if DBError != nil {
		log.Fatal("Failed to connect to DB : " + DBError.Error())
	}
	log.Println("Database connected successfully")

	// Create admin user if it doesn't exist
	if err := user.CreateAdminIfNotExists(DB); err != nil {
		log.Printf("Warning: Failed to create admin user: %v", err)
	} else {
		log.Println("Admin user check completed")
	}

	// Serve static files (uploads)
	app.Static("/uploads", "./uploads")

	user.RegisterRoutes(app, DB)
	auth.RegisterRoutes(app, DB)

	apiV1 := app.Group("/api/v1")
	organization.RegisterRoutes(apiV1, DB)
	department.RegisterRRoutes(apiV1, DB)
	branch.RegisterRoutes(apiV1, DB)
	college.RegisterRoutes(apiV1, DB)
	project.RegisterRoutes(apiV1, DB)
	course.RegisterRoutes(apiV1, DB)
	student.RegisterRoutes(apiV1, DB)
	supervisor.RegisterRoutes(apiV1, DB)
	milestone.RegisterRoutes(apiV1, DB)
	chat.RegisterRoutes(apiV1, DB)
	notification.RegisterRoutes(apiV1, DB)
	application.RegisterRoutes(apiV1, DB)
	invitation.RegisterRoutes(apiV1, DB)
	delegatedaccess.RegisterRoutes(apiV1, DB)
	analytics.RegisterRoutes(apiV1, DB)
	supervisorrequest.RegisterRoutes(apiV1, DB)
	portfolio.RegisterRoutes(apiV1, DB)

	log.Println("All routes registered successfully")

	// Get port from environment (Railway uses PORT, local dev uses APP_PORT)
	port := "3000"

	fmt.Println("Server starting on port " + port)
	if err := app.Listen("0.0.0.0:" + port); err != nil {
		log.Fatal("Failed to start server: " + err.Error())
	}

}
