package delegatedaccess

import (
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterRoutes(router fiber.Router, db *gorm.DB) {
	delegated := router.Group("/delegated-access", user.JWTProtect([]string{"university-admin", "delegated-admin"}))

	delegated.Post("/", func(c *fiber.Ctx) error {
		return Create(c, db)
	})

	delegated.Get("/", func(c *fiber.Ctx) error {
		return GetAll(c, db)
	})

	delegated.Delete("/:id", func(c *fiber.Ctx) error {
		return Delete(c, db)
	})
}



