package user

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var SECRET_KEY = []byte(os.Getenv("SECRET_KEY"))

func GenerateHash(password string) string {
	passwordBytes := []byte(password)
	passwordHash, _ := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.DefaultCost)
	return string(passwordHash)
}

func IsPasswordValid(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil // Return true when password is valid (err is nil)
}

func GenerateToken(user User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(SECRET_KEY)
}

func VerifyToken(tokenString string) (jwt.MapClaims, error) {

	foundToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		return SECRET_KEY, nil
	})

	if err != nil {
		return nil, err
	}

	if !foundToken.Valid {
		return nil, errors.New("token is invalid")
	}

	claims := foundToken.Claims.(jwt.MapClaims)

	return claims, nil

}

func Verify(c *fiber.Ctx) error {
	auth := c.Get("Authorization")

	if auth == "" {
		return c.Status(401).JSON(fiber.Map{"msg": "authentication required"})
	}

	tokenString := strings.TrimPrefix(auth, "Bearer ")

	if tokenString == "" {
		return c.Status(401).JSON(fiber.Map{"msg": "invalid key"})
	}

	fmt.Println(tokenString)

	claims, err := VerifyToken(tokenString)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"msg": "user sessions expired"})
	}

	return c.JSON(fiber.Map{"msg": "valid session ongoing", "data": claims})

}

func requiresOrganization(role string) bool {
	switch strings.ToLower(role) {
	case "partner", "university-admin":
		return true
	default:
		return false
	}
}

// OrganizationRow represents organization table structure to avoid import cycle
type OrganizationRow struct {
	ID         uint      `gorm:"column:id"`
	Name       string    `gorm:"column:name"`
	Type       string    `gorm:"column:type"`
	Email      string    `gorm:"column:email"`
	IsApproved bool      `gorm:"column:is_approved"`
	KYCStatus  string    `gorm:"column:kyc_status"`
	UserID     uint      `gorm:"column:user_id"`
	Website    string    `gorm:"column:website"`
	Logo       string    `gorm:"column:logo"`
	BrandColor string    `gorm:"column:brand_color"`
	Address    string    `gorm:"column:address"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func findOrganizationByUserID(db *gorm.DB, userID uint) (*OrganizationRow, error) {
	var org OrganizationRow
	if err := db.Table("organizations").Where("user_id = ?", userID).First(&org).Error; err != nil {
		return nil, err
	}
	return &org, nil
}

func mapOrganizationType(t string) string {
	switch strings.ToLower(t) {
	case "company":
		return "PARTNER"
	case "university":
		return "UNIVERSITY"
	default:
		return strings.ToUpper(t)
	}
}

func mapKYCStatus(status string, isApproved bool) string {
	switch strings.ToUpper(status) {
	case "APPROVED":
		return "APPROVED"
	case "REJECTED":
		return "REJECTED"
	case "PENDING":
		return "PENDING"
	}

	if isApproved {
		return "APPROVED"
	}
	return "PENDING"
}

func buildOrganizationResponse(org *OrganizationRow, email string) fiber.Map {
	orgEmail := org.Email
	if orgEmail == "" {
		orgEmail = email
	}

	return fiber.Map{
		"id":         org.ID,
		"name":       org.Name,
		"type":       mapOrganizationType(org.Type),
		"kycStatus":  mapKYCStatus(org.KYCStatus, org.IsApproved),
		"isApproved": org.IsApproved,
		"email":      orgEmail,
		"website":    org.Website,
		"logo":       org.Logo,
		"brandColor": org.BrandColor,
		"address":    org.Address,
		"createdAt":  org.CreatedAt,
		"updatedAt":  org.UpdatedAt,
	}
}

func Login(c *fiber.Ctx, db *gorm.DB) error {
	// Use a separate struct for login request since Password field has json:"-" in User model
	type LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var loginReq LoginRequest
	if err := c.BodyParser(&loginReq); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to verify the parsed information"})
	}

	var foundUser User
	if err := db.Where("email = ?", loginReq.Email).First(&foundUser).Error; err != nil {
		return c.Status(401).JSON(fiber.Map{"msg": "user not found"})

	}

	if !IsPasswordValid(foundUser.Password, loginReq.Password) {
		return c.Status(401).JSON(fiber.Map{"msg": "invalid password"})
	}

	// Load student record if user is a student to get courseId
	// Query students table directly to avoid import cycle
	if foundUser.Role == "student" {
		var courseID uint
		if err := db.Table("students").Where("user_id = ?", foundUser.ID).Select("course_id").Scan(&courseID).Error; err == nil && courseID > 0 {
			foundUser.CourseID = courseID
		}
	}

	var organizationPayload fiber.Map
	needsOrganizationSetup := false

	if requiresOrganization(foundUser.Role) {
		// For partners and university-admins, find org by user_id
		org, err := findOrganizationByUserID(db, foundUser.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Organization doesn't exist - delete the user for cleanup
				if deleteErr := db.Delete(&foundUser).Error; deleteErr != nil {
					// Log error but continue to return appropriate response
					fmt.Printf("Warning: failed to delete orphaned user %d: %v\n", foundUser.ID, deleteErr)
				}
				return c.Status(401).JSON(fiber.Map{"msg": "user account has been removed"})
			} else {
				return c.Status(500).JSON(fiber.Map{"msg": "failed to load organization"})
			}
		} else {
			organizationPayload = buildOrganizationResponse(org, foundUser.Email)

			orgID := org.ID
			foundUser.OrgID = &orgID

			if !org.IsApproved {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"msg":   "organization pending approval",
					"error": "ORGANIZATION_PENDING_APPROVAL",
					"data": fiber.Map{
						"organization": organizationPayload,
					},
				})
			}
		}
	} else if foundUser.Role == "student" && foundUser.CourseID > 0 {
		// For students, find org through course -> department -> organization
		var orgID uint
		if err := db.Table("courses").
			Joins("JOIN departments ON courses.department_id = departments.id").
			Where("courses.id = ?", foundUser.CourseID).
			Select("departments.organization_id").
			Scan(&orgID).Error; err == nil && orgID > 0 {
			var org OrganizationRow
			if err := db.Table("organizations").Where("id = ?", orgID).First(&org).Error; err == nil {
				organizationPayload = buildOrganizationResponse(&org, foundUser.Email)
				foundUser.OrgID = &orgID
			}
		}
	} else if foundUser.Role == "supervisor" {
		// For supervisors, find org through supervisor -> department -> organization
		var orgID uint
		if err := db.Table("supervisors").
			Joins("JOIN departments ON supervisors.department_id = departments.id").
			Where("supervisors.user_id = ?", foundUser.ID).
			Select("departments.organization_id").
			Scan(&orgID).Error; err == nil && orgID > 0 {
			var org OrganizationRow
			if err := db.Table("organizations").Where("id = ?", orgID).First(&org).Error; err == nil {
				organizationPayload = buildOrganizationResponse(&org, foundUser.Email)
				foundUser.OrgID = &orgID
			}
		}
	} else if foundUser.Role == "delegated-admin" {
		// For delegated admins, find org through delegated_access table
		var orgID uint
		if err := db.Table("delegated_accesses").
			Where("delegated_user_id = ? AND is_active = ?", foundUser.ID, true).
			Select("organization_id").
			Scan(&orgID).Error; err == nil && orgID > 0 {
			var org OrganizationRow
			if err := db.Table("organizations").Where("id = ?", orgID).First(&org).Error; err == nil {
				organizationPayload = buildOrganizationResponse(&org, foundUser.Email)
				foundUser.OrgID = &orgID

				if !org.IsApproved {
					return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
						"msg":   "organization pending approval",
						"error": "ORGANIZATION_PENDING_APPROVAL",
						"data": fiber.Map{
							"organization": organizationPayload,
						},
					})
				}
			} else {
				// Organization not found - delete the user for cleanup
				if deleteErr := db.Delete(&foundUser).Error; deleteErr != nil {
					fmt.Printf("Warning: failed to delete orphaned delegated user %d: %v\n", foundUser.ID, deleteErr)
				}
				return c.Status(401).JSON(fiber.Map{"msg": "user account has been removed"})
			}
		} else {
			// No active delegation found - delete the user for cleanup
			if deleteErr := db.Delete(&foundUser).Error; deleteErr != nil {
				fmt.Printf("Warning: failed to delete orphaned delegated user %d: %v\n", foundUser.ID, deleteErr)
			}
			return c.Status(401).JSON(fiber.Map{"msg": "user account has been removed"})
		}
	}

	token, tokenErr := GenerateToken(foundUser)

	if tokenErr != nil {
		c.Status(400).JSON(fiber.Map{"msg": "failed to verify session"})
	}

	// Password is excluded from JSON via json:"-" tag in User model

	var data = map[string]any{
		"token": token,
		"user":  foundUser,
	}

	if organizationPayload != nil {
		data["organization"] = organizationPayload
	}

	if needsOrganizationSetup {
		data["needsOrganizationSetup"] = true
	}

	return c.JSON(fiber.Map{"msg": "logged in successfully", "data": data})

}

func SignUp(c *fiber.Ctx, db *gorm.DB) error {

	var user User
	var tmpPassword = user.Password

	// Parse incoming JSON first
	if err := c.BodyParser(&user); err != nil {
		return c.Status(401).JSON(fiber.Map{"msg": "invalid credentials"})
	}

	// Hash the user-provided password
	hashed := GenerateHash(user.Password)
	user.Password = hashed

	if user.Email == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "provide the email"})
	}

	if user.Name == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "provide your full name"})
	}

	if user.Role == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "select your role as either company_admin or university_admin"})
	}

	if user.Email == "kigongovincent81@gmail.com" {
		user.Role = "super-admin"
	}

	var tmpUser User
	if err := db.Where("email = ?", user.Email).First(&tmpUser).Error; err == nil {
		return c.Status(402).JSON(fiber.Map{"msg": "user with email " + user.Email + " already exists"})
	}

	// Save to DB
	if err := db.Create(&user).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "Invalid credentials submitted"})
	}

	// return c.Status(201).JSON(fiber.Map{"msg": "account created successfully"})
	user.Password = tmpPassword
	return Login(c, db)

}

func FindOne(c *fiber.Ctx, db *gorm.DB, user_id uint) (User, error) {
	var user User
	if err := db.Where("id = ?", user_id).First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}

func CreateGroup(c *fiber.Ctx, db *gorm.DB) error {
	// Frontend sends: { name, capacity, courseId, leaderId, memberIds: [] }
	type CreateGroupRequest struct {
		Name      string `json:"name"`
		Capacity  int    `json:"capacity"`
		CourseID  uint   `json:"courseId"`
		LeaderID  uint   `json:"leaderId"`
		MemberIDs []uint `json:"memberIds"`
	}

	var req CreateGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid group data"})
	}

	// Verify the authenticated user is the leader
	authenticatedUserID := c.Locals("user_id").(uint)
	if req.LeaderID != authenticatedUserID {
		return c.Status(403).JSON(fiber.Map{"msg": "you can only create groups where you are the leader"})
	}

	// Validate leader is a student
	var leader User
	if err := db.First(&leader, req.LeaderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "leader not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to validate leader: " + err.Error()})
	}

	if leader.Role != "student" {
		return c.Status(400).JSON(fiber.Map{"msg": "leader must be a student"})
	}

	// Validate all members exist and are students (no course restriction - allow cross-campus groups)
	for _, memberID := range req.MemberIDs {
		if memberID == req.LeaderID {
			continue // Skip leader, already validated
		}

		var member User
		if err := db.First(&member, memberID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return c.Status(404).JSON(fiber.Map{"msg": fmt.Sprintf("member %d not found", memberID)})
			}
			return c.Status(400).JSON(fiber.Map{"msg": fmt.Sprintf("failed to validate member %d: %s", memberID, err.Error())})
		}

		if member.Role != "student" {
			return c.Status(400).JSON(fiber.Map{"msg": fmt.Sprintf("member %d must be a student", memberID)})
		}
	}

	// Validate capacity
	totalMembers := 1 + len(req.MemberIDs) // Leader + members
	if totalMembers > req.Capacity {
		return c.Status(400).JSON(fiber.Map{"msg": fmt.Sprintf("total members (%d) exceeds capacity (%d)", totalMembers, req.Capacity)})
	}

	// Create group with leader as UserID
	group := Group{
		UserID:   req.LeaderID, // Leader is the creator
		Name:     req.Name,
		Capacity: req.Capacity,
	}

	if err := db.Create(&group).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to create group: " + err.Error()})
	}

	// Add members (including leader) to the group
	var allMemberIDs []uint
	allMemberIDs = append(allMemberIDs, req.LeaderID) // Leader is always a member
	for _, memberID := range req.MemberIDs {
		if memberID != req.LeaderID { // Avoid duplicate
			allMemberIDs = append(allMemberIDs, memberID)
		}
	}

	// Add members to the group
	var members []User
	for _, memberID := range allMemberIDs {
		var member User
		if err := db.First(&member, memberID).Error; err == nil {
			members = append(members, member)
		}
	}

	if len(members) > 0 {
		if err := db.Model(&group).Association("Members").Append(members); err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to add members: " + err.Error()})
		}
	}

	// Reload with relations
	db.Preload("User").Preload("Members").First(&group, group.ID)

	// Transform to frontend format
	transformed := transformGroupToFrontendFormat(group)
	if group.User.CourseID > 0 {
		transformed["courseId"] = group.User.CourseID
	}

	return c.Status(201).JSON(fiber.Map{"data": transformed})
}

func AddToGroup(c *fiber.Ctx, db *gorm.DB) error {

	type Body struct {
		User  uint `json:"user"`
		Group uint `json:"group"`
	}

	var body Body

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid input"})
	}

	var group Group
	if err := db.First(&group, body.Group).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get group"})
	}

	var user User
	if err := db.First(&user, body.User).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get user"})
	}

	if err := db.Model(&group).Association("Members").Append(&user); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to add user to group" + err.Error()})
	}

	return c.SendStatus(201)
}

func RemoveFromGroup(c *fiber.Ctx, db *gorm.DB) error {

	type Body struct {
		Group uint `json:"group"`
		User  uint `json:"user"`
	}

	var body Body
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid input"})
	}

	var group Group
	if err := db.First(&group, body.Group).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to load group"})
	}

	var usr User
	if err := db.First(&usr, body.User).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to load user"})
	}

	if err := db.Model(&group).Association("Members").Delete(&usr); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to remove user"})
	}

	return c.SendStatus(200)
}

// transformGroupToFrontendFormat converts backend Group to frontend format
func transformGroupToFrontendFormat(group Group) map[string]interface{} {
	memberIDs := make([]uint, 0, len(group.Members))
	memberDetails := make([]map[string]interface{}, 0, len(group.Members))

	for _, member := range group.Members {
		memberIDs = append(memberIDs, member.ID)
		// Include member details for team member display
		memberDetails = append(memberDetails, map[string]interface{}{
			"id":    member.ID,
			"name":  member.Name,
			"email": member.Email,
			"profile": map[string]interface{}{
				"avatar":   member.Profile.Avatar,
				"bio":      member.Profile.Bio,
				"phone":    member.Profile.Phone,
				"location": member.Profile.Location,
			},
		})
	}

	// Include leader details
	leaderDetails := map[string]interface{}{
		"id":    group.User.ID,
		"name":  group.User.Name,
		"email": group.User.Email,
		"profile": map[string]interface{}{
			"avatar":   group.User.Profile.Avatar,
			"bio":      group.User.Profile.Bio,
			"phone":    group.User.Profile.Phone,
			"location": group.User.Profile.Location,
		},
	}

	return map[string]interface{}{
		"id":        group.ID,
		"courseId":  0,            // Will be set from User's CourseID if needed
		"leaderId":  group.UserID, // UserID is the leader
		"memberIds": memberIDs,
		"members":   memberDetails, // Include full member details
		"user":      leaderDetails, // Include leader details
		"name":      group.Name,
		"capacity":  group.Capacity,
		"createdAt": group.CreatedAt,
		"updatedAt": group.UpdatedAt,
	}
}

// GetAllGroups retrieves all groups with optional filters
func GetAllGroups(c *fiber.Ctx, db *gorm.DB) error {
	var groups []Group
	query := db.Model(&Group{})

	// Filter by courseId (if Group model has CourseID field)
	if courseId := c.Query("courseId"); courseId != "" {
		courseIdUint, err := strconv.ParseUint(courseId, 10, 32)
		if err == nil {
			// Note: This assumes Group has CourseID field - adjust if needed
			query = query.Where("course_id = ?", uint(courseIdUint))
		}
	}

	// Get authenticated user's role and ID
	userID := c.Locals("user_id").(uint)
	role, _ := c.Locals("role").(string)

	// Check if querying for a specific user (for university-admins viewing student details)
	// Allow university-admins to query groups for a specific userId
	if targetUserId := c.Query("userId"); targetUserId != "" && (role == "university-admin" || role == "delegated-admin") {
		targetUserIdUint, err := strconv.ParseUint(targetUserId, 10, 32)
		if err == nil {
			// Query groups for the target user (where they are leader or member)
			query = query.Where("user_id = ? OR id IN (SELECT group_id FROM user_groups WHERE user_id = ?)", uint(targetUserIdUint), uint(targetUserIdUint))
		} else {
			return c.Status(400).JSON(fiber.Map{"msg": "invalid userId parameter"})
		}
	} else {
		// Default: Filter by current user's groups (user is leader or member)
		query = query.Where("user_id = ? OR id IN (SELECT group_id FROM user_groups WHERE user_id = ?)", userID, userID)
	}

	if err := query.Preload("User").Preload("Members").Find(&groups).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get groups: " + err.Error()})
	}

	// Transform to frontend format
	transformedGroups := make([]map[string]interface{}, len(groups))
	for i, group := range groups {
		transformed := transformGroupToFrontendFormat(group)
		// Set courseId from leader's courseId
		if group.User.CourseID > 0 {
			transformed["courseId"] = group.User.CourseID
		}
		transformedGroups[i] = transformed
	}

	return c.JSON(fiber.Map{"data": transformedGroups})
}

// GetGroupByID retrieves a group by ID
func GetGroupByID(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	var group Group

	if err := db.Preload("User").Preload("Members").First(&group, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "group not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get group: " + err.Error()})
	}

	// Transform to frontend format
	transformed := transformGroupToFrontendFormat(group)
	if group.User.CourseID > 0 {
		transformed["courseId"] = group.User.CourseID
	}

	return c.JSON(fiber.Map{"data": transformed})
}

// UpdateGroup updates a group
func UpdateGroup(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)

	var group Group
	if err := db.First(&group, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "group not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to find group"})
	}

	// Check if user is the group leader
	if group.UserID != userID {
		return c.Status(403).JSON(fiber.Map{"msg": "you don't have permission to update this group"})
	}

	// Parse update data - frontend sends: { name?, capacity?, memberIds? }
	type UpdateGroupRequest struct {
		Name      *string `json:"name"`
		Capacity  *int    `json:"capacity"`
		MemberIDs []uint  `json:"memberIds"`
	}

	var req UpdateGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid update data"})
	}

	// Update basic fields
	if req.Name != nil {
		group.Name = *req.Name
	}
	if req.Capacity != nil {
		group.Capacity = *req.Capacity
	}

	if err := db.Model(&group).Updates(Group{
		Name:     group.Name,
		Capacity: group.Capacity,
	}).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to update group: " + err.Error()})
	}

	// Handle memberIds update if provided
	if req.MemberIDs != nil {
		// Validate capacity if provided
		if req.Capacity != nil {
			totalMembers := 1 + len(req.MemberIDs) // Leader + members
			if totalMembers > *req.Capacity {
				return c.Status(400).JSON(fiber.Map{"msg": fmt.Sprintf("total members (%d) exceeds capacity (%d)", totalMembers, *req.Capacity)})
			}
		} else {
			totalMembers := 1 + len(req.MemberIDs) // Leader + members
			if totalMembers > group.Capacity {
				return c.Status(400).JSON(fiber.Map{"msg": fmt.Sprintf("total members (%d) exceeds capacity (%d)", totalMembers, group.Capacity)})
			}
		}

		// Validate all members exist and are students (no course restriction - allow cross-campus groups)
		for _, memberID := range req.MemberIDs {
			if memberID == group.UserID {
				continue // Skip leader, already validated
			}

			var member User
			if err := db.First(&member, memberID).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return c.Status(404).JSON(fiber.Map{"msg": fmt.Sprintf("member %d not found", memberID)})
				}
				return c.Status(400).JSON(fiber.Map{"msg": fmt.Sprintf("failed to validate member %d: %s", memberID, err.Error())})
			}

			if member.Role != "student" {
				return c.Status(400).JSON(fiber.Map{"msg": fmt.Sprintf("member %d must be a student", memberID)})
			}
		}

		// Ensure leader is always in memberIds
		allMemberIDs := make([]uint, 0)
		allMemberIDs = append(allMemberIDs, group.UserID) // Leader is always a member
		for _, memberID := range req.MemberIDs {
			if memberID != group.UserID { // Avoid duplicate
				allMemberIDs = append(allMemberIDs, memberID)
			}
		}

		// Get all members
		var members []User
		for _, memberID := range allMemberIDs {
			var member User
			if err := db.First(&member, memberID).Error; err == nil {
				members = append(members, member)
			}
		}

		// Replace all members
		if err := db.Model(&group).Association("Members").Replace(members); err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to update members: " + err.Error()})
		}
	}

	// Reload with relations
	db.Preload("User").Preload("Members").First(&group, group.ID)

	// Transform to frontend format
	transformed := transformGroupToFrontendFormat(group)
	if group.User.CourseID > 0 {
		transformed["courseId"] = group.User.CourseID
	}

	return c.JSON(fiber.Map{
		"msg":  "group updated successfully",
		"data": transformed,
	})
}

// DeleteGroup deletes a group
func DeleteGroup(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)

	var group Group
	if err := db.First(&group, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "group not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to find group"})
	}

	// Check if user is the group leader
	if group.UserID != userID {
		return c.Status(403).JSON(fiber.Map{"msg": "you don't have permission to delete this group"})
	}

	if err := db.Delete(&group).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to delete group: " + err.Error()})
	}

	return c.JSON(fiber.Map{"msg": "group deleted successfully"})
}

// GetCurrentUser returns the current authenticated user
func GetCurrentUser(c *fiber.Ctx, db *gorm.DB) error {
	userID := c.Locals("user_id").(uint)
	var usr User

	if err := db.Preload("Groups").First(&usr, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "user not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get user: " + err.Error()})
	}

	// Load student record if user is a student to get courseId
	// Query students table directly to avoid import cycle
	if usr.Role == "student" {
		var courseID uint
		if err := db.Table("students").Where("user_id = ?", usr.ID).Select("course_id").Scan(&courseID).Error; err == nil && courseID > 0 {
			usr.CourseID = courseID
		}
	}

	// Password is excluded from JSON via json:"-" tag in User model

	return c.JSON(fiber.Map{"data": usr})
}

// GetByID retrieves a user by ID
func GetByID(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	var usr User

	if err := db.Preload("Groups").First(&usr, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "user not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get user: " + err.Error()})
	}

	// Password is excluded from JSON via json:"-" tag in User model

	return c.JSON(fiber.Map{"data": usr})
}

// GetAll retrieves all users with optional role filter
func GetAll(c *fiber.Ctx, db *gorm.DB) error {
	var users []User
	query := db.Model(&User{})

	// Filter by role
	if role := c.Query("role"); role != "" {
		query = query.Where("role = ?", role)
	}

	if err := query.Preload("Groups").Find(&users).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get users: " + err.Error()})
	}

	// Password is excluded from JSON via json:"-" tag in User model

	return c.JSON(fiber.Map{"data": users})
}

// SearchUsers searches for users with optional filters
// Query params: role, search (name/email search), limit (default 50)
func SearchUsers(c *fiber.Ctx, db *gorm.DB) error {
	var users []User
	query := db.Model(&User{})

	// Filter by role (required for security - only search students for group creation)
	role := c.Query("role")
	if role == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "role parameter is required"})
	}
	query = query.Where("role = ?", role)

	// Filter by universityId if provided (for students - filter by same university)
	if universityId := c.Query("universityId"); universityId != "" && role == "student" {
		universityID, err := strconv.ParseUint(universityId, 10, 64)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "invalid university ID"})
		}

		// Find students through course -> department -> organization relationship
		// Get all course IDs for departments in this organization
		var courseIds []uint
		if err := db.Table("courses").
			Select("courses.id").
			Joins("JOIN departments ON courses.department_id = departments.id").
			Where("departments.organization_id = ?", universityID).
			Pluck("courses.id", &courseIds).Error; err != nil {
			return c.Status(400).JSON(fiber.Map{"msg": "failed to get courses for university"})
		}

		// If no courses found for this university, return empty result
		if len(courseIds) == 0 {
			return c.JSON(fiber.Map{"data": []User{}})
		}

		// Filter users by students in those courses
		query = query.Joins("JOIN students ON users.id = students.user_id").
			Where("students.course_id IN ?", courseIds)
	}

	// Search by name or email (case-insensitive)
	// Only apply search filter if search query is provided
	if search := c.Query("search"); search != "" && strings.TrimSpace(search) != "" {
		searchPattern := "%" + strings.ToLower(strings.TrimSpace(search)) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(email) LIKE ?", searchPattern, searchPattern)
	}

	// Limit results (default 50, max 100)
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			if parsedLimit > 100 {
				limit = 100
			} else {
				limit = parsedLimit
			}
		}
	}
	query = query.Limit(limit)

	if err := query.Preload("Groups").Find(&users).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to search users: " + err.Error()})
	}

	// Password is excluded from JSON via json:"-" tag in User model

	return c.JSON(fiber.Map{"data": users})
}

// UpdateCurrentUser updates the current authenticated user (uses token's user_id)
func UpdateCurrentUser(c *fiber.Ctx, db *gorm.DB) error {
	userID := c.Locals("user_id").(uint)

	// Get current user from database
	var usr User
	if err := db.First(&usr, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "user not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to find user"})
	}

	var updateData User
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid update data: " + err.Error()})
	}

	// Don't allow updating password through this endpoint (use separate endpoint)
	updateData.Password = ""
	// Don't allow changing role
	role := c.Locals("role").(string)
	if role != "super-admin" {
		updateData.Role = usr.Role
	}

	// Update basic user fields
	if updateData.Name != "" {
		usr.Name = updateData.Name
	}
	if updateData.Email != "" {
		usr.Email = updateData.Email
	}

	// Update profile fields - handle both empty and non-empty values
	// Use pointer checks or explicit field updates to handle partial updates
	if updateData.Profile.Avatar != usr.Profile.Avatar {
		usr.Profile.Avatar = updateData.Profile.Avatar
	}
	if updateData.Profile.Bio != usr.Profile.Bio {
		usr.Profile.Bio = updateData.Profile.Bio
	}
	if updateData.Profile.Phone != usr.Profile.Phone {
		usr.Profile.Phone = updateData.Profile.Phone
	}
	if updateData.Profile.Location != usr.Profile.Location {
		usr.Profile.Location = updateData.Profile.Location
	}
	if updateData.Profile.Skills != nil {
		usr.Profile.Skills = updateData.Profile.Skills
	}

	if err := db.Save(&usr).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to update user: " + err.Error()})
	}

	// Reload with relations
	db.Preload("Groups").First(&usr, usr.ID)
	// Password is excluded from JSON via json:"-" tag in User model

	return c.JSON(fiber.Map{
		"msg":  "user updated successfully",
		"data": usr,
	})
}

// UpdateUser updates a user (admin can update any user by ID)
func UpdateUser(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)

	// Users can only update themselves unless they're admin
	var usr User
	if err := db.First(&usr, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "user not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to find user"})
	}

	// Check permission (user can update themselves, or admin can update anyone)
	role := c.Locals("role").(string)
	if usr.ID != userID && role != "super-admin" && role != "university-admin" && role != "delegated-admin" {
		return c.Status(403).JSON(fiber.Map{"msg": "you don't have permission to update this user"})
	}

	var updateData User
	if err := c.BodyParser(&updateData); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid update data: " + err.Error()})
	}

	// Don't allow updating password through this endpoint (use separate endpoint)
	updateData.Password = ""
	// Don't allow changing role unless super-admin
	if role != "super-admin" {
		updateData.Role = usr.Role
	}

	// Update basic user fields
	if updateData.Name != "" {
		usr.Name = updateData.Name
	}
	if updateData.Email != "" {
		usr.Email = updateData.Email
	}

	// Update profile fields - handle both empty and non-empty values
	// Use pointer checks or explicit field updates to handle partial updates
	if updateData.Profile.Avatar != usr.Profile.Avatar {
		usr.Profile.Avatar = updateData.Profile.Avatar
	}
	if updateData.Profile.Bio != usr.Profile.Bio {
		usr.Profile.Bio = updateData.Profile.Bio
	}
	if updateData.Profile.Phone != usr.Profile.Phone {
		usr.Profile.Phone = updateData.Profile.Phone
	}
	if updateData.Profile.Location != usr.Profile.Location {
		usr.Profile.Location = updateData.Profile.Location
	}
	if updateData.Profile.Skills != nil {
		usr.Profile.Skills = updateData.Profile.Skills
	}

	if err := db.Save(&usr).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to update user: " + err.Error()})
	}

	// Reload with relations
	db.Preload("Groups").First(&usr, usr.ID)
	// Password is excluded from JSON via json:"-" tag in User model

	return c.JSON(fiber.Map{
		"msg":  "user updated successfully",
		"data": usr,
	})
}

// DeleteUser deletes a user
func DeleteUser(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	var usr User
	if err := db.First(&usr, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "user not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to find user"})
	}

	// Only super-admin can delete users, or users can delete themselves
	if usr.ID != userID && role != "super-admin" {
		return c.Status(403).JSON(fiber.Map{"msg": "you don't have permission to delete this user"})
	}

	if err := db.Delete(&usr).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to delete user: " + err.Error()})
	}

	return c.JSON(fiber.Map{"msg": "user deleted successfully"})
}

// GetUserSettings retrieves user settings
func GetUserSettings(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	// Users can only view their own settings unless admin
	idUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid user ID"})
	}

	if uint(idUint) != userID && role != "super-admin" && role != "university-admin" {
		return c.Status(403).JSON(fiber.Map{"msg": "you don't have permission to view these settings"})
	}

	// For now, return empty settings object
	// In production, you'd have a UserSettings model
	settings := map[string]interface{}{
		"notifications": map[string]bool{
			"emailNotifications":  true,
			"milestoneUpdates":    true,
			"projectApplications": true,
			"paymentAlerts":       true,
			"weeklyReports":       true,
		},
		"accountSettings": map[string]string{
			"language": "en",
			"timezone": "UTC",
			"currency": "USD",
		},
		"security": map[string]interface{}{
			"twoFactorEnabled": false,
			"sessionTimeout":   "24h",
		},
	}

	return c.JSON(fiber.Map{"data": settings})
}

// UpdateUserSettings updates user settings
func UpdateUserSettings(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)

	// Users can only update their own settings unless admin
	idUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid user ID"})
	}

	if uint(idUint) != userID && role != "super-admin" && role != "university-admin" {
		return c.Status(403).JSON(fiber.Map{"msg": "you don't have permission to update these settings"})
	}

	var settings map[string]interface{}
	if err := c.BodyParser(&settings); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid settings data: " + err.Error()})
	}

	// In production, save to UserSettings model/table
	// For now, just return the updated settings

	return c.JSON(fiber.Map{
		"msg":  "settings updated successfully",
		"data": settings,
	})
}
