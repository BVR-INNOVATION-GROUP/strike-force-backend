package config

import (
	"fmt"
	"os"
	"strings"

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

// ensureStudentDNASnapshotColumns adds DNA Snapshot columns to students table if missing.
// Uses IF NOT EXISTS to avoid errors when columns already exist (e.g. re-runs, existing data).
// All new columns use DEFAULT so existing rows get safe values.
func ensureStudentDNASnapshotColumns(db *gorm.DB) {
	columns := []struct {
		sql  string
		name string
	}{
		{`ALTER TABLE students ADD COLUMN IF NOT EXISTS has_completed_dna_snapshot BOOLEAN DEFAULT false`, "has_completed_dna_snapshot"},
		{`ALTER TABLE students ADD COLUMN IF NOT EXISTS dna_snapshot_responses JSONB`, "dna_snapshot_responses"},
		{`ALTER TABLE students ADD COLUMN IF NOT EXISTS dna_archetype VARCHAR(100)`, "dna_archetype"},
		{`ALTER TABLE students ADD COLUMN IF NOT EXISTS dna_snapshot_completed_at TIMESTAMP WITH TIME ZONE`, "dna_snapshot_completed_at"},
	}
	for _, col := range columns {
		if err := db.Exec(col.sql).Error; err != nil {
			errStr := strings.ToLower(err.Error())
			// Fallback: some PostgreSQL versions may not support IF NOT EXISTS on ADD COLUMN
			if !strings.Contains(errStr, "already exists") && !strings.Contains(errStr, "duplicate column") {
				fmt.Printf("Warning: Could not ensure %s column exists: %v\n", col.name, err)
			}
		}
	}
}

// ensureImpersonationLogsTable creates impersonation_logs table if not exists
func ensureImpersonationLogsTable(db *gorm.DB) {
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS impersonation_logs (
			id SERIAL PRIMARY KEY,
			impersonator_id INTEGER NOT NULL,
			target_user_id INTEGER NOT NULL,
			target_email VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`).Error
	if err != nil {
		fmt.Printf("Warning: Could not ensure impersonation_logs table: %v\n", err)
	}
}

// ensureAdminAuditLogsTable creates admin_audit_logs table if not exists
func ensureAdminAuditLogsTable(db *gorm.DB) {
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS admin_audit_logs (
			id SERIAL PRIMARY KEY,
			admin_id INTEGER NOT NULL,
			action VARCHAR(100) NOT NULL,
			target_type VARCHAR(50),
			target_id INTEGER,
			details TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`).Error
	if err != nil {
		fmt.Printf("Warning: Could not ensure admin_audit_logs table: %v\n", err)
	}
}

// ensureLoginPageLogosTable creates login_page_logos table if not exists
func ensureLoginPageLogosTable(db *gorm.DB) {
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS login_page_logos (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			logo_url VARCHAR(512) NOT NULL,
			alt_text VARCHAR(255),
			sort_order INTEGER DEFAULT 0,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`).Error
	if err != nil {
		fmt.Printf("Warning: Could not ensure login_page_logos table: %v\n", err)
	}
}

// ensureStudentIDColumn manually ensures the student_id column exists.
// Uses IF NOT EXISTS to avoid errors when column already exists (safe for existing data).
// This is a fallback if AutoMigrate fails to create it.
func ensureStudentIDColumn(db *gorm.DB) {
	// ADD COLUMN IF NOT EXISTS - no DEFAULT needed; existing rows get NULL, which is valid
	err := db.Exec(`ALTER TABLE students ADD COLUMN IF NOT EXISTS student_id VARCHAR(50)`).Error
	if err != nil {
		errStr := strings.ToLower(err.Error())
		if !strings.Contains(errStr, "already exists") && !strings.Contains(errStr, "duplicate column") {
			fmt.Printf("Warning: Could not ensure student_id column exists: %v\n", err)
			return
		}
	}
	
	// Try to add unique index (PostgreSQL allows multiple NULLs in unique index)
	// Use IF NOT EXISTS to avoid errors if index already exists
	err = db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_students_student_id 
		ON students (student_id) 
		WHERE student_id IS NOT NULL
	`).Error
	
	if err != nil {
		// Check if index already exists
		errStr := strings.ToLower(err.Error())
		if !strings.Contains(errStr, "already exists") {
			fmt.Printf("Warning: Failed to create unique index on student_id: %v\n", err)
		}
	}
}

func ConnectToDB() (*gorm.DB, error) {
	// Get required database configuration from environment
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	sslMode := os.Getenv("DB_SSLMODE")

	// Validate required environment variables
	var missingVars []string
	if dbHost == "" {
		missingVars = append(missingVars, "DB_HOST")
	}
	if dbPort == "" {
		missingVars = append(missingVars, "DB_PORT")
	}
	if dbUser == "" {
		missingVars = append(missingVars, "DB_USER")
	}
	if dbPassword == "" {
		missingVars = append(missingVars, "DB_PASSWORD")
	}
	if dbName == "" {
		missingVars = append(missingVars, "DB_NAME")
	}

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required database environment variables: %v. Please set these environment variables before starting the application", missingVars)
	}

	// Default SSL mode to "disable" for local development
	if sslMode == "" {
		sslMode = "disable"
	}

	// Construct DSN
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost,
		dbPort,
		dbUser,
		dbPassword,
		dbName,
		sslMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		return nil, fmt.Errorf("failed to initialize database, got error %w", err)
	}

	migrationErr := db.AutoMigrate(&user.User{}, &organization.Organization{}, &branch.Branch{}, &college.College{}, &course.Course{}, &department.Department{}, &project.Project{}, &milestone.Milestone{}, &application.Application{}, &chat.Message{}, &dispute.Dispute{}, &invitation.Invitation{}, &notification.Notification{}, &student.Student{}, &supervisor.Supervisor{}, &supervisorrequest.SupervisorRequest{}, &portfolio.PortfolioItem{}, &auth.PasswordResetToken{}, &delegatedaccess.DelegatedAccess{})

	if migrationErr != nil {
		fmt.Printf("Migration issue: %v\n", migrationErr)
	}
	
	// Always ensure student_id column exists (defensive check)
	// This handles cases where AutoMigrate fails silently or partially
	ensureStudentIDColumn(db)

	// Ensure DNA Snapshot columns exist (for databases created before these were added)
	ensureStudentDNASnapshotColumns(db)

	// Ensure impersonation_logs table exists for admin audit
	ensureImpersonationLogsTable(db)

	// Ensure login_page_logos table exists for branding
	ensureLoginPageLogosTable(db)

	// Ensure admin_audit_logs table exists for audit
	ensureAdminAuditLogsTable(db)

	fmt.Println("Connected to DB successfully")

	return db, nil
}
