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

}
