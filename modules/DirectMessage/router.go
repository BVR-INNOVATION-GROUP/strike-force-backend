package directmessage

import (
	user "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/User"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// RegisterRoutes registers direct message HTTP routes.
func RegisterRoutes(r fiber.Router, db *gorm.DB) {
	participants := []string{"super-admin", "partner", "university-admin"}

	r.Post("/direct-messages/threads", user.JWTProtect([]string{"super-admin"}), func(c *fiber.Ctx) error {
		return CreateOrGetThread(c, db)
	})

	r.Get("/direct-messages/university-org/:orgId/admin-user", user.JWTProtect([]string{"super-admin"}), func(c *fiber.Ctx) error {
		return GetUniversityAdminForOrg(c, db)
	})

	g := r.Group("/direct-messages", user.JWTProtect(participants))
	g.Get("/threads", func(c *fiber.Ctx) error {
		return ListThreads(c, db)
	})
	g.Get("/threads/:id/messages", func(c *fiber.Ctx) error {
		return ListMessages(c, db)
	})
	g.Post("/threads/:id/messages", func(c *fiber.Ctx) error {
		return SendMessage(c, db)
	})
}
