package organization

import (
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterRoutes(router fiber.Router, db *gorm.DB) {

	org := router.Group("/org", user.JWTProtect([]string{"partner", "university-admin", "super-admin"}))

	org.Post("/", func(c *fiber.Ctx) error {
		return Register(c, db)
	})
	org.Get("/", func(c *fiber.Ctx) error {
		t := c.Query("type")
		if t != "" {
			return GetByType(c, db, t)
		}
		// If no type specified, return all organizations (admin only)
		return GetAll(c, db)
	})

	// Nested endpoint for select forms (organizations with departments and courses)
	org.Get("/nested", func(c *fiber.Ctx) error {
		return GetNestedWithDepartmentsAndCourses(c, db)
	})

	// Partner dashboard stats endpoint - must be before /:id routes to avoid route conflict
	org.Get("/partner/dashboard", func(c *fiber.Ctx) error {
		return GetPartnerDashboardStats(c, db)
	})

	org.Get("/:id", func(c *fiber.Ctx) error {
		return GetByID(c, db)
	})

	org.Get("/:id/dashboard", func(c *fiber.Ctx) error {
		return GetDashboardStats(c, db)
	})

	org.Put("/:id", func(c *fiber.Ctx) error {
		return Update(c, db)
	})

	org.Delete("/:id", func(c *fiber.Ctx) error {
		return Delete(c, db)
	})

	org.Post("/upload-logo", func(c *fiber.Ctx) error {
		return UploadLogo(c, db)
	})

}
