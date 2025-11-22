package main

import (
	"fmt"
	"log"
	"os"

	"github.com/BVR-INNOVATION-GROUP/strike-force-backend/config"
	department "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Department"
	organization "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Organization"
	project "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Project"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {

	app := fiber.New()

	app.Use(cors.New())

	envErr := godotenv.Load()

	if envErr != nil {
		log.Fatal("Failed to load .env")
	}

	DB, DBError := config.ConnectToDB()

	if DBError != nil {
		log.Fatal("Failed to connect to DB : " + DBError.Error())
	}

	user.RegisterRoutes(app, DB)

	apiV1 := app.Group("/api/v1")
	organization.RegisterRoutes(apiV1, DB)
	department.RegisterRRoutes(apiV1, DB)
	project.RegisterRoutes(apiV1, DB)

	fmt.Println("server running on port " + os.Getenv("APP_PORT"))
	app.Listen(":" + os.Getenv("APP_PORT"))

}
