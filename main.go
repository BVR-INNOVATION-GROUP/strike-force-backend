package main

import (
	"fmt"
	"log"
	"os"

	"github.com/BVR-INNOVATION-GROUP/strike-force-backend/config"
	analytics "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Analytics"
	application "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Application"
	auth "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Auth"
	chat "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Chat"
	course "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Course"
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

	// Check if users table is empty and seed if needed
	checkAndSeed(DB)

	// Serve static files (uploads)
	app.Static("/uploads", "./uploads")

	user.RegisterRoutes(app, DB)
	auth.RegisterRoutes(app, DB)

	apiV1 := app.Group("/api/v1")
	organization.RegisterRoutes(apiV1, DB)
	department.RegisterRRoutes(apiV1, DB)
	project.RegisterRoutes(apiV1, DB)
	course.RegisterRoutes(apiV1, DB)
	student.RegisterRoutes(apiV1, DB)
	supervisor.RegisterRoutes(apiV1, DB)
	milestone.RegisterRoutes(apiV1, DB)
	chat.RegisterRoutes(apiV1, DB)
	notification.RegisterRoutes(apiV1, DB)
	application.RegisterRoutes(apiV1, DB)
	invitation.RegisterRoutes(apiV1, DB)
	analytics.RegisterRoutes(apiV1, DB)
	supervisorrequest.RegisterRoutes(apiV1, DB)
	portfolio.RegisterRoutes(apiV1, DB)

	log.Println("All routes registered successfully")

	// Get port from environment (Railway uses PORT, local dev uses APP_PORT)
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("APP_PORT")
	}
	if port == "" {
		port = "8080" // Default fallback
		log.Println("Warning: No PORT or APP_PORT set, using default 8080")
	}

	fmt.Println("Server starting on port " + port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatal("Failed to start server: " + err.Error())
	}

}
