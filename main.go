package main

import (
	"fmt"
	"log"
	"os"

	"github.com/BVR-INNOVATION-GROUP/strike-force-backend/config"
	"github.com/BVR-INNOVATION-GROUP/strike-force-backend/pkg/seed"
	admin "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Admin"
	analytics "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Analytics"
	application "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Application"
	auth "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Auth"
	branch "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Branch"
	chat "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Chat"
	college "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/College"
	course "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Course"
	delegatedaccess "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/DelegatedAccess"
	directmessage "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/DirectMessage"
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
	"gorm.io/gorm"
)

// initializeSuperAdmin creates a super admin user if one doesn't exist
func initializeSuperAdmin(db *gorm.DB) error {
	// Get super admin credentials from environment variables
	superAdminEmail := os.Getenv("SUPER_ADMIN_EMAIL")
	if superAdminEmail == "" {
		superAdminEmail = "admin@strikeforce.com" // Default email
	}

	superAdminPassword := os.Getenv("SUPER_ADMIN_PASSWORD")
	if superAdminPassword == "" {
		superAdminPassword = "admin123" // Default password - should be changed in production
	}

	superAdminName := os.Getenv("SUPER_ADMIN_NAME")
	if superAdminName == "" {
		superAdminName = "Super Admin" // Default name
	}

	// Check if any super admin already exists
	var existingSuperAdmin user.User
	result := db.Where("role = ?", "super-admin").First(&existingSuperAdmin)
	
	if result.Error == nil {
		// Super admin already exists
		log.Printf("Super admin already exists: %s", existingSuperAdmin.Email)
		return nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		// Some other error occurred
		return fmt.Errorf("error checking for super admin: %w", result.Error)
	}

	// Check if the email is already taken by another user
	var existingUser user.User
	if err := db.Where("email = ?", superAdminEmail).First(&existingUser).Error; err == nil {
		// Email exists but not as super-admin, update the role
		existingUser.Role = "super-admin"
		if err := db.Save(&existingUser).Error; err != nil {
			return fmt.Errorf("failed to update user to super admin: %w", err)
		}
		log.Printf("Updated existing user to super admin: %s", superAdminEmail)
		return nil
	} else if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("error checking for existing user: %w", err)
	}

	// Create super admin
	hashedPassword := user.GenerateHash(superAdminPassword)
	if hashedPassword == "" {
		return fmt.Errorf("failed to hash super admin password")
	}

	superAdmin := user.User{
		Email:    superAdminEmail,
		Name:     superAdminName,
		Role:     "super-admin",
		Password: hashedPassword,
		Profile: user.Profile{
			Bio:      "Platform administrator",
			Location: "Global",
		},
	}

	if err := db.Create(&superAdmin).Error; err != nil {
		return fmt.Errorf("failed to create super admin: %w", err)
	}

	log.Printf("Super admin created successfully: %s (password: %s)", superAdminEmail, superAdminPassword)
	return nil
}

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

	// Initialize super admin if it doesn't exist
	log.Println("Checking for super admin...")
	if err := initializeSuperAdmin(DB); err != nil {
		log.Printf("Warning: Failed to initialize super admin: %v", err)
	} else {
		log.Println("Super admin check completed")
	}

	// Seed database when SEED=true (uses SEED_PASSWORD for all seeded user passwords)
	if err := seed.Run(DB); err != nil {
		log.Printf("Warning: Database seeding failed: %v", err)
	}

	// Serve static files (uploads)
	app.Static("/uploads", "./uploads")

	user.RegisterRoutes(app, DB)
	auth.RegisterRoutes(app, DB)

	apiV1 := app.Group("/api/v1")
	// Public route for login page logos (no auth)
	apiV1.Get("/login-logos", func(c *fiber.Ctx) error {
		return admin.GetPublicLoginLogos(c, DB)
	})
	apiV1.Get("/organizations/public-logos", func(c *fiber.Ctx) error {
		return organization.GetPublicLogos(c, DB)
	})
	organization.RegisterRoutes(apiV1, DB)
	department.RegisterRRoutes(apiV1, DB)
	branch.RegisterRoutes(apiV1, DB)
	college.RegisterRoutes(apiV1, DB)
	project.RegisterRoutes(apiV1, DB)
	course.RegisterRoutes(apiV1, DB)
	student.RegisterRoutes(apiV1, DB)
	// University-admin / delegated-admin: delete student by user ID (student must be in their org)
	apiV1.Delete("/students/user/:userId", user.JWTProtect([]string{"university-admin", "delegated-admin"}), func(c *fiber.Ctx) error {
		return admin.DeleteStudentByUserID(c, DB)
	})
	supervisor.RegisterRoutes(apiV1, DB)
	milestone.RegisterRoutes(apiV1, DB)
	chat.RegisterRoutes(apiV1, DB)
	notification.RegisterRoutes(apiV1, DB)
	application.RegisterRoutes(apiV1, DB)
	invitation.RegisterRoutes(apiV1, DB)
	delegatedaccess.RegisterRoutes(apiV1, DB)
	directmessage.RegisterRoutes(apiV1, DB)
	analytics.RegisterRoutes(apiV1, DB)
	admin.RegisterRoutes(apiV1, DB)
	supervisorrequest.RegisterRoutes(apiV1, DB)
	portfolio.RegisterRoutes(apiV1, DB)

	log.Println("All routes registered successfully")

	// Get port from environment (Railway uses PORT, local dev uses APP_PORT)
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("APP_PORT")
	}
	if port == "" {
		port = "3000" // Default fallback
	}

	fmt.Println("Server starting on port " + port)
	if err := app.Listen("0.0.0.0:" + port); err != nil {
		log.Fatal("Failed to start server: " + err.Error())
	}

}
