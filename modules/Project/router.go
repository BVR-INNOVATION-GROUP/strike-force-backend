package project

import (
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterRoutes(r fiber.Router, db *gorm.DB) {
	projects := r.Group("/projects", user.JWTProtect([]string{"partner"}))
	projects.Post("/", func(c *fiber.Ctx) error {
		return Create(c, db)
	})
}
