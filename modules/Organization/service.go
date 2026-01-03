package organization

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"strconv"
	"strings"
	"time"

	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// normalizeOrganizationType converts backend type to frontend format
func normalizeOrganizationType(t string) string {
	switch strings.ToLower(t) {
	case "university":
		return "UNIVERSITY"
	case "company":
		return "PARTNER"
	default:
		return strings.ToUpper(t)
	}
}

// normalizeKYCStatus converts is_approved to frontend kycStatus format
func normalizeKYCStatus(isApproved bool) string {
	if isApproved {
		return "APPROVED"
	}
	return "PENDING"
}

// GenerateRandomPassword generates a secure random password
func GenerateRandomPassword(length int) (string, error) {
	if length < 8 {
		length = 12 // Default to 12 characters
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Use base64 encoding for a readable password
	password := base64.URLEncoding.EncodeToString(bytes)
	// Take only the first 'length' characters
	if len(password) > length {
		password = password[:length]
	}

	return password, nil
}

// transformOrganizationForResponse transforms Organization to frontend format
func transformOrganizationForResponse(org Organization) map[string]interface{} {
	email := ""
	if org.User.ID != 0 {
		email = org.User.Email
	}

	return map[string]interface{}{
		"id":         org.ID,
		"type":       normalizeOrganizationType(org.Type),
		"name":       org.Name,
		"email":      email,
		"kycStatus":  normalizeKYCStatus(org.IsApproved),
		"logo":       org.Logo,
		"website":    org.Website,
		"brandColor": org.BrandColor,
		"address":    org.Address,
		"createdAt":  org.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"updatedAt":  org.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// organizationCreateRequest represents the request payload for organization creation
type organizationCreateRequest struct {
	Name           string                 `json:"name"`
	Type           string                 `json:"type"`
	Address        string                 `json:"address"`
	Website        string                 `json:"website"`
	BrandColor     string                 `json:"brandColor"`
	Description    string                 `json:"description"`
	BillingProfile map[string]interface{} `json:"billingProfile"`
}

func Register(c *fiber.Ctx, db *gorm.DB) error {
	role, _ := c.Locals("role").(string)
	isSuperAdmin := role == "super-admin"

	var org Organization
	var req organizationCreateRequest

	// Parse the request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "something is wrong with the submitted data : " + err.Error()})
	}

	// Map request to organization
	org.Name = strings.TrimSpace(req.Name)
	org.Type = strings.ToLower(strings.TrimSpace(req.Type))
	org.Address = strings.TrimSpace(req.Address)
	org.Website = strings.TrimSpace(req.Website)
	org.BrandColor = strings.TrimSpace(req.BrandColor)

	if org.Type == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "please select the type as either university or company"})
	}

	// Handle super-admin organization creation
	if isSuperAdmin {
		// Extract owner information from billingProfile
		var ownerEmail, ownerName, ownerPhone string
		if req.BillingProfile != nil {
			if email, ok := req.BillingProfile["email"].(string); ok && email != "" {
				ownerEmail = strings.TrimSpace(email)
			}
			if contactName, ok := req.BillingProfile["contactName"].(string); ok && contactName != "" {
				ownerName = strings.TrimSpace(contactName)
			}
			if phone, ok := req.BillingProfile["phone"].(string); ok && phone != "" {
				ownerPhone = strings.TrimSpace(phone)
			}
		}

		// Validate that email is provided for super-admin creation
		if ownerEmail == "" {
			return c.Status(400).JSON(fiber.Map{"msg": "email is required when creating organization as super-admin"})
		}

		// If email is provided, create a user account for the owner
		if ownerEmail != "" {
			// Check if user already exists
			var existingUser user.User
			if err := db.Where("email = ?", ownerEmail).First(&existingUser).Error; err == nil {
				return c.Status(400).JSON(fiber.Map{"msg": "user with email " + ownerEmail + " already exists"})
			}

			// Generate random password
			randomPassword, err := GenerateRandomPassword(12)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"msg": "failed to generate password"})
			}

			// Determine role based on organization type
			ownerRole := "partner"
			if org.Type == "university" {
				ownerRole = "university-admin"
			}

			// Create user account
			newUser := user.User{
				Email:    ownerEmail,
				Name:     ownerName,
				Role:     ownerRole,
				Password: user.GenerateHash(randomPassword),
			}
			if ownerPhone != "" {
				newUser.Profile.Phone = ownerPhone
			}

			if err := db.Create(&newUser).Error; err != nil {
				return c.Status(400).JSON(fiber.Map{"msg": "failed to create user account: " + err.Error()})
			}

			org.UserID = newUser.ID
			org.IsApproved = true // Pre-approve organizations created by super-admin

			// Create organization
			if err := db.Create(&org).Error; err != nil {
				// Rollback: delete user if organization creation fails
				db.Delete(&newUser)
				return c.Status(400).JSON(fiber.Map{"msg": "failed to add organization: " + err.Error()})
			}

			// Reload organization with user relation
			db.Preload("User").First(&org, org.ID)

			// Send welcome email with credentials
			if err := SendOrganizationCreationEmail(org, ownerEmail, ownerName, randomPassword); err != nil {
				log.Printf("failed to send creation email to %s: %v", ownerEmail, err)
				// Don't fail the request if email fails
			}

			return c.Status(201).JSON(fiber.Map{
				"msg":  org.Type + " created successfully",
				"data": transformOrganizationForResponse(org),
			})
		} else {
			// Super-admin creating org without owner email - use current user
			id := c.Locals("user_id")
			org.UserID = id.(uint)
			org.IsApproved = true
		}
	} else {
		// Regular user registration
		id := c.Locals("user_id")
		org.UserID = id.(uint)
		org.IsApproved = false
	}

	// Create organization for non-super-admin or super-admin without owner email
	if err := db.Create(&org).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to add organization: " + err.Error()})
	}

	// Reload with relations
	db.Preload("User").First(&org, org.ID)

	return c.Status(201).JSON(fiber.Map{
		"msg":  org.Type + " created successfully",
		"data": transformOrganizationForResponse(org),
	})
}

func GetByType(c *fiber.Ctx, db *gorm.DB, t string) error {
	var orgs []Organization

	// Normalize type to lowercase for case-insensitive matching
	// Frontend may send "UNIVERSITY" but database stores "university"
	typeLower := strings.ToLower(t)

	query := db.Where("LOWER(type) = ?", typeLower)

	// For partners, only show approved organizations
	if role, ok := c.Locals("role").(string); ok && role == "partner" {
		query = query.Where("is_approved = ?", true)
	}

	if err := query.Preload("User").Find(&orgs).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to fetch organizations: " + err.Error()})
	}

	// Transform organizations to frontend format
	transformedOrgs := make([]map[string]interface{}, len(orgs))
	for i, org := range orgs {
		transformedOrgs[i] = transformOrganizationForResponse(org)
	}

	return c.JSON(fiber.Map{"data": transformedOrgs})
}

// GetAll retrieves all organizations
func GetAll(c *fiber.Ctx, db *gorm.DB) error {
	var orgs []Organization
	query := db.Model(&Organization{})

	// For partners, only show approved organizations
	if role, ok := c.Locals("role").(string); ok && role == "partner" {
		query = query.Where("is_approved = ?", true)
	}

	if err := query.Preload("User").Find(&orgs).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get organizations: " + err.Error()})
	}

	// Transform organizations to frontend format
	transformedOrgs := make([]map[string]interface{}, len(orgs))
	for i, org := range orgs {
		transformedOrgs[i] = transformOrganizationForResponse(org)
	}

	return c.JSON(fiber.Map{"data": transformedOrgs})
}

// GetByID retrieves an organization by ID
func GetByID(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	var org Organization

	if err := db.Preload("User").First(&org, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "organization not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get organization: " + err.Error()})
	}

	return c.JSON(fiber.Map{"data": transformOrganizationForResponse(org)})
}

// Update updates an organization
type organizationUpdateInput struct {
	Name       *string `json:"name"`
	Type       *string `json:"type"`
	Address    *string `json:"address"`
	Website    *string `json:"website"`
	Logo       *string `json:"logo"`
	BrandColor *string `json:"brandColor"`
	KYCStatus  *string `json:"kycStatus"`
}

func Update(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	var org Organization
	if err := db.First(&org, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "organization not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to find organization"})
	}

	// Check if user owns the organization or is super-admin
	if org.UserID != userID && role != "super-admin" {
		return c.Status(403).JSON(fiber.Map{"msg": "you don't have permission to update this organization"})
	}

	var input organizationUpdateInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid update data: " + err.Error()})
	}

	prevApproved := org.IsApproved
	fieldsChanged := false

	if input.Name != nil {
		org.Name = strings.TrimSpace(*input.Name)
		fieldsChanged = true
	}
	if input.Type != nil {
		org.Type = strings.ToLower(strings.TrimSpace(*input.Type))
		fieldsChanged = true
	}
	if input.Address != nil {
		org.Address = strings.TrimSpace(*input.Address)
		fieldsChanged = true
	}
	if input.Website != nil {
		org.Website = strings.TrimSpace(*input.Website)
		fieldsChanged = true
	}
	if input.Logo != nil {
		org.Logo = strings.TrimSpace(*input.Logo)
		fieldsChanged = true
	}
	if input.BrandColor != nil {
		org.BrandColor = strings.TrimSpace(*input.BrandColor)
		fieldsChanged = true
	}
	statusChanged := false
	if input.KYCStatus != nil {
		next := strings.ToUpper(strings.TrimSpace(*input.KYCStatus))
		switch next {
		case "APPROVED":
			org.IsApproved = true
		case "REJECTED":
			org.IsApproved = false
		default:
			org.IsApproved = false
		}
		statusChanged = org.IsApproved != prevApproved
		fieldsChanged = true
	}

	if err := db.Save(&org).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to update organization: " + err.Error()})
	}

	// Reload with relations
	db.Preload("User").First(&org, org.ID)

	ownerEmail := org.User.Email
	if ownerEmail != "" {
		if statusChanged {
			if err := SendOrganizationStatusEmail(org, ownerEmail, prevApproved); err != nil {
				log.Printf("failed to send status email to %s: %v", ownerEmail, err)
			}
		} else if fieldsChanged {
			if err := SendOrganizationUpdateEmail(org, ownerEmail); err != nil {
				log.Printf("failed to send update email to %s: %v", ownerEmail, err)
			}
		}
	}

	return c.JSON(fiber.Map{
		"msg":  "organization updated successfully",
		"data": transformOrganizationForResponse(org),
	})
}

type departmentStatsResponse struct {
	DepartmentID      uint   `json:"departmentId"`
	DepartmentName    string `json:"departmentName"`
	ActiveProjects    int64  `json:"activeProjects"`
	CompletedProjects int64  `json:"completedProjects"`
	PendingProjects   int64  `json:"pendingProjects"`
}

type recentProjectResponse struct {
	ID             uint   `json:"id"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	DepartmentName string `json:"departmentName"`
}

type dashboardStatsResponse struct {
	TotalStudents   int64                     `json:"totalStudents"`
	ActiveProjects  int64                     `json:"activeProjects"`
	PendingReviews  int64                     `json:"pendingReviews"`
	DepartmentStats []departmentStatsResponse `json:"departmentStats"`
	RecentProjects  []recentProjectResponse   `json:"recentProjects"`
	StudentTrend    []studentTrendPoint       `json:"studentTrend"`
}

type partnerDashboardStatsResponse struct {
	TotalProjects     int64   `json:"totalProjects"`
	ActiveProjects    int64   `json:"activeProjects"`
	CompletedProjects int64   `json:"completedProjects"`
	TotalBudget       float64 `json:"totalBudget"`
}

type studentTrendPoint struct {
	Month string `json:"month"`
	Count int64  `json:"count"`
}

func GetDashboardStats(c *fiber.Ctx, db *gorm.DB) error {
	idParam := c.Params("id")
	orgID, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"msg": "invalid organization id"})
	}

	var org Organization
	if err := db.First(&org, orgID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"msg": "organization not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"msg": "failed to load organization"})
	}

	stats, err := buildDashboardStats(db, uint(orgID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"msg": "failed to load dashboard stats"})
	}

	return c.JSON(fiber.Map{"data": stats})
}

func buildDashboardStats(db *gorm.DB, orgID uint) (dashboardStatsResponse, error) {
	stats := dashboardStatsResponse{}

	studentQuery := db.Table("students").
		Joins("JOIN courses ON students.course_id = courses.id").
		Joins("JOIN departments ON courses.department_id = departments.id").
		Where("departments.organization_id = ?", orgID)
	if err := studentQuery.Count(&stats.TotalStudents).Error; err != nil {
		return stats, err
	}

	activeStatuses := []string{"in-progress", "pending", "draft"}
	projectBaseQuery := db.Table("projects").
		Joins("JOIN departments ON projects.department_id = departments.id").
		Where("departments.organization_id = ?", orgID)
	if err := projectBaseQuery.
		Where("projects.status IN ?", activeStatuses).
		Count(&stats.ActiveProjects).Error; err != nil {
		return stats, err
	}

	reviewStatuses := []string{"SUPERVISOR_REVIEW", "PARTNER_REVIEW"}
	reviewQuery := db.Table("milestones").
		Joins("JOIN projects ON milestones.project_id = projects.id").
		Joins("JOIN departments ON projects.department_id = departments.id").
		Where("departments.organization_id = ?", orgID).
		Where("milestones.status IN ?", reviewStatuses)
	if err := reviewQuery.Count(&stats.PendingReviews).Error; err != nil {
		return stats, err
	}

	var departments []struct {
		ID   uint
		Name string
	}
	if err := db.Table("departments").Where("organization_id = ?", orgID).Scan(&departments).Error; err != nil {
		return stats, err
	}

	deptMap := make(map[uint]*departmentStatsResponse, len(departments))
	for _, dept := range departments {
		deptMap[dept.ID] = &departmentStatsResponse{
			DepartmentID:   dept.ID,
			DepartmentName: dept.Name,
		}
	}

	var projectRows []struct {
		DepartmentID   uint
		DepartmentName string
		Status         string
	}
	if err := db.Table("projects").
		Select("departments.id as department_id, departments.name as department_name, projects.status").
		Joins("JOIN departments ON projects.department_id = departments.id").
		Where("departments.organization_id = ?", orgID).
		Scan(&projectRows).Error; err != nil {
		return stats, err
	}

	for _, row := range projectRows {
		entry, ok := deptMap[row.DepartmentID]
		if !ok {
			entry = &departmentStatsResponse{
				DepartmentID:   row.DepartmentID,
				DepartmentName: row.DepartmentName,
			}
			deptMap[row.DepartmentID] = entry
		}

		switch strings.ToLower(row.Status) {
		case "completed":
			entry.CompletedProjects++
		case "in-progress", "published":
			entry.ActiveProjects++
		case "pending", "draft":
			entry.PendingProjects++
		default:
			entry.PendingProjects++
		}
	}

	stats.DepartmentStats = make([]departmentStatsResponse, 0, len(deptMap))
	for _, entry := range deptMap {
		stats.DepartmentStats = append(stats.DepartmentStats, *entry)
	}

	var recentProjects []recentProjectResponse
	if err := db.Table("projects").
		Select("projects.id, projects.title, projects.status, departments.name as department_name").
		Joins("JOIN departments ON projects.department_id = departments.id").
		Where("departments.organization_id = ?", orgID).
		Order("projects.updated_at DESC").
		Limit(5).
		Scan(&recentProjects).Error; err != nil {
		return stats, err
	}
	stats.RecentProjects = recentProjects

	trend, err := fetchStudentTrend(db, orgID)
	if err != nil {
		return stats, err
	}
	stats.StudentTrend = trend

	return stats, nil
}

func fetchStudentTrend(db *gorm.DB, orgID uint) ([]studentTrendPoint, error) {
	threshold := time.Now().AddDate(0, -6, 0)
	var rows []struct {
		Month time.Time
		Count int64
	}

	err := db.Table("students").
		Select("DATE_TRUNC('month', students.created_at) AS month, COUNT(*) AS count").
		Joins("JOIN courses ON students.course_id = courses.id").
		Joins("JOIN departments ON courses.department_id = departments.id").
		Where("departments.organization_id = ?", orgID).
		Where("students.created_at >= ?", threshold).
		Group("month").
		Order("month ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	trend := make([]studentTrendPoint, len(rows))
	for i, row := range rows {
		trend[i] = studentTrendPoint{
			Month: row.Month.Format("Jan 2006"),
			Count: row.Count,
		}
	}

	return trend, nil
}

// Delete deletes an organization
func Delete(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	var org Organization
	if err := db.First(&org, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "organization not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to find organization"})
	}

	// Check if user owns the organization or is super-admin
	if org.UserID != userID && role != "super-admin" {
		return c.Status(403).JSON(fiber.Map{"msg": "you don't have permission to delete this organization"})
	}

	// Store the user ID before deleting the organization
	associatedUserID := org.UserID

	if err := db.Delete(&org).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to delete organization: " + err.Error()})
	}

	// Delete the associated user (owner of the organization)
	var associatedUser user.User
	if err := db.First(&associatedUser, associatedUserID).Error; err == nil {
		if err := db.Delete(&associatedUser).Error; err != nil {
			// Log the error but don't fail the request since org is already deleted
			log.Printf("Warning: failed to delete associated user %d: %v", associatedUserID, err)
		}
	}

	return c.JSON(fiber.Map{"msg": "organization deleted successfully"})
}

// GetNestedWithDepartmentsAndCourses retrieves all organizations (filtered by type if provided) with nested departments and courses
// This endpoint is optimized for populating select forms in the frontend
func GetNestedWithDepartmentsAndCourses(c *fiber.Ctx, db *gorm.DB) error {
	var orgs []Organization

	// Build query - filter by type if provided (e.g., "university")
	typeParam := c.Query("type")

	// Use same pattern as GetByType which works correctly
	// Start with Model() to ensure we're querying the correct table
	query := db.Model(&Organization{})

	if typeParam != "" {
		typeLower := strings.ToLower(typeParam)
		query = query.Where("LOWER(type) = ?", typeLower)
	}

	// For partners, only show approved organizations
	if role, ok := c.Locals("role").(string); ok && role == "partner" {
		query = query.Where("is_approved = ?", true)
	}

	// Fetch organizations with User preloaded
	if err := query.Preload("User").Find(&orgs).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to fetch organizations: " + err.Error()})
	}

	// Transform to nested response format
	type CourseResponse struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}

	type DepartmentResponse struct {
		ID      uint             `json:"id"`
		Name    string           `json:"name"`
		Courses []CourseResponse `json:"courses"`
	}

	type OrganizationResponse struct {
		ID          uint                 `json:"id"`
		Name        string               `json:"name"`
		Type        string               `json:"type"`
		Departments []DepartmentResponse `json:"departments"`
	}

	// Local structs to avoid import cycles (matching Department and Course table structure)
	type DepartmentRow struct {
		ID             uint   `gorm:"column:id"`
		Name           string `gorm:"column:name"`
		OrganizationID uint   `gorm:"column:organization_id"`
	}

	type CourseRow struct {
		ID           uint   `gorm:"column:id"`
		Name         string `gorm:"column:name"`
		DepartmentID uint   `gorm:"column:department_id"`
	}

	response := make([]OrganizationResponse, len(orgs))
	for i, org := range orgs {
		// Get departments for this organization (query directly from departments table)
		var depts []DepartmentRow
		if err := db.Table("departments").Where("organization_id = ?", org.ID).Find(&depts).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to fetch departments: " + err.Error()})
		}

		deptResponses := make([]DepartmentResponse, len(depts))
		for j, dept := range depts {
			// Get courses for this department (query directly from courses table)
			var courses []CourseRow
			if err := db.Table("courses").Where("department_id = ?", dept.ID).Find(&courses).Error; err != nil {
				return c.Status(400).JSON(fiber.Map{"msg": "failed to fetch courses: " + err.Error()})
			}

			courseResponses := make([]CourseResponse, len(courses))
			for k, course := range courses {
				courseResponses[k] = CourseResponse{
					ID:   course.ID,
					Name: course.Name,
				}
			}

			deptResponses[j] = DepartmentResponse{
				ID:      dept.ID,
				Name:    dept.Name,
				Courses: courseResponses,
			}
		}

		response[i] = OrganizationResponse{
			ID:          org.ID,
			Name:        org.Name,
			Type:        normalizeOrganizationType(org.Type),
			Departments: deptResponses,
		}
	}

	return c.JSON(fiber.Map{"data": response})
}

// GetPartnerDashboardStats returns dashboard statistics for a partner
func GetPartnerDashboardStats(c *fiber.Ctx, db *gorm.DB) error {
	userID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	// Only partners can access this endpoint
	if role != "partner" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"msg": "access denied"})
	}

	stats, err := buildPartnerDashboardStats(db, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"msg": "failed to load dashboard stats: " + err.Error()})
	}

	return c.JSON(fiber.Map{"data": stats})
}

func buildPartnerDashboardStats(db *gorm.DB, partnerID uint) (partnerDashboardStatsResponse, error) {
	stats := partnerDashboardStatsResponse{}

	// Count total projects
	if err := db.Table("projects").Where("user_id = ?", partnerID).Count(&stats.TotalProjects).Error; err != nil {
		return stats, err
	}

	// Count active projects
	if err := db.Table("projects").
		Where("user_id = ? AND status = ?", partnerID, "in-progress").
		Count(&stats.ActiveProjects).Error; err != nil {
		return stats, err
	}

	// Count completed projects
	if err := db.Table("projects").
		Where("user_id = ? AND status = ?", partnerID, "completed").
		Count(&stats.CompletedProjects).Error; err != nil {
		return stats, err
	}

	// Calculate total budget
	var budgetRows []struct {
		BudgetValue uint `gorm:"column:budget_value"`
	}
	if err := db.Table("projects").
		Select("budget_value").
		Where("user_id = ?", partnerID).
		Scan(&budgetRows).Error; err != nil {
		return stats, err
	}

	for _, row := range budgetRows {
		stats.TotalBudget += float64(row.BudgetValue)
	}

	return stats, nil
}
