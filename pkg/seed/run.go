package seed

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Organization"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Run seeds the database when SEED=true. Uses SEED_PASSWORD for all user passwords.
// Returns nil if SEED is not true or seeding completed successfully.
func Run(db *gorm.DB) error {
	seedEnv := strings.ToLower(strings.TrimSpace(os.Getenv("SEED")))
	if seedEnv != "true" && seedEnv != "1" && seedEnv != "yes" {
		return nil
	}

	password := os.Getenv("SEED_PASSWORD")
	if password == "" {
		password = "SeedPass123!"
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("seed: failed to hash password: %w", err)
	}
	passwordHash := string(hashedPassword)

	log.Println("SEED=true: Starting database seeding...")
	rand.Seed(time.Now().UnixNano())

	numOrgs := getEnvInt("SEED_ORGS", 10)
	numUsers := getEnvInt("SEED_USERS", 100)
	numProjects := getEnvInt("SEED_PROJECTS", 50)
	numApplications := getEnvInt("SEED_APPLICATIONS", 200)
	numGroups := getEnvInt("SEED_GROUPS", 20)
	numMessages := getEnvInt("SEED_MESSAGES", 500)
	numDisputes := getEnvInt("SEED_DISPUTES", 30)
	numInvitations := getEnvInt("SEED_INVITATIONS", 50)
	numNotifications := getEnvInt("SEED_NOTIFICATIONS", 300)
	numSupervisorRequests := getEnvInt("SEED_SUPERVISOR_REQUESTS", 40)

	organizations := seedOrganizations(db, numOrgs, passwordHash)
	branches := seedBranches(db, organizations)
	colleges := seedColleges(db, organizations)
	departments := seedDepartments(db, organizations, colleges)
	courses := seedCourses(db, departments)
	users := seedUsers(db, organizations, courses, passwordHash, numUsers)
	students := seedStudents(db, users, courses)
	supervisors := seedSupervisors(db, users, departments)
	projects := seedProjects(db, organizations, departments, supervisors, numProjects)
	applications := seedApplications(db, users, projects, numApplications)
	milestones := seedMilestones(db, projects)
	portfolioItems := seedPortfolio(db, users, projects, milestones)
	groups := seedGroups(db, users, numGroups)
	messages := seedChatMessages(db, groups, users, numMessages)
	disputes := seedDisputes(db, users, numDisputes)
	invitations := seedInvitations(db, organizations, departments, numInvitations)
	notifications := seedNotifications(db, users, numNotifications)
	supervisorRequests := seedSupervisorRequests(db, projects, users, groups, numSupervisorRequests)
	delegatedAccesses := seedDelegatedAccesses(db, organizations, users)

	log.Printf("SEED: Created %d organizations, %d branches, %d colleges, %d departments, %d courses",
		len(organizations), len(branches), len(colleges), len(departments), len(courses))
	log.Printf("SEED: Created %d users, %d students, %d projects, %d applications",
		len(users), len(students), len(projects), len(applications))
	log.Printf("SEED: Created %d milestones, %d portfolio items, %d groups, %d messages",
		len(milestones), len(portfolioItems), len(groups), len(messages))
	log.Printf("SEED: Created %d disputes, %d invitations, %d notifications, %d supervisor requests, %d delegated accesses",
		len(disputes), len(invitations), len(notifications), len(supervisorRequests), len(delegatedAccesses))
	log.Printf("SEED: All users have password from SEED_PASSWORD env var")
	log.Println("SEED: Database seeding completed successfully")

	_ = organization.Organization{}
	return nil
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}
