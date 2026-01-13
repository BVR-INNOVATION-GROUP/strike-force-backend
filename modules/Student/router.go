package student

import (
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterRoutes(r fiber.Router, db *gorm.DB) {
	students := r.Group("/students", user.JWTProtect([]string{"university-admin", "delegated-admin", "student"}))

	students.Post("/:courseId/bulk", func(c *fiber.Ctx) error {
		return CreateBulkForCourse(c, db)
	})

	students.Post("/:courseId", func(c *fiber.Ctx) error {
		return CreateForCourse(c, db)
	})

	students.Post("/", func(c *fiber.Ctx) error {
		return Create(c, db)
	})

	students.Get("/", func(c *fiber.Ctx) error {
		return FindByCourse(c, db)
	})

	students.Put("/:id", func(c *fiber.Ctx) error {
		return Update(c, db)
	})

	// DNA Snapshot routes (student only)
	dnaRoutes := r.Group("/students/dna", user.JWTProtect([]string{"student"}))
	dnaRoutes.Post("/snapshot", func(c *fiber.Ctx) error {
		return SubmitDNASnapshot(c, db)
	})
	dnaRoutes.Get("/snapshot", func(c *fiber.Ctx) error {
		return GetDNASnapshot(c, db)
	})

	// DNA Snapshot admin routes (university-admin, delegated-admin)
	adminDnaRoutes := r.Group("/students/:studentId/dna", user.JWTProtect([]string{"university-admin", "delegated-admin"}))
	adminDnaRoutes.Get("/snapshot", func(c *fiber.Ctx) error {
		return GetStudentDNASnapshot(c, db)
	})
}
