package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"gorm.io/gorm"
)

// checkAndSeed checks if the users table is empty and seeds the database if needed
func checkAndSeed(db *gorm.DB) {
	var userCount int64
	db.Model(&user.User{}).Count(&userCount)

	if userCount == 0 {
		log.Println("Users table is empty. Starting database seeding...")
		runSeed()
		log.Println("Database seeding completed successfully!")
	} else {
		log.Printf("Users table has %d users. Skipping seed.", userCount)
	}
}

// runSeed runs the database seeding by executing the seed command
func runSeed() {
	// Get the path to the seed command
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Println("Warning: Could not determine seed command path. Skipping auto-seed.")
		log.Println("Please run manually: go run cmd/seed/main.go")
		return
	}

	// Get the backend directory
	backendDir := filepath.Dir(filename)
	seedCmdPath := filepath.Join(backendDir, "cmd", "seed", "main.go")

	// Check if seed file exists
	if _, err := os.Stat(seedCmdPath); os.IsNotExist(err) {
		log.Println("Warning: Seed command not found. Skipping auto-seed.")
		log.Println("Please run manually: go run cmd/seed/main.go")
		return
	}

	// Run the seed command with default values
	// Default: 10 orgs, 100 users, 50 projects, 200 applications
	// Use the package path instead of single file to include seeders.go
	log.Println("Running seed command with default values...")
	cmd := exec.Command("go", "run", "./cmd/seed", "10", "100", "50", "200")
	cmd.Dir = backendDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Error running seed command: %v", err)
		log.Println("You can manually seed the database by running: go run ./cmd/seed")
	}
}

