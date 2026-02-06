package main

import (
	"fmt"
	"log"
	"os"

	"github.com/BVR-INNOVATION-GROUP/strike-force-backend/config"
	"github.com/BVR-INNOVATION-GROUP/strike-force-backend/pkg/seed"
	"github.com/joho/godotenv"
)

func main() {
	envErr := godotenv.Load()
	if envErr != nil {
		envErr = godotenv.Load("../.env")
		if envErr != nil {
			log.Println("Warning: Failed to load .env file. Make sure environment variables are set.")
		}
	}

	// Force seeding for CLI - set SEED=true for this run
	os.Setenv("SEED", "true")

	// Use SEED_PASSWORD from env, default to SeedPass123!
	if os.Getenv("SEED_PASSWORD") == "" {
		os.Setenv("SEED_PASSWORD", "SeedPass123!")
	}

	db, err := config.ConnectToDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err := seed.Run(db); err != nil {
		log.Fatal("Seeding failed:", err)
	}

	fmt.Println("\n✅ Database seeding completed successfully!")
	fmt.Println("All users have the password from SEED_PASSWORD env var.")
}
