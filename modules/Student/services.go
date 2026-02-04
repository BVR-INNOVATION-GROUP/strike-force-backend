package student

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	branch "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Branch"
	course "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Course"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CreateStudentRequest struct {
	Email            string       `json:"email"`
	Name             string       `json:"name"`
	Profile          user.Profile `json:"profile,omitempty"`
	Gender           string       `json:"gender,omitempty"`
	District         string       `json:"district,omitempty"`
	UniversityBranch string       `json:"universityBranch,omitempty"` // Deprecated, use BranchID
	BranchID         *uint        `json:"branchId,omitempty"`
	BirthYear        int          `json:"birthYear,omitempty"`
	EnrollmentYear   int          `json:"enrollmentYear,omitempty"`
}

type BulkStudentsRequest struct {
	Students []CreateStudentRequest `json:"students"`
}

// isDuplicateEmailError returns true if the error is a PostgreSQL unique constraint violation on email
func isDuplicateEmailError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "23505") ||
		(strings.Contains(errStr, "unique constraint") && strings.Contains(errStr, "email")) ||
		strings.Contains(errStr, "duplicate key")
}

// enrollExistingUserInCourse adds an existing user as a student to a course
func enrollExistingUserInCourse(db *gorm.DB, courseId uint64, existingUser user.User, req CreateStudentRequest) (Student, string, error) {
	var courseRecord course.Course
	if err := db.First(&courseRecord, courseId).Error; err != nil {
		return Student{}, "", fmt.Errorf("failed to load course: %w", err)
	}

	var branchRecord *branch.Branch
	if req.BranchID != nil {
		var branch branch.Branch
		if err := db.First(&branch, *req.BranchID).Error; err == nil {
			branchRecord = &branch
		}
	}

	enrollmentYear := req.EnrollmentYear
	if enrollmentYear == 0 {
		enrollmentYear = time.Now().Year()
	}

	studentID, err := GenerateStudentID(db, courseRecord, branchRecord, req.District, enrollmentYear)
	if err != nil {
		return Student{}, "", fmt.Errorf("failed to generate student ID: %w", err)
	}

	student := Student{
		UserID:           existingUser.ID,
		StudentID:        &studentID,
		CourseID:         uint(courseId),
		BranchID:         req.BranchID,
		Gender:           req.Gender,
		District:         req.District,
		UniversityBranch: req.UniversityBranch,
		BirthYear:        req.BirthYear,
		EnrollmentYear:   enrollmentYear,
	}

	if err := db.Create(&student).Error; err != nil {
		return Student{}, "", fmt.Errorf("failed to create student record: %w", err)
	}

	db.Preload("User").Preload("Course").Preload("Branch").First(&student, student.ID)
	// Empty password - no email to send since user already has an account
	return student, "", nil
}

func createStudentForCourse(db *gorm.DB, courseId uint64, req CreateStudentRequest) (Student, string, error) {
	if req.Email == "" {
		return Student{}, "", fmt.Errorf("email is required")
	}
	if req.Name == "" {
		return Student{}, "", fmt.Errorf("name is required")
	}

	var existingUser user.User
	if err := db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		// User exists - enroll them in this course if not already enrolled
		var existingStudent Student
		if err := db.Where("user_id = ? AND course_id = ?", existingUser.ID, courseId).First(&existingStudent).Error; err == nil {
			return Student{}, "", fmt.Errorf("student with email %s is already enrolled in this course", req.Email)
		}
		// Enroll existing user in the course (no new user creation, no password email)
		return enrollExistingUserInCourse(db, courseId, existingUser, req)
	}

	randomPassword, err := GenerateRandomPassword(12)
	if err != nil {
		return Student{}, "", fmt.Errorf("failed to generate password: %w", err)
	}

	newUser := user.User{
		Email:    req.Email,
		Name:     req.Name,
		Role:     "student",
		Profile:  req.Profile,
		Password: user.GenerateHash(randomPassword),
	}

	if err := db.Create(&newUser).Error; err != nil {
		if isDuplicateEmailError(err) {
			return Student{}, "", fmt.Errorf("a user with email %s already exists. You can add them to this course using the same email", req.Email)
		}
		return Student{}, "", fmt.Errorf("failed to create user: %w", err)
	}

	// Load course and branch for ID generation
	var courseRecord course.Course
	if err := db.First(&courseRecord, courseId).Error; err != nil {
		db.Delete(&newUser)
		return Student{}, "", fmt.Errorf("failed to load course: %w", err)
	}

	var branchRecord *branch.Branch
	if req.BranchID != nil {
		var branch branch.Branch
		if err := db.First(&branch, *req.BranchID).Error; err == nil {
			branchRecord = &branch
		}
	}

	// Set default enrollment year if not provided
	enrollmentYear := req.EnrollmentYear
	if enrollmentYear == 0 {
		enrollmentYear = time.Now().Year() // Default to current year
	}

	// Generate student ID
	studentID, err := GenerateStudentID(db, courseRecord, branchRecord, req.District, enrollmentYear)
	if err != nil {
		db.Delete(&newUser)
		return Student{}, "", fmt.Errorf("failed to generate student ID: %w", err)
	}

	student := Student{
		UserID:           newUser.ID,
		StudentID:        &studentID,
		CourseID:         uint(courseId),
		BranchID:         req.BranchID,
		Gender:           req.Gender,
		District:         req.District,
		UniversityBranch: req.UniversityBranch, // Keep for backward compatibility
		BirthYear:        req.BirthYear,
		EnrollmentYear:   enrollmentYear,
	}

	if err := db.Create(&student).Error; err != nil {
		db.Delete(&newUser)
		return Student{}, "", fmt.Errorf("failed to create student record: %w", err)
	}

	db.Preload("User").Preload("Course").Preload("Branch").First(&student, student.ID)
	return student, randomPassword, nil
}

func Create(c *fiber.Ctx, db *gorm.DB) error {

	var student Student
	if err := c.BodyParser(&student); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid student details"})
	}

	UserID := c.Locals("user_id").(uint)
	CreatorID := course.FindById(db, student.CourseID)

	if UserID != CreatorID {
		return c.Status(403).JSON(fiber.Map{"msg": "you're not allowed to add students"})
	}

	tmpUser := student.User
	tmpUser.Role = "student"

	if tmpUser.Email == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "provide the email"})
	}

	if tmpUser.Name == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "provide your full name"})
	}

	// Check if user already exists
	var existingUser user.User
	if err := db.Where("email = ?", tmpUser.Email).First(&existingUser).Error; err == nil {
		return c.Status(402).JSON(fiber.Map{"msg": "user with email " + tmpUser.Email + " already exists"})
	}

	// Hash password if provided
	if tmpUser.Password != "" {
		tmpUser.Password = user.GenerateHash(tmpUser.Password)
	} else {
		// Generate a default password if not provided
		tmpUser.Password = user.GenerateHash("changeme123")
	}

	// Create the user first
	if err := db.Create(&tmpUser).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to create user: " + err.Error()})
	}

	// Set the user ID for the student record
	student.UserID = tmpUser.ID
	// CourseID is already set from the request body

	// Load course and branch for ID generation
	var courseRecord course.Course
	if err := db.First(&courseRecord, student.CourseID).Error; err != nil {
		db.Delete(&tmpUser)
		return c.Status(400).JSON(fiber.Map{"msg": "failed to load course: " + err.Error()})
	}

	var branchRecord *branch.Branch
	if student.BranchID != nil {
		var branch branch.Branch
		if err := db.First(&branch, *student.BranchID).Error; err == nil {
			branchRecord = &branch
		}
	}

	// Set default enrollment year if not provided
	enrollmentYear := student.EnrollmentYear
	if enrollmentYear == 0 {
		enrollmentYear = time.Now().Year() // Default to current year
	}

	// Generate student ID
	studentID, err := GenerateStudentID(db, courseRecord, branchRecord, student.District, enrollmentYear)
	if err != nil {
		db.Delete(&tmpUser)
		return c.Status(400).JSON(fiber.Map{"msg": "failed to generate student ID: " + err.Error()})
	}
	student.StudentID = &studentID

	// Create the student record
	if err := db.Create(&student).Error; err != nil {
		// Rollback: delete the user if student creation fails
		db.Delete(&tmpUser)
		return c.Status(400).JSON(fiber.Map{"msg": "failed to add student: " + err.Error()})
	}

	// Reload student with relations
	db.Preload("User").Preload("Course").First(&student, student.ID)

	return c.Status(201).JSON(fiber.Map{"msg": "student created successfully", "data": student})
}

// CreateForCourse creates a student for a specific course (takes courseId as param)
func CreateForCourse(c *fiber.Ctx, db *gorm.DB) error {
	courseIdParam := c.Params("courseId")
	courseId, err := strconv.ParseUint(courseIdParam, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid course ID"})
	}

	// Verify course exists
	var courseRecord course.Course
	if err := db.First(&courseRecord, courseId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "course not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to find course"})
	}

	var req CreateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid student details: " + err.Error()})
	}

	student, password, err := createStudentForCourse(db, courseId, req)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": err.Error()})
	}

	// Only send password email for newly created users (existing users already have accounts)
	credentialsSent := false
	if password != "" {
		if err := SendPasswordEmail(req.Email, req.Name, password); err != nil {
			fmt.Printf("Warning: Failed to send password email to %s: %v\n", req.Email, err)
		} else {
			credentialsSent = true
		}
	}

	msg := "student created successfully"
	if !credentialsSent && password == "" {
		msg = "student added to course successfully (existing user - they can log in with their current credentials)"
	}

	return c.Status(201).JSON(fiber.Map{"msg": msg, "data": student, "credentialsSent": credentialsSent})
}

func FindByCourse(c *fiber.Ctx, db *gorm.DB) error {

	var students []Student
	course := c.Query("course")
	universityId := c.Query("universityId")
	courseId := c.Query("courseId")

	// Handle universityId query parameter
	if universityId != "" {
		UniversityID, err := strconv.ParseUint(universityId, 10, 64)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "invalid university ID"})
		}

		// Find students through course -> department -> organization relationship
		// First get all courses for departments in this organization
		var courseIds []uint
		if err := db.Table("courses").
			Select("courses.id").
			Joins("JOIN departments ON courses.department_id = departments.id").
			Where("departments.organization_id = ?", UniversityID).
			Pluck("courses.id", &courseIds).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to get courses for university"})
		}

		if len(courseIds) == 0 {
			return c.JSON(fiber.Map{"data": []Student{}})
		}

		// Now get students for those courses
		if err := db.Where("course_id IN ?", courseIds).
			Preload("User").
			Preload("Course").
			Preload("Course.Department").
			Preload("Course.Department.Organization").
			Preload("Branch").
			Find(&students).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to get students"})
		}

		return c.JSON(fiber.Map{"data": students})
	}

	// Handle courseId query parameter
	if courseId != "" {
		CourseID, err := strconv.ParseUint(courseId, 10, 64)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "invalid course ID"})
		}

		if err := db.Where("course_id = ?", CourseID).
			Preload("User").
			Preload("Course").
			Preload("Course.Department").
			Preload("Course.Department.Organization").
			Preload("Branch").
			Find(&students).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to get students"})
		}

		return c.JSON(fiber.Map{"data": students})
	}

	// Handle legacy "course" query parameter
	if course != "" {
		CourseID, err := strconv.ParseUint(course, 10, 64)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "missing course info"})
		}

		if err := db.Where("course_id = ?", CourseID).
			Preload("User").
			Preload("Course").
			Preload("Course.Department").
			Preload("Course.Department.Organization").
			Preload("Branch").
			Find(&students).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to get students"})
		}

		return c.JSON(fiber.Map{"data": students})
	}

	return c.Status(400).JSON(fiber.Map{"msg": "missing course or universityId parameter"})

}

type UpdateStudentRequest struct {
	Name           string `json:"name,omitempty"`
	Gender         string `json:"gender,omitempty"`
	District       string `json:"district,omitempty"`
	BranchID       *uint  `json:"branchId,omitempty"`
	BirthYear      int    `json:"birthYear,omitempty"`
	EnrollmentYear int    `json:"enrollmentYear,omitempty"`
}

// Update updates a student's information
func Update(c *fiber.Ctx, db *gorm.DB) error {
	studentIdParam := c.Params("id")
	studentId, err := strconv.ParseUint(studentIdParam, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid student ID"})
	}

	var req UpdateStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid request: " + err.Error()})
	}

	var student Student
	if err := db.First(&student, studentId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "student not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to find student"})
	}

	// Update student fields if provided
	if req.Name != "" {
		// Update user name as well
		var userRecord user.User
		if err := db.First(&userRecord, student.UserID).Error; err == nil {
			userRecord.Name = req.Name
			db.Save(&userRecord)
		}
	}
	if req.Gender != "" {
		student.Gender = req.Gender
	}
	if req.District != "" {
		student.District = req.District
	}
	if req.BranchID != nil {
		student.BranchID = req.BranchID
	}
	if req.BirthYear > 0 {
		student.BirthYear = req.BirthYear
	}
	if req.EnrollmentYear > 0 {
		student.EnrollmentYear = req.EnrollmentYear
	}

	if err := db.Save(&student).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to update student: " + err.Error()})
	}

	// Reload with relations
	db.Preload("User").Preload("Course").Preload("Branch").First(&student, student.ID)

	return c.JSON(fiber.Map{"msg": "student updated successfully", "data": student})
}

func CreateBulkForCourse(c *fiber.Ctx, db *gorm.DB) error {
	courseIdParam := c.Params("courseId")
	courseId, err := strconv.ParseUint(courseIdParam, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid course ID"})
	}

	var courseRecord course.Course
	if err := db.First(&courseRecord, courseId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "course not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to find course"})
	}

	var req BulkStudentsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid student details: " + err.Error()})
	}

	if len(req.Students) == 0 {
		return c.Status(400).JSON(fiber.Map{"msg": "no students provided"})
	}

	type BulkResult struct {
		Student Student `json:"student"`
		Error   string  `json:"error,omitempty"`
	}

	var successes []Student
	var failures []BulkResult

	for _, studentReq := range req.Students {
		student, password, err := createStudentForCourse(db, courseId, studentReq)
		if err != nil {
			failures = append(failures, BulkResult{Error: fmt.Sprintf("%s (%s)", err.Error(), studentReq.Email)})
			continue
		}

		if password != "" {
			if err := SendPasswordEmail(studentReq.Email, studentReq.Name, password); err != nil {
				fmt.Printf("Warning: Failed to send password email to %s: %v\n", studentReq.Email, err)
			}
		}

		successes = append(successes, student)
	}

	status := fiber.Map{
		"msg":       "bulk student creation completed",
		"created":   len(successes),
		"failed":    len(failures),
		"successes": successes,
		"errors":    failures,
		"course_id": courseId,
	}

	if len(successes) == 0 {
		return c.Status(400).JSON(status)
	}

	return c.Status(201).JSON(status)
}

// DNA Snapshot types and functions
type DNASnapshotRequest struct {
	Responses map[string]string `json:"responses"` // question_id -> answer
}

type DNAArchetype struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Traits      []string `json:"traits"`
}

// CalculateDNAArchetype determines the student's DNA archetype based on their responses
func CalculateDNAArchetype(responses map[string]string) DNAArchetype {
	// Scoring system for different archetypes
	scores := map[string]int{
		"builder":      0,
		"explorer":     0,
		"strategist":   0,
		"collaborator": 0,
	}

	// Question 1: Orientation
	if responses["q1"] == "I jump in and figure things out as I go" {
		scores["builder"] += 3
		scores["explorer"] += 2
	} else if responses["q1"] == "I observe first, then act carefully" {
		scores["strategist"] += 3
		scores["collaborator"] += 2
	} else if responses["q1"] == "I prefer structured guidance before acting" {
		scores["collaborator"] += 3
	} else if responses["q1"] == "I wait until I'm confident I can deliver well" {
		scores["strategist"] += 2
	}

	// Question 2: Energy source
	if responses["q2"] == "Solving real-world problems" {
		scores["builder"] += 3
	} else if responses["q2"] == "Learning new skills" {
		scores["explorer"] += 3
	} else if responses["q2"] == "Working with ambitious people" {
		scores["collaborator"] += 3
	} else if responses["q2"] == "Building something of my own" {
		scores["builder"] += 2
		scores["explorer"] += 2
	}

	// Question 3: Environment
	if responses["q3"] == "Fast-moving and unstructured" {
		scores["explorer"] += 3
		scores["builder"] += 2
	} else if responses["q3"] == "Clear goals with some flexibility" {
		scores["builder"] += 3
		scores["strategist"] += 2
	} else if responses["q3"] == "Well-defined roles and expectations" {
		scores["collaborator"] += 3
	} else if responses["q3"] == "Independent, self-directed work" {
		scores["explorer"] += 2
		scores["strategist"] += 2
	}

	// Question 4: Exposure level
	if responses["q4"] == "Leading or owning project outcomes" {
		scores["builder"] += 3
		scores["strategist"] += 2
	} else if responses["q4"] == "Active contributor on real projects" {
		scores["builder"] += 2
		scores["collaborator"] += 2
	}

	// Question 5: Team role
	if responses["q5"] == "Organises and coordinates" {
		scores["strategist"] += 3
		scores["collaborator"] += 2
	} else if responses["q5"] == "Builds or executes" {
		scores["builder"] += 3
	} else if responses["q5"] == "Thinks through strategy and direction" {
		scores["strategist"] += 3
		scores["explorer"] += 2
	} else if responses["q5"] == "Supports and improves what exists" {
		scores["collaborator"] += 3
	}

	// Question 6: Ownership comfort
	if responses["q6"] == "I like full ownership and accountability" {
		scores["builder"] += 3
		scores["strategist"] += 2
	} else if responses["q6"] == "I'm comfortable owning parts of work" {
		scores["builder"] += 2
		scores["collaborator"] += 2
	}

	// Question 7: Handling mistakes
	if responses["q7"] == "Push through independently" {
		scores["builder"] += 2
		scores["explorer"] += 2
	} else if responses["q7"] == "Ask for feedback and support" {
		scores["collaborator"] += 3
	} else if responses["q7"] == "Step back and reassess direction" {
		scores["strategist"] += 3
	}

	// Question 8: Future direction
	if responses["q8"] == "Entrepreneurship or venture building" {
		scores["builder"] += 3
		scores["explorer"] += 2
	} else if responses["q8"] == "Freelance / independent work" {
		scores["explorer"] += 3
	} else if responses["q8"] == "Employment and career growth" {
		scores["collaborator"] += 2
		scores["strategist"] += 2
	}

	// Question 9: StrikeForce value
	if responses["q9"] == "Real project experience" {
		scores["builder"] += 3
	} else if responses["q9"] == "Clarity on my strengths" {
		scores["explorer"] += 2
		scores["strategist"] += 2
	} else if responses["q9"] == "Networks and mentors" {
		scores["collaborator"] += 3
	} else if responses["q9"] == "A place to build long-term credibility" {
		scores["strategist"] += 2
		scores["builder"] += 2
	}

	// Question 10: Community engagement
	if responses["q10"] == "I actively participate in communities" {
		scores["collaborator"] += 3
	} else if responses["q10"] == "I often initiate or organise groups" {
		scores["strategist"] += 3
		scores["collaborator"] += 2
	} else if responses["q10"] == "I prefer working independently" {
		scores["explorer"] += 2
		scores["builder"] += 2
	}

	// Question 11: Purpose-driven association
	if responses["q11"] == "Yes" {
		scores["collaborator"] += 2
		scores["strategist"] += 2
	}

	// Question 12: Time commitment
	if responses["q12"] == "7–10 hours/week" || responses["q12"] == "Depends on the opportunity" {
		scores["builder"] += 2
		scores["explorer"] += 2
	}

	// Find the archetype with highest score
	maxScore := 0
	archetypeName := "builder" // default
	for name, score := range scores {
		if score > maxScore {
			maxScore = score
			archetypeName = name
		}
	}

	// Return archetype details
	archetypes := map[string]DNAArchetype{
		"builder": {
			Name:        "The Builder",
			Description: "You are drawn to action, ownership, and learning by doing. You perform best when working on real problems and taking responsibility for outcomes.",
			Traits: []string{
				"Action-oriented",
				"Ownership-driven",
				"Problem-solver",
				"Results-focused",
			},
		},
		"explorer": {
			Name:        "The Explorer",
			Description: "You thrive on discovery, learning, and independence. You're comfortable navigating uncertainty and building your own path.",
			Traits: []string{
				"Curious",
				"Independent",
				"Adaptable",
				"Self-directed",
			},
		},
		"strategist": {
			Name:        "The Strategist",
			Description: "You think ahead, organize effectively, and create clarity from complexity. You excel at planning and direction-setting.",
			Traits: []string{
				"Analytical",
				"Organized",
				"Forward-thinking",
				"Direction-setting",
			},
		},
		"collaborator": {
			Name:        "The Collaborator",
			Description: "You build through relationships, support others, and create value in team settings. You thrive in structured, supportive environments.",
			Traits: []string{
				"Relationship-focused",
				"Supportive",
				"Team-oriented",
				"Community-minded",
			},
		},
	}

	if archetype, ok := archetypes[archetypeName]; ok {
		return archetype
	}

	// Fallback
	return archetypes["builder"]
}

// SubmitDNASnapshot handles DNA snapshot submission
func SubmitDNASnapshot(c *fiber.Ctx, db *gorm.DB) error {
	userID := c.Locals("user_id").(uint)

	// Find student record
	var student Student
	if err := db.Where("user_id = ?", userID).First(&student).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "student record not found"})
		}
		return c.Status(500).JSON(fiber.Map{"msg": "failed to find student record"})
	}

	// Check if already completed
	if student.HasCompletedDNASnapshot {
		return c.Status(400).JSON(fiber.Map{"msg": "DNA snapshot already completed"})
	}

	var req DNASnapshotRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid request: " + err.Error()})
	}

	// Validate that we have responses
	if len(req.Responses) == 0 {
		return c.Status(400).JSON(fiber.Map{"msg": "responses are required"})
	}

	// Calculate archetype
	archetype := CalculateDNAArchetype(req.Responses)

	// Convert responses to JSON
	responsesJSON, err := json.Marshal(req.Responses)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to process responses"})
	}

	// Update student record
	now := time.Now()
	responsesJSONData := datatypes.JSON(responsesJSON)
	archetypeName := archetype.Name
	student.HasCompletedDNASnapshot = true
	student.DNASnapshotResponses = &responsesJSONData
	student.DNAArchetype = &archetypeName
	student.DNASnapshotCompletedAt = &now

	if err := db.Save(&student).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to save DNA snapshot: " + err.Error()})
	}

	return c.JSON(fiber.Map{
		"msg": "DNA snapshot completed successfully",
		"data": fiber.Map{
			"archetype": archetype,
		},
	})
}

// GetDNASnapshot retrieves the student's DNA snapshot (if completed)
func GetDNASnapshot(c *fiber.Ctx, db *gorm.DB) error {
	userID := c.Locals("user_id").(uint)

	var student Student
	if err := db.Where("user_id = ?", userID).First(&student).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "student record not found"})
		}
		return c.Status(500).JSON(fiber.Map{"msg": "failed to find student record"})
	}

	if !student.HasCompletedDNASnapshot {
		return c.Status(404).JSON(fiber.Map{"msg": "DNA snapshot not completed"})
	}

	// Reconstruct archetype from stored name
	archetypes := map[string]DNAArchetype{
		"The Builder": {
			Name:        "The Builder",
			Description: "You are drawn to action, ownership, and learning by doing. You perform best when working on real problems and taking responsibility for outcomes.",
			Traits: []string{
				"Action-oriented",
				"Ownership-driven",
				"Problem-solver",
				"Results-focused",
			},
		},
		"The Explorer": {
			Name:        "The Explorer",
			Description: "You thrive on discovery, learning, and independence. You're comfortable navigating uncertainty and building your own path.",
			Traits: []string{
				"Curious",
				"Independent",
				"Adaptable",
				"Self-directed",
			},
		},
		"The Strategist": {
			Name:        "The Strategist",
			Description: "You think ahead, organize effectively, and create clarity from complexity. You excel at planning and direction-setting.",
			Traits: []string{
				"Analytical",
				"Organized",
				"Forward-thinking",
				"Direction-setting",
			},
		},
		"The Collaborator": {
			Name:        "The Collaborator",
			Description: "You build through relationships, support others, and create value in team settings. You thrive in structured, supportive environments.",
			Traits: []string{
				"Relationship-focused",
				"Supportive",
				"Team-oriented",
				"Community-minded",
			},
		},
	}

	var archetypeName string
	if student.DNAArchetype != nil {
		archetypeName = *student.DNAArchetype
	} else {
		archetypeName = "The Builder" // fallback
	}

	archetype, ok := archetypes[archetypeName]
	if !ok {
		archetype = archetypes["The Builder"] // fallback
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"archetype":    archetype,
			"completedAt":  student.DNASnapshotCompletedAt,
			"hasCompleted": student.HasCompletedDNASnapshot,
		},
	})
}

// GetStudentDNASnapshot retrieves a student's DNA snapshot (admin access)
func GetStudentDNASnapshot(c *fiber.Ctx, db *gorm.DB) error {
	studentIdParam := c.Params("studentId")
	studentId, err := strconv.ParseUint(studentIdParam, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid student ID"})
	}

	var student Student
	if err := db.First(&student, studentId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "student not found"})
		}
		return c.Status(500).JSON(fiber.Map{"msg": "failed to find student"})
	}

	if !student.HasCompletedDNASnapshot {
		return c.JSON(fiber.Map{
			"data": fiber.Map{
				"hasCompleted": false,
				"archetype":    nil,
				"completedAt":  nil,
			},
		})
	}

	// Reconstruct archetype from stored name
	archetypes := map[string]DNAArchetype{
		"The Builder": {
			Name:        "The Builder",
			Description: "You are drawn to action, ownership, and learning by doing. You perform best when working on real problems and taking responsibility for outcomes.",
			Traits: []string{
				"Action-oriented",
				"Ownership-driven",
				"Problem-solver",
				"Results-focused",
			},
		},
		"The Explorer": {
			Name:        "The Explorer",
			Description: "You thrive on discovery, learning, and independence. You're comfortable navigating uncertainty and building your own path.",
			Traits: []string{
				"Curious",
				"Independent",
				"Adaptable",
				"Self-directed",
			},
		},
		"The Strategist": {
			Name:        "The Strategist",
			Description: "You think ahead, organize effectively, and create clarity from complexity. You excel at planning and direction-setting.",
			Traits: []string{
				"Analytical",
				"Organized",
				"Forward-thinking",
				"Direction-setting",
			},
		},
		"The Collaborator": {
			Name:        "The Collaborator",
			Description: "You build through relationships, support others, and create value in team settings. You thrive in structured, supportive environments.",
			Traits: []string{
				"Relationship-focused",
				"Supportive",
				"Team-oriented",
				"Community-minded",
			},
		},
	}

	var archetypeName string
	if student.DNAArchetype != nil {
		archetypeName = *student.DNAArchetype
	} else {
		archetypeName = "The Builder" // fallback
	}

	archetype, ok := archetypes[archetypeName]
	if !ok {
		archetype = archetypes["The Builder"] // fallback
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"archetype":    archetype,
			"completedAt":  student.DNASnapshotCompletedAt,
			"hasCompleted": student.HasCompletedDNASnapshot,
		},
	})
}
