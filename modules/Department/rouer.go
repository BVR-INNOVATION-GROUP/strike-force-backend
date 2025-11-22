package department

import (
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterRRoutes(r fiber.Router, db *gorm.DB) {
	departments := r.Group("/departments", user.JWTProtect([]string{"university-admin"}))

	departments.Post("/", func(c *fiber.Ctx) error {
		return Create(c, db)
	})

}
