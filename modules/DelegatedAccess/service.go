package delegatedaccess

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	organization "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Organization"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

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

// CreateDelegatedAccessRequest represents the request to create a delegated user
type CreateDelegatedAccessRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Create creates a delegated user and sends credentials via email
func Create(c *fiber.Ctx, db *gorm.DB) error {
	userID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	// Only university-admin can delegate access
	if role != "university-admin" {
		return c.Status(403).JSON(fiber.Map{"msg": "only university admins can delegate access"})
	}

	var req CreateDelegatedAccessRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid request data: " + err.Error()})
	}

	// Validate email
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "email is required"})
	}

	// Validate name
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "name is required"})
	}

	// Get the delegator (university admin)
	var delegator user.User
	if err := db.First(&delegator, userID).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"msg": "delegator not found"})
	}

	// Get the organization for this university admin
	var org organization.Organization
	if err := db.Where("user_id = ?", userID).First(&org).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"msg": "organization not found for this admin"})
	}

	// Check if user with this email already exists
	var existingUser user.User
	if err := db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		// User exists, check if they already have delegated access
		var existingDelegation DelegatedAccess
		if err := db.Where("delegated_user_id = ? AND organization_id = ?", existingUser.ID, org.ID).First(&existingDelegation).Error; err == nil {
			return c.Status(400).JSON(fiber.Map{"msg": "this user already has delegated access"})
		}
		// User exists but no delegation - only allow if they're already a delegated-admin
		// Otherwise, return error as we can't change their role
		if existingUser.Role != "delegated-admin" {
			return c.Status(400).JSON(fiber.Map{"msg": "a user with this email already exists with a different role"})
		}
		// User exists and is already a delegated-admin, create delegation record
		delegation := DelegatedAccess{
			DelegatedUserID: existingUser.ID,
			DelegatorID:     userID,
			OrganizationID:  org.ID,
			IsActive:        true,
		}
		if err := db.Create(&delegation).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to create delegation: " + err.Error()})
		}
		db.Preload("DelegatedUser").Preload("Organization").First(&delegation, delegation.ID)
		return c.Status(201).JSON(fiber.Map{
			"msg":  "delegated access granted to existing user",
			"data": delegation,
		})
	}

	// Generate random password
	randomPassword, err := GenerateRandomPassword(12)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to generate password"})
	}

	// Create new user with role "delegated-admin"
	newUser := user.User{
		Email:    req.Email,
		Name:     req.Name,
		Password: user.GenerateHash(randomPassword),
		Role:     "delegated-admin",
	}

	if err := db.Create(&newUser).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to create user: " + err.Error()})
	}

	// Create delegation record
	delegation := DelegatedAccess{
		DelegatedUserID: newUser.ID,
		DelegatorID:     userID,
		OrganizationID:  org.ID,
		IsActive:        true,
	}

	if err := db.Create(&delegation).Error; err != nil {
		// Rollback user creation if delegation fails
		db.Delete(&newUser)
		return c.Status(400).JSON(fiber.Map{"msg": "failed to create delegation: " + err.Error()})
	}

	// Reload with relations
	db.Preload("DelegatedUser").Preload("Organization").Preload("Delegator").First(&delegation, delegation.ID)

	// Send email with credentials
	delegatorName := delegator.Name
	if delegatorName == "" {
		delegatorName = delegator.Email
	}
	if err := SendDelegatedAccessEmail(newUser.Email, newUser.Name, randomPassword, org.Name, delegatorName); err != nil {
		// Log error but don't fail the request - user is created
		fmt.Printf("Failed to send delegated access email: %v\n", err)
	}

	return c.Status(201).JSON(fiber.Map{
		"msg":  "delegated access created successfully",
		"data": delegation,
	})
}

// GetAll returns all delegated users for the current university admin
func GetAll(c *fiber.Ctx, db *gorm.DB) error {
	userID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	// Only university-admin can view delegated users
	if role != "university-admin" {
		return c.Status(403).JSON(fiber.Map{"msg": "only university admins can view delegated access"})
	}

	// Get the organization for this university admin
	var org organization.Organization
	if err := db.Where("user_id = ?", userID).First(&org).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"msg": "organization not found for this admin"})
	}

	// Get all delegations for this organization
	var delegations []DelegatedAccess
	if err := db.Where("organization_id = ? AND delegator_id = ?", org.ID, userID).
		Preload("DelegatedUser").
		Preload("Organization").
		Preload("Delegator").
		Find(&delegations).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to fetch delegations: " + err.Error()})
	}

	return c.JSON(fiber.Map{
		"msg":  "delegated access retrieved successfully",
		"data": delegations,
	})
}

// Delete withdraws delegated access by setting IsActive to false or deleting the record
func Delete(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	// Only university-admin can withdraw access
	if role != "university-admin" {
		return c.Status(403).JSON(fiber.Map{"msg": "only university admins can withdraw delegated access"})
	}

	var delegation DelegatedAccess
	if err := db.First(&delegation, id).Error; err != nil {
		return c.Status(404).JSON(fiber.Map{"msg": "delegation not found"})
	}

	// Verify the delegator owns this delegation
	if delegation.DelegatorID != userID {
		return c.Status(403).JSON(fiber.Map{"msg": "you don't have permission to withdraw this access"})
	}

	// Delete the delegation record
	if err := db.Delete(&delegation).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to withdraw access: " + err.Error()})
	}

	// Optionally delete the user if they have no other roles/connections
	// For now, we'll just delete the delegation and leave the user
	// In production, you might want to check if user has other roles before deleting

	return c.JSON(fiber.Map{"msg": "delegated access withdrawn successfully"})
}



