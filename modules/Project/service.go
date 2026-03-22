package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	course "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Course"
	department "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Department"
	organization "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Organization"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Valid team structure values
const (
	TeamStructureIndividuals = "individuals"
	TeamStructureGroups      = "groups"
	TeamStructureBoth        = "both"
)

// ValidTeamStructures is a map of valid team structure values
var ValidTeamStructures = map[string]bool{
	TeamStructureIndividuals: true,
	TeamStructureGroups:      true,
	TeamStructureBoth:        true,
}

// ValidateTeamStructure validates that the team structure value is one of the accepted values
func ValidateTeamStructure(teamStructure string) bool {
	if teamStructure == "" {
		return false
	}
	return ValidTeamStructures[teamStructure]
}

func Create(c *fiber.Ctx, db *gorm.DB) error {
	// Parse request body - frontend sends camelCase field names
	type CreateRequest struct {
		DepartmentID           *int     `json:"departmentId,omitempty"`
		CourseID               *int     `json:"courseId,omitempty"`
		UniversityID           *uint    `json:"universityId,omitempty"`
		Title                  string   `json:"title"`
		Description            string   `json:"description"` // Kept for backward compatibility
		Summary                string   `json:"summary,omitempty"`
		ChallengeStatement     string   `json:"challengeStatement,omitempty"`
		ScopeActivities        string   `json:"scopeActivities,omitempty"`
		DeliverablesMilestones string   `json:"deliverablesMilestones,omitempty"`
		TeamStructure          string   `json:"teamStructure,omitempty"`
		Duration               string   `json:"duration,omitempty"`
		Expectations           string   `json:"expectations,omitempty"`
		Skills                 []string `json:"skills"`
		BudgetValue            *float64 `json:"budget,omitempty"`   // Frontend sends budget as number
		Currency               *string  `json:"currency,omitempty"` // Frontend sends currency separately
		Deadline               string   `json:"deadline"`
		Capacity               uint     `json:"capacity"`
		Status                 string   `json:"status"`
		Attachments            []string `json:"attachments"`
		PartnerSignature       string   `json:"partnerSignature,omitempty"` // Partner signature data URL
	}

	var req CreateRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get project details: " + err.Error()})
	}

	// Validate team structure if provided
	if req.TeamStructure != "" && !ValidateTeamStructure(req.TeamStructure) {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid team structure. Must be one of: individuals, groups, both"})
	}

	// Build project from request
	project := Project{
		Title:                  req.Title,
		Description:            req.Description,
		Summary:                req.Summary,
		ChallengeStatement:     req.ChallengeStatement,
		ScopeActivities:        req.ScopeActivities,
		DeliverablesMilestones: req.DeliverablesMilestones,
		TeamStructure:          req.TeamStructure,
		Duration:               req.Duration,
		Expectations:           req.Expectations,
		Deadline:               req.Deadline,
		Capacity:               req.Capacity,
		Status:                 req.Status,
		UserID:                 c.Locals("user_id").(uint),
		PartnerSignature:       req.PartnerSignature,
		// SupervisorID is optional - will be set to 0 by default, but we'll omit it from insert
	}

	// Handle departmentId
	if req.DepartmentID != nil {
		project.DepartmentID = *req.DepartmentID
	}

	// Handle courseId (optional)
	if req.CourseID != nil && *req.CourseID > 0 {
		courseID := uint(*req.CourseID)
		project.CourseID = &courseID
	}

	// Handle skills - convert to JSON
	if len(req.Skills) > 0 {
		skillsJSON, err := json.Marshal(req.Skills)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to process skills: " + err.Error()})
		}
		project.Skills = datatypes.JSON(skillsJSON)
	}

	// Handle attachments - convert to JSON
	if len(req.Attachments) > 0 {
		attachmentsJSON, err := json.Marshal(req.Attachments)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to process attachments: " + err.Error()})
		}
		project.Attachments = datatypes.JSON(attachmentsJSON)
	}

	// Handle budget - frontend sends budget as number and currency separately
	if req.BudgetValue != nil && req.Currency != nil {
		project.Budget = Budget{
			Currency: *req.Currency,
			Value:    uint(*req.BudgetValue),
		}
	}

	// Validate budget is set
	if project.Budget.Currency == "" || project.Budget.Value == 0 {
		return c.Status(400).JSON(fiber.Map{"msg": "budget and currency are required"})
	}

	// Validate department exists
	if project.DepartmentID == 0 {
		return c.Status(400).JSON(fiber.Map{"msg": "department_id is required"})
	}

	var dept department.Department
	if err := db.Preload("Organization").First(&dept, project.DepartmentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "department not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to validate department: " + err.Error()})
	}

	// If universityId is provided, validate that department belongs to that university
	if req.UniversityID != nil && *req.UniversityID != 0 {
		if dept.OrganizationID != *req.UniversityID {
			return c.Status(400).JSON(fiber.Map{"msg": "department does not belong to the specified university"})
		}
	}

	// Validate that the department's organization is a university
	if dept.Organization.Type != "university" && dept.Organization.Type != "UNIVERSITY" {
		return c.Status(400).JSON(fiber.Map{"msg": "department must belong to a university organization"})
	}

	// Validate course if courseId is provided
	if project.CourseID != nil && *project.CourseID > 0 {
		var courseRecord course.Course
		if err := db.First(&courseRecord, *project.CourseID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(404).JSON(fiber.Map{"msg": "course not found"})
			}
			return c.Status(400).JSON(fiber.Map{"msg": "failed to validate course: " + err.Error()})
		}
		// Validate that course belongs to the same department
		if courseRecord.DepartmentID != uint(project.DepartmentID) {
			return c.Status(400).JSON(fiber.Map{"msg": "course does not belong to the specified department"})
		}
	}

	// Create project - SupervisorID is now nullable, so nil is fine
	if err := db.Create(&project).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to add project: " + err.Error()})
	}

	// Preload all relations for consistent response format
	if err := db.Preload("User").Preload("Supervisor").Preload("Department").Preload("Department.Organization").Preload("Course").First(&project, project.ID).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "project created but failed to load details: " + err.Error()})
	}

	data := fiber.Map{
		"msg":  "project created successfully",
		"data": project,
	}

	return c.Status(201).JSON(data)
}

func GetByOwner(c *fiber.Ctx, db *gorm.DB) error {

	var projects []Project

	if err := db.Where("user_id = ?", c.Locals("user_id").(uint)).Preload("Department").Preload("Department.Organization").Preload("Course").Preload("User").Preload("Supervisor").Find(&projects).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get projects: " + err.Error()})
	}

	return c.JSON(fiber.Map{"data": projects})

}

func Update(c *fiber.Ctx, db *gorm.DB) error {
	// Parse request body - frontend sends camelCase field names
	type UpdateRequest struct {
		ID                     *uint    `json:"id"`
		DepartmentID           *int     `json:"departmentId,omitempty"`
		CourseID               *int     `json:"courseId,omitempty"`
		UniversityID           *uint    `json:"universityId,omitempty"`
		Title                  *string  `json:"title,omitempty"`
		Description            *string  `json:"description,omitempty"`
		Summary                *string  `json:"summary,omitempty"`
		ChallengeStatement     *string  `json:"challengeStatement,omitempty"`
		ScopeActivities        *string  `json:"scopeActivities,omitempty"`
		DeliverablesMilestones *string  `json:"deliverablesMilestones,omitempty"`
		TeamStructure          *string  `json:"teamStructure,omitempty"`
		Duration               *string  `json:"duration,omitempty"`
		Expectations           *string  `json:"expectations,omitempty"`
		Skills                 []string `json:"skills,omitempty"`
		BudgetValue            *float64 `json:"budget,omitempty"`
		Currency               *string  `json:"currency,omitempty"`
		Deadline               *string  `json:"deadline,omitempty"`
		Capacity               *uint    `json:"capacity,omitempty"`
		Status                 *string  `json:"status,omitempty"`
		Attachments            []string `json:"attachments,omitempty"`
	}

	var req UpdateRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get project details: " + err.Error()})
	}

	// Validate project ID is provided
	if req.ID == nil || *req.ID == 0 {
		return c.Status(400).JSON(fiber.Map{"msg": "project id is required"})
	}

	// Load existing project
	var project Project
	if err := db.First(&project, *req.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "project not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to load project: " + err.Error()})
	}

	// Authorization check - only project owner or super-admin can update
	userID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	if role != "super-admin" && project.UserID != userID {
		return c.Status(403).JSON(fiber.Map{"msg": "you don't have permission to update this project"})
	}

	// Update fields if provided
	if req.Title != nil {
		project.Title = *req.Title
	}
	if req.Description != nil {
		project.Description = *req.Description
	}
	if req.Summary != nil {
		project.Summary = *req.Summary
	}
	if req.ChallengeStatement != nil {
		project.ChallengeStatement = *req.ChallengeStatement
	}
	if req.ScopeActivities != nil {
		project.ScopeActivities = *req.ScopeActivities
	}
	if req.DeliverablesMilestones != nil {
		project.DeliverablesMilestones = *req.DeliverablesMilestones
	}
	if req.TeamStructure != nil {
		// Validate team structure
		if !ValidateTeamStructure(*req.TeamStructure) {
			return c.Status(400).JSON(fiber.Map{"msg": "invalid team structure. Must be one of: individuals, groups, both"})
		}
		project.TeamStructure = *req.TeamStructure
	}
	if req.Duration != nil {
		project.Duration = *req.Duration
	}
	if req.Expectations != nil {
		project.Expectations = *req.Expectations
	}
	if req.Deadline != nil {
		project.Deadline = *req.Deadline
	}
	if req.Capacity != nil {
		project.Capacity = *req.Capacity
	}
	if req.Status != nil {
		project.Status = *req.Status
	}

	// Handle departmentId
	if req.DepartmentID != nil {
		project.DepartmentID = *req.DepartmentID
		// Validate department exists
		var dept department.Department
		if err := db.Preload("Organization").First(&dept, project.DepartmentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(404).JSON(fiber.Map{"msg": "department not found"})
			}
			return c.Status(400).JSON(fiber.Map{"msg": "failed to validate department: " + err.Error()})
		}
		// Validate that the department's organization is a university
		if dept.Organization.Type != "university" && dept.Organization.Type != "UNIVERSITY" {
			return c.Status(400).JSON(fiber.Map{"msg": "department must belong to a university organization"})
		}
	}

	// Handle courseId (optional)
	if req.CourseID != nil {
		if *req.CourseID > 0 {
			courseID := uint(*req.CourseID)
			project.CourseID = &courseID
			// Validate course exists and belongs to the department
			if project.DepartmentID > 0 {
				var courseRecord course.Course
				if err := db.First(&courseRecord, *project.CourseID).Error; err != nil {
					if err == gorm.ErrRecordNotFound {
						return c.Status(404).JSON(fiber.Map{"msg": "course not found"})
					}
					return c.Status(400).JSON(fiber.Map{"msg": "failed to validate course: " + err.Error()})
				}
				// Validate that course belongs to the same department
				if courseRecord.DepartmentID != uint(project.DepartmentID) {
					return c.Status(400).JSON(fiber.Map{"msg": "course does not belong to the specified department"})
				}
			}
		} else {
			// courseId is 0 or negative, set to nil
			project.CourseID = nil
		}
	}

	// Handle skills - convert to JSON
	if req.Skills != nil {
		skillsJSON, err := json.Marshal(req.Skills)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to process skills: " + err.Error()})
		}
		project.Skills = datatypes.JSON(skillsJSON)
	}

	// Handle attachments - convert to JSON
	if req.Attachments != nil {
		attachmentsJSON, err := json.Marshal(req.Attachments)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to process attachments: " + err.Error()})
		}
		project.Attachments = datatypes.JSON(attachmentsJSON)
	}

	// Handle budget - frontend sends budget as number and currency separately
	if req.BudgetValue != nil && req.Currency != nil {
		project.Budget = Budget{
			Currency: *req.Currency,
			Value:    uint(*req.BudgetValue),
		}
	} else if req.BudgetValue != nil {
		// Only budget value provided, keep existing currency
		project.Budget.Value = uint(*req.BudgetValue)
	} else if req.Currency != nil {
		// Only currency provided, keep existing value
		project.Budget.Currency = *req.Currency
	}

	// Validate budget is set (if updating budget)
	if req.BudgetValue != nil || req.Currency != nil {
		if project.Budget.Currency == "" || project.Budget.Value == 0 {
			return c.Status(400).JSON(fiber.Map{"msg": "budget and currency are required"})
		}
	}

	// Update project
	if err := db.Save(&project).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to update project: " + err.Error()})
	}

	// Preload relations for response
	if err := db.Preload("Department").Preload("Department.Organization").Preload("Course").Preload("User").Preload("Supervisor").First(&project, project.ID).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "project updated but failed to load details: " + err.Error()})
	}

	return c.JSON(fiber.Map{"data": project})
}

func UpdateStatus(c *fiber.Ctx, db *gorm.DB) error {
	status := c.Query("status")
	project := c.Query("project")
	userID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	// Parse request body for signature and MOU URL (if provided)
	type UpdateStatusRequest struct {
		UniversityAdminSignature string `json:"universityAdminSignature,omitempty"`
		MOUURL                   string `json:"mouUrl,omitempty"`
	}
	var req UpdateStatusRequest
	c.BodyParser(&req) // Ignore error if body is empty

	ProjectID, _ := strconv.ParseUint(project, 10, 64)

	var tmp Project

	// Load project with department and organization for authorization check
	if err := db.Preload("Department").Preload("Department.Organization").First(&tmp, ProjectID).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid project " + err.Error()})
	}

	// Authorization check
	// Super-admins can update any project
	// Partners can only update their own projects
	// University admins can only update projects in their university
	if role != "super-admin" {
		if role == "partner" {
			// Partner can only update their own projects
			if tmp.UserID != userID {
				return c.Status(403).JSON(fiber.Map{"msg": "you don't have permission to update this project status"})
			}
		} else if role == "university-admin" || role == "delegated-admin" {
			// University admin and delegated admin can only update projects in their university
			// Get user's organization ID
			var userOrgID uint
			var err error
			if role == "delegated-admin" {
				err = db.Table("delegated_accesses").
					Where("delegated_user_id = ? AND is_active = ?", userID, true).
					Select("organization_id").
					Scan(&userOrgID).Error
			} else {
				err = db.Table("organizations").
					Where("user_id = ?", userID).
					Select("id").
					Scan(&userOrgID).Error
			}
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"msg": "failed to get user organization"})
			}
			// Check if project's department belongs to user's organization
			if tmp.Department.OrganizationID != userOrgID {
				return c.Status(403).JSON(fiber.Map{"msg": "you don't have permission to update this project status"})
			}
		} else {
			// Other roles (student, supervisor) cannot update project status
			return c.Status(403).JSON(fiber.Map{"msg": "you don't have permission to update project status"})
		}
	}

	// Validate status value
	validStatuses := map[string]bool{
		"draft": true, "published": true, "in-progress": true,
		"on-hold": true, "completed": true, "cancelled": true,
		"pending": true, "suspended": true,
	}
	if !validStatuses[status] {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid status. Must be one of: draft, published, in-progress, on-hold, completed, cancelled, pending, suspended"})
	}

	// If approving (status = "published") and university-admin, require signature
	if status == "published" && (role == "university-admin" || role == "delegated-admin") && req.UniversityAdminSignature == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "signature is required to approve a project"})
	}

	// Update project status, signature, and MOU URL if provided
	updates := map[string]interface{}{
		"status": status,
	}
	if req.UniversityAdminSignature != "" {
		updates["university_admin_signature"] = req.UniversityAdminSignature
	}
	if req.MOUURL != "" {
		updates["mou_url"] = req.MOUURL
	}

	// If deapproving (status = "draft"), delete MOU from Cloudinary and clear signatures
	if status == "draft" {
		// Delete MOU from Cloudinary if it exists
		if tmp.MOUURL != "" {
			if err := DeleteMOUFromCloudinary(tmp.MOUURL); err != nil {
				// Log error but don't fail the status update
				fmt.Printf("Warning: Failed to delete MOU from Cloudinary: %v\n", err)
			}
		}
		// Clear MOU URL and signatures in database
		updates["mou_url"] = ""
		updates["university_admin_signature"] = ""
	}

	if err := db.Model(&tmp).Updates(updates).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to update project status"})
	}

	// Reload project with relations for response
	if err := db.Preload("Department").Preload("Department.Organization").Preload("Course").Preload("User").Preload("Supervisor").First(&tmp, tmp.ID).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "project status updated but failed to load details"})
	}

	return c.JSON(fiber.Map{"data": tmp})
}

func AssignSupervisor(c *fiber.Ctx, db *gorm.DB) error {

	type Body struct {
		UserID    uint `json:"userId"`
		ProjectID uint `json:"projectId"`
	}

	var body Body
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid input"})
	}

	var proj Project
	if err := db.First(&proj, body.ProjectID).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "project not found" + err.Error()})
	}

	var usr user.User
	if err := db.First(&usr, body.UserID).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "supervisor not found"})
	}

	if err := db.Model(&proj).Update("supervisor_id", body.UserID).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to assign supervisor: " + err.Error()})
	}

	db.Preload("Supervisor").First(&proj, proj.ID)

	return c.Status(202).JSON(fiber.Map{"msg": "supervisor assigned successfully", "data": proj})
}

// GetAll retrieves all projects with optional filters and pagination
func GetAll(c *fiber.Ctx, db *gorm.DB) error {
	var projects []Project
	query := db.Model(&Project{})

	// For students, only show published (approved) projects
	// Always enforce published status for students, regardless of query params
	role, ok := c.Locals("role").(string)
	statusParam := c.Query("status")

	// Always filter to published projects for students
	// Also filter if status param is explicitly "published" (for extra safety)
	if ok && role == "student" {
		// Students can only see published projects - ignore any status query param
		query = query.Where("status = ?", "published")
	} else if statusParam == "published" {
		// If status param is explicitly "published", enforce it regardless of role
		query = query.Where("status = ?", "published")
	} else {
		// Filter by status for other roles if provided
		if statusParam != "" {
			query = query.Where("status = ?", statusParam)
		}
	}

	// Filter by partnerId
	// partnerId can be either a user_id or an organization_id
	// If it's an organization_id (for partner organizations), we need to get the user_id from that organization
	if partnerId := c.Query("partnerId"); partnerId != "" {
		partnerIdUint, err := strconv.ParseUint(partnerId, 10, 32)
		if err == nil {
			// First, check if this is a partner organization ID
			var org organization.Organization
			if err := db.First(&org, uint(partnerIdUint)).Error; err == nil {
				// Check if it's a partner organization (type is "company" or "partner")
				orgType := strings.ToLower(org.Type)
				if orgType == "company" || orgType == "partner" {
					// It's a partner organization ID, use the organization's UserID
					query = query.Where("user_id = ?", org.UserID)
				} else {
					// It's not a partner organization, treat it as a user_id directly
					query = query.Where("user_id = ?", uint(partnerIdUint))
				}
			} else {
				// It's not an organization ID, treat it as a user_id directly
				query = query.Where("user_id = ?", uint(partnerIdUint))
			}
		}
	}

	// Filter by universityId (via department)
	if universityId := c.Query("universityId"); universityId != "" {
		universityIdUint, err := strconv.ParseUint(universityId, 10, 32)
		if err == nil {
			query = query.Joins("JOIN departments ON projects.department_id = departments.id").
				Where("departments.organization_id = ?", uint(universityIdUint))
		}
	}

	// Filter by supervisorId
	if supervisorId := c.Query("supervisorId"); supervisorId != "" {
		supervisorIdUint, err := strconv.ParseUint(supervisorId, 10, 32)
		if err == nil {
			query = query.Where("supervisor_id = ?", uint(supervisorIdUint))
		}
	}

	// Filter by departmentId
	if departmentId := c.Query("departmentId"); departmentId != "" {
		departmentIdUint, err := strconv.ParseUint(departmentId, 10, 32)
		if err == nil {
			query = query.Where("department_id = ?", uint(departmentIdUint))
		}
	}

	// Get total count before pagination
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to count projects: " + err.Error()})
	}

	// Pagination
	page := 1
	limit := 10
	if pageStr := c.Query("page"); pageStr != "" {
		if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := (page - 1) * limit
	totalPages := int((total + int64(limit) - 1) / int64(limit)) // Ceiling division

	// Apply pagination
	query = query.Offset(offset).Limit(limit)

	// Preload relations and execute query
	if err := query.Preload("User").Preload("Supervisor").Preload("Department").Preload("Department.Organization").Preload("Course").Find(&projects).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get projects: " + err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":       projects,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
	})
}

// GetByID retrieves a project by ID
func GetByID(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	var proj Project

	if err := db.Preload("User").Preload("Supervisor").Preload("Department").Preload("Department.Organization").Preload("Course").First(&proj, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "project not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get project: " + err.Error()})
	}

	return c.JSON(fiber.Map{"data": proj})
}

// Delete deletes a project
func Delete(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	var proj Project

	if err := db.First(&proj, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "project not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to find project"})
	}

	userID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)
	if role != "super-admin" && proj.UserID != userID {
		return c.Status(403).JSON(fiber.Map{"msg": "you don't have permission to delete this project"})
	}

	if err := db.Delete(&proj).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to delete project: " + err.Error()})
	}

	return c.JSON(fiber.Map{"msg": "project deleted successfully"})
}

// UploadFiles handles file uploads for projects
func UploadFiles(c *fiber.Ctx, db *gorm.DB) error {
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to parse form: " + err.Error()})
	}

	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(400).JSON(fiber.Map{"msg": "no files provided"})
	}

	// Create uploads directory if it doesn't exist
	uploadDir := "uploads/projects"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to create upload directory"})
	}

	var paths []string
	for _, file := range files {
		// Generate unique filename
		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), file.Filename)
		filePath := filepath.Join(uploadDir, filename)

		if err := c.SaveFile(file, filePath); err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to save file: " + err.Error()})
		}

		paths = append(paths, filePath)
	}

	return c.JSON(fiber.Map{
		"msg":  "files uploaded successfully",
		"data": fiber.Map{"paths": paths},
	})
}
