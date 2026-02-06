package course

import (
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterRoutes(r fiber.Router, db *gorm.DB) {

	courses := r.Group("/courses", user.JWTProtect([]string{"university-admin", "delegated-admin", "super-admin"}))

	courses.Post("/", func(c *fiber.Ctx) error {
		return Create(c, db)
	})

	courses.Get("/", func(c *fiber.Ctx) error {
		return FindByDept(c, db)
	})

	courses.Get("/:id", func(c *fiber.Ctx) error {
		return GetByID(c, db)
	})

	courses.Put("/:id", func(c *fiber.Ctx) error {
		return Update(c, db)
	})

	courses.Delete("/:id", func(c *fiber.Ctx) error {
		return Delete(c, db)
	})

}
