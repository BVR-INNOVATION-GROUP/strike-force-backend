package portfolio

import (
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterRoutes(r fiber.Router, db *gorm.DB) {
	portfolio := r.Group("/portfolio", user.JWTProtect([]string{"student", "partner", "university-admin", "super-admin"}))

	portfolio.Get("/", func(c *fiber.Ctx) error {
		return GetAll(c, db)
	})

	portfolio.Get("/:id", func(c *fiber.Ctx) error {
		return GetByID(c, db)
	})

	portfolio.Post("/", func(c *fiber.Ctx) error {
		return Create(c, db)
	})

	portfolio.Put("/:id", func(c *fiber.Ctx) error {
		return Update(c, db)
	})

	portfolio.Delete("/:id", func(c *fiber.Ctx) error {
		return Delete(c, db)
	})
}











