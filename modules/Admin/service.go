package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	application "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Application"
	auth "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Auth"
	chat "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Chat"
	delegatedaccess "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/DelegatedAccess"
	department "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Department"
	dispute "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Dispute"
	course "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Course"
	milestone "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Milestone"
	supervisor "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Supervisor"
	notification "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Notification"
	organization "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Organization"
	portfolio "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Portfolio"
	project 	"github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Project"
	student "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Student"
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// FinancialSummaryResponse represents platform-wide financial metrics
type FinancialSummaryResponse struct {
	TotalProjectBudget   float64            `json:"totalProjectBudget"`
	TotalMilestoneAmount float64            `json:"totalMilestoneAmount"`
	ByCurrency           map[string]float64 `json:"byCurrency,omitempty"`
	ProjectCount         int                `json:"projectCount"`
	MilestoneCount       int                `json:"milestoneCount"`
}

// GetFinancialSummary returns platform-wide financial metrics (super-admin only)
func GetFinancialSummary(c *fiber.Ctx, db *gorm.DB) error {
	var projects []project.Project
	if err := db.Find(&projects).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get projects: " + err.Error()})
	}

	var milestones []milestone.Milestone
	if err := db.Find(&milestones).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get milestones: " + err.Error()})
	}

	totalBudget := 0.0
	totalMilestone := 0.0
	byCurrency := make(map[string]float64)

	for _, p := range projects {
		val := float64(p.Budget.Value)
		totalBudget += val
		curr := p.Budget.Currency
		if curr == "" {
			curr = "USD"
		}
		byCurrency["budget_"+curr] = byCurrency["budget_"+curr] + val
	}

	for _, m := range milestones {
		val := float64(m.Amount)
		totalMilestone += val
		curr := m.Currency
		if curr == "" {
			curr = "USD"
		}
		byCurrency["milestone_"+curr] = byCurrency["milestone_"+curr] + val
	}

	return c.JSON(fiber.Map{
		"data": FinancialSummaryResponse{
			TotalProjectBudget:   totalBudget,
			TotalMilestoneAmount: totalMilestone,
			ByCurrency:           byCurrency,
			ProjectCount:         len(projects),
			MilestoneCount:       len(milestones),
		},
	})
}

// GetAdminStudents returns all students for super-admin (with optional filters)
func GetAdminStudents(c *fiber.Ctx, db *gorm.DB) error {
	query := db.Model(&student.Student{}).Preload("User").Preload("Course").Preload("Course.Department").Preload("Course.Department.Organization").Preload("Branch")

	if universityId := c.Query("universityId"); universityId != "" {
		universityIdUint, err := strconv.ParseUint(universityId, 10, 32)
		if err == nil {
			query = query.Where("course_id IN (SELECT id FROM courses WHERE department_id IN (SELECT id FROM departments WHERE organization_id = ?))", uint(universityIdUint))
		}
	}

	if departmentId := c.Query("departmentId"); departmentId != "" {
		departmentIdUint, err := strconv.ParseUint(departmentId, 10, 32)
		if err == nil {
			query = query.Where("course_id IN (SELECT id FROM courses WHERE department_id = ?)", uint(departmentIdUint))
		}
	}

	if courseId := c.Query("courseId"); courseId != "" {
		courseIdUint, err := strconv.ParseUint(courseId, 10, 32)
		if err == nil {
			query = query.Where("course_id = ?", uint(courseIdUint))
		}
	}

	var students []student.Student
	if err := query.Find(&students).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get students: " + err.Error()})
	}

	return c.JSON(fiber.Map{"data": students})
}

// clearUserReferencesForDelete clears or sets null all references to the user so the user row can be deleted.
func clearUserReferencesForDelete(tx *gorm.DB, uid uint) error {
	if err := tx.Exec("DELETE FROM user_groups WHERE user_id = ?", uid).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("sender_id = ?", uid).Delete(&chat.Message{}).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("user_id = ?", uid).Delete(&notification.Notification{}).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("user_id = ?", uid).Delete(&portfolio.PortfolioItem{}).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("user_id = ?", uid).Delete(&auth.PasswordResetToken{}).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("delegated_user_id = ? OR delegator_id = ?", uid, uid).Delete(&delegatedaccess.DelegatedAccess{}).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("user_id = ?", uid).Delete(&student.Student{}).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("user_id = ?", uid).Delete(&supervisor.Supervisor{}).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("issuer_id = ? OR defendant_id = ?", uid, uid).Delete(&dispute.Dispute{}).Error; err != nil {
		return err
	}
	if err := tx.Exec("UPDATE invitations SET user_id = NULL WHERE user_id = ?", uid).Error; err != nil {
		return err
	}
	var groupIDs []uint
	if err := tx.Table("groups").Where("user_id = ?", uid).Pluck("id", &groupIDs).Error; err != nil {
		return err
	}
	for _, gid := range groupIDs {
		if err := tx.Unscoped().Where("group_id = ?", gid).Delete(&chat.Message{}).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM user_groups WHERE group_id = ?", gid).Error; err != nil {
			return err
		}
	}
	if err := tx.Table("groups").Where("user_id = ?", uid).Delete(nil).Error; err != nil {
		return err
	}
	var projectIDs []uint
	if err := tx.Table("projects").Where("user_id = ?", uid).Pluck("id", &projectIDs).Error; err != nil {
		return err
	}
	for _, pid := range projectIDs {
		if err := tx.Where("project_id = ?", pid).Delete(&application.Application{}).Error; err != nil {
			return err
		}
		if err := tx.Where("project_id = ?", pid).Delete(&milestone.Milestone{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Table("projects").Where("user_id = ?", uid).Delete(nil).Error; err != nil {
		return err
	}
	var org organization.Organization
	if err := tx.Where("user_id = ?", uid).First(&org).Error; err == nil {
		if err := tx.Delete(&org).Error; err != nil {
			return err
		}
	}
	return nil
}

// DeleteAdminStudent deletes a student (and associated user) - super-admin only
func DeleteAdminStudent(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "student id is required"})
	}
	studentIdUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid student id"})
	}

	var s student.Student
	if err := db.First(&s, uint(studentIdUint)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "student not found"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to find student"})
	}

	userID := s.UserID

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&s).Error; err != nil {
			return err
		}
		if err := clearUserReferencesForDelete(tx, userID); err != nil {
			return err
		}
		return tx.Table("users").Where("id = ?", userID).Delete(nil).Error
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to delete student: " + err.Error()})
	}

	if adminIDRaw := c.Locals("user_id"); adminIDRaw != nil {
		sid := uint(studentIdUint)
		logAdminAction(db, adminIDRaw.(uint), "delete_student", "student", &sid, fmt.Sprintf("deleted student id=%d (user_id=%d)", s.ID, userID))
	}
	return c.JSON(fiber.Map{"msg": "student deleted successfully"})
}

// DeleteStudentByUserID deletes a student (and associated user) by user ID. For university-admin and delegated-admin only; student must belong to admin's organization.
func DeleteStudentByUserID(c *fiber.Ctx, db *gorm.DB) error {
	userIDParam := c.Params("userId")
	if userIDParam == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "user id is required"})
	}
	uid, err := strconv.ParseUint(userIDParam, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid user id"})
	}
	targetUserID := uint(uid)

	adminID := c.Locals("user_id").(uint)
	role := c.Locals("role").(string)
	adminOrgID := organization.FindByIdForAdmin(db, adminID, role)
	if adminOrgID == 0 {
		return c.Status(403).JSON(fiber.Map{"msg": "you don't have permission to delete students"})
	}

	var s student.Student
	if err := db.Joins("JOIN courses ON students.course_id = courses.id").
		Joins("JOIN departments ON courses.department_id = departments.id").
		Where("students.user_id = ? AND departments.organization_id = ?", targetUserID, adminOrgID).
		First(&s).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "student not found or not in your organization"})
		}
		return c.Status(400).JSON(fiber.Map{"msg": "failed to find student"})
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&s).Error; err != nil {
			return err
		}
		if err := clearUserReferencesForDelete(tx, targetUserID); err != nil {
			return err
		}
		return tx.Table("users").Where("id = ?", targetUserID).Delete(nil).Error
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to delete student: " + err.Error()})
	}
	return c.JSON(fiber.Map{"msg": "student deleted successfully"})
}

// GetAdminDepartments returns all departments for super-admin (optional universityId filter)
func GetAdminDepartments(c *fiber.Ctx, db *gorm.DB) error {
	query := db.Model(&department.Department{}).Preload("Organization").Preload("College")
	if universityId := c.Query("universityId"); universityId != "" {
		universityIdUint, err := strconv.ParseUint(universityId, 10, 32)
		if err == nil {
			query = query.Where("organization_id = ?", uint(universityIdUint))
		}
	}
	var depts []department.Department
	if err := query.Find(&depts).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get departments: " + err.Error()})
	}
	return c.JSON(fiber.Map{"data": depts})
}

// GetAdminCourses returns all courses for super-admin (optional departmentId filter)
func GetAdminCourses(c *fiber.Ctx, db *gorm.DB) error {
	query := db.Model(&course.Course{}).Preload("Department").Preload("Department.Organization")
	if departmentId := c.Query("departmentId"); departmentId != "" {
		departmentIdUint, err := strconv.ParseUint(departmentId, 10, 32)
		if err == nil {
			query = query.Where("department_id = ?", uint(departmentIdUint))
		}
	}
	var courses []course.Course
	if err := query.Find(&courses).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get courses: " + err.Error()})
	}
	return c.JSON(fiber.Map{"data": courses})
}

// GetAdminSupervisors returns supervisors for a university (super-admin only)
func GetAdminSupervisors(c *fiber.Ctx, db *gorm.DB) error {
	universityId := c.Query("universityId")
	if universityId == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "universityId is required"})
	}
	universityIdUint, err := strconv.ParseUint(universityId, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid universityId"})
	}
	var supervisors []supervisor.Supervisor
	if err := db.Model(&supervisor.Supervisor{}).
		Preload("User").Preload("Department").
		Joins("JOIN departments ON supervisors.department_id = departments.id").
		Where("departments.organization_id = ?", uint(universityIdUint)).
		Find(&supervisors).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get supervisors: " + err.Error()})
	}
	return c.JSON(fiber.Map{"data": supervisors})
}

// DNASurveyItem represents a student's DNA snapshot for admin view
type DNASurveyItem struct {
	StudentID       uint     `json:"studentId"`
	UserID          uint     `json:"userId"`
	StudentName     string   `json:"studentName"`
	StudentEmail    string   `json:"studentEmail"`
	UniversityName  string   `json:"universityName"`
	CourseName      string   `json:"courseName"`
	HasCompleted    bool     `json:"hasCompleted"`
	DNAArchetype    *string  `json:"dnaArchetype,omitempty"`
	CompletedAt     *string  `json:"completedAt,omitempty"`
}

// GetAdminStudentSurveys returns DNA snapshot data for super-admin
func GetAdminStudentSurveys(c *fiber.Ctx, db *gorm.DB) error {
	query := db.Model(&student.Student{}).
		Preload("User").Preload("Course").Preload("Course.Department").Preload("Course.Department.Organization")

	if universityId := c.Query("universityId"); universityId != "" {
		universityIdUint, err := strconv.ParseUint(universityId, 10, 32)
		if err == nil {
			query = query.Where("course_id IN (SELECT id FROM courses WHERE department_id IN (SELECT id FROM departments WHERE organization_id = ?))", uint(universityIdUint))
		}
	}
	if courseId := c.Query("courseId"); courseId != "" {
		courseIdUint, err := strconv.ParseUint(courseId, 10, 32)
		if err == nil {
			query = query.Where("course_id = ?", uint(courseIdUint))
		}
	}
	if completed := c.Query("completed"); completed == "true" {
		query = query.Where("has_completed_dna_snapshot = ?", true)
	} else if completed == "false" {
		query = query.Where("has_completed_dna_snapshot = ?", false)
	}
	if archetype := c.Query("archetype"); archetype != "" {
		query = query.Where("dna_archetype = ?", archetype)
	}

	var students []student.Student
	if err := query.Find(&students).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get student surveys: " + err.Error()})
	}

	items := make([]DNASurveyItem, 0, len(students))
	for _, s := range students {
		completedAt := ""
		if s.DNASnapshotCompletedAt != nil {
			completedAt = s.DNASnapshotCompletedAt.Format("2006-01-02")
		}
		orgName := ""
		if s.Course.Department.Organization.ID != 0 {
			orgName = s.Course.Department.Organization.Name
		}
		items = append(items, DNASurveyItem{
			StudentID:      s.ID,
			UserID:         s.UserID,
			StudentName:    s.User.Name,
			StudentEmail:   s.User.Email,
			UniversityName: orgName,
			CourseName:     s.Course.Name,
			HasCompleted:   s.HasCompletedDNASnapshot,
			DNAArchetype:   s.DNAArchetype,
			CompletedAt:    &completedAt,
		})
	}

	return c.JSON(fiber.Map{"data": items})
}

// RegisterAdminUser creates a user (super-admin only)
func RegisterAdminUser(c *fiber.Ctx, db *gorm.DB) error {
	type Req struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Role     string `json:"role"`
		Password string `json:"password"`
		OrgID    *uint  `json:"orgId,omitempty"`
	}
	var req Req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid request"})
	}
	if req.Email == "" || req.Name == "" || req.Role == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "email, name, role, and password are required"})
	}
	validRoles := map[string]bool{"student": true, "partner": true, "supervisor": true, "university-admin": true, "delegated-admin": true}
	if !validRoles[req.Role] {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid role"})
	}
	var existing user.User
	if err := db.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return c.Status(400).JSON(fiber.Map{"msg": "email already exists"})
	}
	hashed := user.GenerateHash(req.Password)
	if hashed == "" {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to hash password"})
	}
	u := user.User{Email: req.Email, Name: req.Name, Role: req.Role, Password: hashed}
	if err := db.Create(&u).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to create user: " + err.Error()})
	}
	if adminIDRaw := c.Locals("user_id"); adminIDRaw != nil {
		tid := u.ID
		logAdminAction(db, adminIDRaw.(uint), "create_user", "user", &tid, fmt.Sprintf("created user %s (id=%d, role=%s)", u.Email, u.ID, u.Role))
	}
	return c.Status(201).JSON(fiber.Map{"msg": "user created", "data": fiber.Map{"id": u.ID, "email": u.Email, "name": u.Name, "role": u.Role}})
}

// BlockAdminUser blocks a user (super-admin only)
func BlockAdminUser(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "user id required"})
	}
	idUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid user id"})
	}
	var u user.User
	if err := db.First(&u, uint(idUint)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "user not found"})
		}
		return c.Status(500).JSON(fiber.Map{"msg": err.Error()})
	}
	if u.Role == "super-admin" {
		return c.Status(403).JSON(fiber.Map{"msg": "cannot block super-admin"})
	}
	if err := db.Model(&u).Update("is_blocked", true).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to block user"})
	}
	if adminIDRaw := c.Locals("user_id"); adminIDRaw != nil {
		tid := uint(idUint)
		logAdminAction(db, adminIDRaw.(uint), "block_user", "user", &tid, fmt.Sprintf("blocked user %s (id=%d)", u.Email, u.ID))
	}
	return c.JSON(fiber.Map{"msg": "user blocked"})
}

// UnblockAdminUser unblocks a user (super-admin only)
func UnblockAdminUser(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "user id required"})
	}
	idUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid user id"})
	}
	var u user.User
	if err := db.First(&u, uint(idUint)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "user not found"})
		}
		return c.Status(500).JSON(fiber.Map{"msg": err.Error()})
	}
	if err := db.Model(&u).Update("is_blocked", false).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to unblock user"})
	}
	if adminIDRaw := c.Locals("user_id"); adminIDRaw != nil {
		tid := uint(idUint)
		logAdminAction(db, adminIDRaw.(uint), "unblock_user", "user", &tid, fmt.Sprintf("unblocked user %s (id=%d)", u.Email, u.ID))
	}
	return c.JSON(fiber.Map{"msg": "user unblocked"})
}

// DeleteAdminUser deletes a user with FK-safe cascade (super-admin only)
func DeleteAdminUser(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "user id required"})
	}
	idUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid user id"})
	}
	uid := uint(idUint)

	var u user.User
	if err := db.First(&u, uid).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "user not found"})
		}
		return c.Status(500).JSON(fiber.Map{"msg": err.Error()})
	}
	if u.Role == "super-admin" {
		return c.Status(403).JSON(fiber.Map{"msg": "cannot delete super-admin"})
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := clearUserReferencesForDelete(tx, uid); err != nil {
			return err
		}
		return tx.Delete(&u).Error
	}); err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to delete user: " + err.Error()})
	}

	if adminIDRaw := c.Locals("user_id"); adminIDRaw != nil {
		logAdminAction(db, adminIDRaw.(uint), "delete_user", "user", &uid, fmt.Sprintf("deleted user %s (id=%d)", u.Email, u.ID))
	}
	return c.JSON(fiber.Map{"msg": "user deleted successfully"})
}

// UpdateAdminUserRole changes a user's role (super-admin only)
func UpdateAdminUserRole(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "user id required"})
	}
	idUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid user id"})
	}
	type Req struct {
		Role string `json:"role"`
	}
	var req Req
	if err := c.BodyParser(&req); err != nil || req.Role == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "role is required"})
	}
	validRoles := map[string]bool{"student": true, "partner": true, "supervisor": true, "university-admin": true, "delegated-admin": true, "super-admin": true}
	if !validRoles[req.Role] {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid role"})
	}
	var u user.User
	if err := db.First(&u, uint(idUint)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "user not found"})
		}
		return c.Status(500).JSON(fiber.Map{"msg": err.Error()})
	}
	if err := db.Model(&u).Update("role", req.Role).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to update role"})
	}
	if adminIDRaw := c.Locals("user_id"); adminIDRaw != nil {
		tid := uint(idUint)
		logAdminAction(db, adminIDRaw.(uint), "update_role", "user", &tid, fmt.Sprintf("changed role to %s for user %s (id=%d)", req.Role, u.Email, u.ID))
	}
	return c.JSON(fiber.Map{"msg": "role updated", "data": fiber.Map{"id": u.ID, "role": req.Role}})
}

// CreateSampleAccount creates a sample user (super-admin only)
func CreateSampleAccount(c *fiber.Ctx, db *gorm.DB) error {
	type Req struct {
		Role string `json:"role"`
		OrgID *uint `json:"orgId,omitempty"`
	}
	var req Req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid request"})
	}
	if req.Role == "" {
		req.Role = "student"
	}
	validRoles := map[string]bool{"student": true, "partner": true, "supervisor": true}
	if !validRoles[req.Role] {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid role for sample account"})
	}
	// Generate unique email
	var count int64
	db.Model(&user.User{}).Where("email LIKE ?", "sample_%@test.com").Count(&count)
	email := fmt.Sprintf("sample_%s_%d@test.com", req.Role, count+1)
	name := fmt.Sprintf("Sample %s %d", req.Role, count+1)
	hashed := user.GenerateHash("Sample123!")
	if hashed == "" {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to hash password"})
	}
	u := user.User{Email: email, Name: name, Role: req.Role, Password: hashed}
	if err := db.Create(&u).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to create sample user"})
	}
	if adminIDRaw := c.Locals("user_id"); adminIDRaw != nil {
		tid := u.ID
		logAdminAction(db, adminIDRaw.(uint), "create_sample_account", "user", &tid, fmt.Sprintf("created sample account %s (id=%d, role=%s)", email, u.ID, req.Role))
	}
	return c.Status(201).JSON(fiber.Map{"msg": "sample account created", "data": fiber.Map{"id": u.ID, "email": email, "name": name, "role": req.Role, "password": "Sample123!"}})
}

// DeleteSampleAccounts deletes users matching pattern (super-admin only)
func DeleteSampleAccounts(c *fiber.Ctx, db *gorm.DB) error {
	pattern := c.Query("pattern")
	if pattern == "" {
		pattern = "%@test.com"
	}
	var users []user.User
	if err := db.Where("email LIKE ?", pattern).Find(&users).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to find users"})
	}
	deleted := 0
	for _, u := range users {
		if u.Role == "super-admin" {
			continue
		}
		if err := db.Delete(&u).Error; err == nil {
			deleted++
		}
	}
	if adminIDRaw := c.Locals("user_id"); adminIDRaw != nil && deleted > 0 {
		logAdminAction(db, adminIDRaw.(uint), "delete_sample_accounts", "bulk", nil, fmt.Sprintf("deleted %d sample accounts matching pattern %s", deleted, pattern))
	}
	return c.JSON(fiber.Map{"msg": fmt.Sprintf("deleted %d sample accounts", deleted), "deleted": deleted})
}

// LoginPageLogo represents a logo displayed on the login page
type LoginPageLogo struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name" gorm:"not null"`
	LogoURL   string    `json:"logoUrl" gorm:"column:logo_url;not null"`
	AltText   string    `json:"altText" gorm:"column:alt_text"`
	SortOrder int       `json:"sortOrder" gorm:"column:sort_order;default:0"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TableName specifies the table name for LoginPageLogo
func (LoginPageLogo) TableName() string {
	return "login_page_logos"
}

// AdminAuditLog represents an audit log entry for admin actions
type AdminAuditLog struct {
	ID         uint      `gorm:"primaryKey"`
	AdminID    uint      `gorm:"column:admin_id;not null"`
	Action     string    `gorm:"column:action;not null"`
	TargetType string    `gorm:"column:target_type"`
	TargetID   *uint     `gorm:"column:target_id"`
	Details    string    `gorm:"column:details;type:text"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

// TableName specifies the table name for AdminAuditLog
func (AdminAuditLog) TableName() string {
	return "admin_audit_logs"
}

func logAdminAction(db *gorm.DB, adminID uint, action, targetType string, targetID *uint, details string) {
	log := AdminAuditLog{AdminID: adminID, Action: action, TargetType: targetType, TargetID: targetID, Details: details}
	if err := db.Create(&log).Error; err != nil {
		fmt.Printf("Admin audit log failed: %v\n", err)
	}
}

// ImpersonationLog represents an audit log entry for admin impersonation
type ImpersonationLog struct {
	ID             uint      `gorm:"primaryKey"`
	ImpersonatorID uint      `gorm:"not null"`
	TargetUserID   uint      `gorm:"not null"`
	TargetEmail    string    `gorm:"not null"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}

// TableName specifies the table name for ImpersonationLog
func (ImpersonationLog) TableName() string {
	return "impersonation_logs"
}

// AdminImpersonate returns a short-lived token for the target user (super-admin only)
func AdminImpersonate(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "user id required"})
	}
	idUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid user id"})
	}
	impersonatorID := c.Locals("user_id")
	if impersonatorID == nil {
		return c.Status(401).JSON(fiber.Map{"msg": "unauthorized"})
	}
	adminID := impersonatorID.(uint)

	var target user.User
	if err := db.First(&target, uint(idUint)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "user not found"})
		}
		return c.Status(500).JSON(fiber.Map{"msg": err.Error()})
	}
	if target.Role == "super-admin" {
		return c.Status(403).JSON(fiber.Map{"msg": "cannot impersonate super-admin"})
	}
	if target.IsBlocked {
		return c.Status(403).JSON(fiber.Map{"msg": "cannot impersonate blocked user"})
	}

	token, err := user.GenerateImpersonationToken(target, adminID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to generate token"})
	}

	// Audit log
	log := ImpersonationLog{ImpersonatorID: adminID, TargetUserID: target.ID, TargetEmail: target.Email}
	if err := db.Create(&log).Error; err != nil {
		// Log error but don't fail the request
		fmt.Printf("Impersonation audit log failed: %v\n", err)
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"token": token,
			"user": fiber.Map{
				"id":    target.ID,
				"name":  target.Name,
				"email": target.Email,
				"role":  target.Role,
			},
		},
	})
}

// GetPublicLoginLogos returns login page logos (public, no auth)
func GetPublicLoginLogos(c *fiber.Ctx, db *gorm.DB) error {
	var logos []LoginPageLogo
	if err := db.Order("sort_order ASC, id ASC").Find(&logos).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to get logos"})
	}
	return c.JSON(fiber.Map{"data": logos})
}

// GetAdminLoginLogos returns login page logos (super-admin only)
func GetAdminLoginLogos(c *fiber.Ctx, db *gorm.DB) error {
	return GetPublicLoginLogos(c, db)
}

// CreateLoginLogo creates a login page logo (super-admin only)
func CreateLoginLogo(c *fiber.Ctx, db *gorm.DB) error {
	type Req struct {
		Name      string `json:"name"`
		LogoURL   string `json:"logoUrl"`
		AltText   string `json:"altText"`
		SortOrder int    `json:"sortOrder"`
	}
	var req Req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid request"})
	}
	if req.Name == "" || req.LogoURL == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "name and logoUrl are required"})
	}
	logo := LoginPageLogo{Name: req.Name, LogoURL: req.LogoURL, AltText: req.AltText, SortOrder: req.SortOrder}
	if err := db.Create(&logo).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to create logo"})
	}
	return c.Status(201).JSON(fiber.Map{"data": logo})
}

// UpdateLoginLogo updates a login page logo (super-admin only)
func UpdateLoginLogo(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "id required"})
	}
	idUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid id"})
	}
	type Req struct {
		Name      *string `json:"name"`
		LogoURL   *string `json:"logoUrl"`
		AltText   *string `json:"altText"`
		SortOrder *int    `json:"sortOrder"`
	}
	var req Req
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid request"})
	}
	var logo LoginPageLogo
	if err := db.First(&logo, uint(idUint)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(404).JSON(fiber.Map{"msg": "logo not found"})
		}
		return c.Status(500).JSON(fiber.Map{"msg": err.Error()})
	}
	if req.Name != nil {
		logo.Name = *req.Name
	}
	if req.LogoURL != nil {
		logo.LogoURL = *req.LogoURL
	}
	if req.AltText != nil {
		logo.AltText = *req.AltText
	}
	if req.SortOrder != nil {
		logo.SortOrder = *req.SortOrder
	}
	if err := db.Save(&logo).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to update logo"})
	}
	return c.JSON(fiber.Map{"data": logo})
}

// DeleteLoginLogo deletes a login page logo (super-admin only)
func DeleteLoginLogo(c *fiber.Ctx, db *gorm.DB) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "id required"})
	}
	idUint, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid id"})
	}
	if err := db.Where("id = ?", uint(idUint)).Delete(&LoginPageLogo{}).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to delete logo"})
	}
	return c.JSON(fiber.Map{"msg": "logo deleted"})
}

// GetStorageUsage returns Cloudinary storage usage (super-admin only)
func GetStorageUsage(c *fiber.Ctx, db *gorm.DB) error {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")
	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return c.JSON(fiber.Map{
			"data": fiber.Map{
				"configured": false,
				"storage":    int64(0),
				"bandwidth":  int64(0),
				"resources":  int64(0),
			},
		})
	}
	url := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/usage", cloudName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to create request"})
	}
	req.SetBasicAuth(apiKey, apiSecret)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to fetch usage"})
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to parse usage"})
	}
	storage := int64(0)
	if v, ok := result["storage"].(map[string]interface{}); ok {
		if s, ok := v["usage"].(float64); ok {
			storage = int64(s)
		}
	}
	bandwidth := int64(0)
	if v, ok := result["bandwidth"].(map[string]interface{}); ok {
		if b, ok := v["usage"].(float64); ok {
			bandwidth = int64(b)
		}
	}
	resources := int64(0)
	if v, ok := result["resources"].(map[string]interface{}); ok {
		if r, ok := v["usage"].(float64); ok {
			resources = int64(r)
		}
	}
	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"configured": true,
			"storage":    storage,
			"bandwidth":  bandwidth,
			"resources":  resources,
		},
	})
}

// GetActiveUsers returns users with last_login_at in last N minutes (super-admin only)
func GetActiveUsers(c *fiber.Ctx, db *gorm.DB) error {
	minutes := c.Query("minutes")
	if minutes == "" {
		minutes = "60"
	}
	m, err := strconv.Atoi(minutes)
	if err != nil || m <= 0 {
		m = 60
	}
	since := time.Now().Add(-time.Duration(m) * time.Minute)
	var users []user.User
	if err := db.Where("last_login_at IS NOT NULL AND last_login_at > ?", since).
		Select("id", "email", "name", "role", "last_login_at").
		Find(&users).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"msg": "failed to get active users"})
	}
	return c.JSON(fiber.Map{"data": users})
}
