package supervisorrequest

import (
	"strconv"

	project "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Project"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// GetAll retrieves all supervisor requests with optional filters
func GetAll(c *fiber.Ctx, db *gorm.DB) error {
	var requests []SupervisorRequest
	query := db.Model(&SupervisorRequest{})

	// Filter by supervisorId
	if supervisorId := c.Query("supervisorId"); supervisorId != "" {
		supervisorIdUint, err := strconv.ParseUint(supervisorId, 10, 32)
		if err == nil {
			query = query.Where("supervisor_id = ?", uint(supervisorIdUint))
		}
	}

	// Filter by projectId
	if projectId := c.Query("projectId"); projectId != "" {
		projectIdUint, err := strconv.ParseUint(projectId, 10, 32)
		if err == nil {
			query = query.Where("project_id = ?", uint(projectIdUint))
		}
	}

	// Filter by studentId (studentOrGroupId)
	if studentId := c.Query("studentId"); studentId != "" {
		studentIdUint, err := strconv.ParseUint(studentId, 10, 32)
		if err == nil {
			query = query.Where("student_or_group_id = ?", uint(studentIdUint))
		}
	}

	if err := query.Preload("Project").Preload("Supervisor").Find(&requests).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get supervisor requests: " + err.Error()})
	}

	return c.JSON(fiber.Map{"data": requests})
}

// GetByID retrieves a supervisor request by ID
func GetByID(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	var request SupervisorRequest

	if err := db.Preload("Project").Preload("Supervisor").First(&request, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "supervisor request not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get supervisor request: " + err.Error()})
	}

	return c.JSON(fiber.Map{"data": request})
}

// Create creates a new supervisor request
func Create(c *fiber.Ctx, db *gorm.DB) error {
	type CreateRequest struct {
		ProjectID        uint   `json:"projectId"`
		SupervisorID     uint   `json:"supervisorId"`
		StudentOrGroupID uint   `json:"studentOrGroupId"`
		Message          string `json:"message"`
	}

	var req CreateRequest
	userID := c.Locals("user_id").(uint)

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid request data: " + err.Error()})
	}

	// Validate required fields
	if req.ProjectID == 0 {
		return c.Status(400).JSON(fiber.Map{"msg": "project_id is required"})
	}

	if req.SupervisorID == 0 {
		return c.Status(400).JSON(fiber.Map{"msg": "supervisor_id is required"})
	}

	// Use authenticated user's ID as studentOrGroupId if not provided
	if req.StudentOrGroupID == 0 {
		req.StudentOrGroupID = userID
	}

	// Verify project exists
	var proj project.Project
	if err := db.First(&proj, req.ProjectID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "project not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to validate project"})
	}

	// Verify supervisor exists
	var supervisor user.User
	if err := db.First(&supervisor, req.SupervisorID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "supervisor not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to validate supervisor"})
	}

	// Check if supervisor has role "supervisor"
	if supervisor.Role != "supervisor" {
		return c.Status(400).JSON(fiber.Map{"msg": "user is not a supervisor"})
	}

	// Check for duplicate pending requests
	var existingRequest SupervisorRequest
	if err := db.Where("project_id = ? AND supervisor_id = ? AND student_or_group_id = ? AND status = ?",
		req.ProjectID, req.SupervisorID, req.StudentOrGroupID, "PENDING").First(&existingRequest).Error; err == nil {
		return c.Status(400).JSON(fiber.Map{"msg": "a pending request already exists for this project and supervisor"})
	}

	// Create request
	request := SupervisorRequest{
		ProjectID:        req.ProjectID,
		SupervisorID:     req.SupervisorID,
		StudentOrGroupID: req.StudentOrGroupID,
		Message:          req.Message,
		Status:           "PENDING",
	}

	if err := db.Create(&request).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to create supervisor request: " + err.Error()})
	}

	// Preload relations for response
	if err := db.Preload("Project").Preload("Supervisor").First(&request, request.ID).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to load request details: " + err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{"data": request})
}

// Update updates a supervisor request
func Update(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	var request SupervisorRequest

	if err := db.First(&request, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "supervisor request not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get supervisor request: " + err.Error()})
	}

	type UpdateRequest struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}

	var req UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid request data: " + err.Error()})
	}

	// Update fields
	if req.Status != "" {
		// Validate status
		validStatuses := map[string]bool{"PENDING": true, "APPROVED": true, "DENIED": true}
		if !validStatuses[req.Status] {
			return c.Status(400).JSON(fiber.Map{"msg": "invalid status. Must be PENDING, APPROVED, or DENIED"})
		}
		request.Status = req.Status

		// If approved, assign supervisor to project
		if req.Status == "APPROVED" {
			var proj project.Project
			if err := db.First(&proj, request.ProjectID).Error; err == nil {
				supervisorID := request.SupervisorID
				proj.SupervisorID = &supervisorID
				if err := db.Save(&proj).Error; err != nil {
					return c.Status(400).JSON(fiber.Map{"msg": "failed to assign supervisor to project: " + err.Error()})
				}
			}
		}
	}

	if req.Message != "" {
		request.Message = req.Message
	}

	if err := db.Save(&request).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to update supervisor request: " + err.Error()})
	}

	// Preload relations for response
	if err := db.Preload("Project").Preload("Supervisor").First(&request, request.ID).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to load request details: " + err.Error()})
	}

	return c.JSON(fiber.Map{"data": request})
}

// Delete deletes a supervisor request
func Delete(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	var request SupervisorRequest

	if err := db.First(&request, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "supervisor request not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get supervisor request: " + err.Error()})
	}

	if err := db.Delete(&request).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to delete supervisor request: " + err.Error()})
	}

	return c.JSON(fiber.Map{"msg": "supervisor request deleted successfully"})
}














