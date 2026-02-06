package admin

import (
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterRoutes(r fiber.Router, db *gorm.DB) {
	admin := r.Group("/admin", user.JWTProtect([]string{"super-admin"}))

	admin.Get("/financial-summary", func(c *fiber.Ctx) error {
		return GetFinancialSummary(c, db)
	})

	admin.Get("/students", func(c *fiber.Ctx) error {
		return GetAdminStudents(c, db)
	})

	admin.Delete("/students/:id", func(c *fiber.Ctx) error {
		return DeleteAdminStudent(c, db)
	})

	admin.Get("/departments", func(c *fiber.Ctx) error {
		return GetAdminDepartments(c, db)
	})

	admin.Get("/courses", func(c *fiber.Ctx) error {
		return GetAdminCourses(c, db)
	})

	admin.Get("/supervisors", func(c *fiber.Ctx) error {
		return GetAdminSupervisors(c, db)
	})

	admin.Get("/student-surveys", func(c *fiber.Ctx) error {
		return GetAdminStudentSurveys(c, db)
	})

	admin.Get("/active-users", func(c *fiber.Ctx) error {
		return GetActiveUsers(c, db)
	})

	admin.Post("/users", func(c *fiber.Ctx) error {
		return RegisterAdminUser(c, db)
	})

	admin.Post("/users/:id/block", func(c *fiber.Ctx) error {
		return BlockAdminUser(c, db)
	})

	admin.Post("/users/:id/unblock", func(c *fiber.Ctx) error {
		return UnblockAdminUser(c, db)
	})

	admin.Delete("/users/:id", func(c *fiber.Ctx) error {
		return DeleteAdminUser(c, db)
	})

	admin.Put("/users/:id/role", func(c *fiber.Ctx) error {
		return UpdateAdminUserRole(c, db)
	})

	admin.Post("/sample-accounts", func(c *fiber.Ctx) error {
		return CreateSampleAccount(c, db)
	})

	admin.Delete("/sample-accounts", func(c *fiber.Ctx) error {
		return DeleteSampleAccounts(c, db)
	})

	admin.Post("/impersonate/:id", func(c *fiber.Ctx) error {
		return AdminImpersonate(c, db)
	})

	admin.Get("/login-logos", func(c *fiber.Ctx) error {
		return GetAdminLoginLogos(c, db)
	})
	admin.Post("/login-logos", func(c *fiber.Ctx) error {
		return CreateLoginLogo(c, db)
	})
	admin.Put("/login-logos/:id", func(c *fiber.Ctx) error {
		return UpdateLoginLogo(c, db)
	})
	admin.Delete("/login-logos/:id", func(c *fiber.Ctx) error {
		return DeleteLoginLogo(c, db)
	})

	admin.Get("/storage-usage", func(c *fiber.Ctx) error {
		return GetStorageUsage(c, db)
	})
}
