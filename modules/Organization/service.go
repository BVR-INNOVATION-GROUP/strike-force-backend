package organization

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func Register(c *fiber.Ctx, db *gorm.DB) error {

	var org Organization

	if err := c.BodyParser(&org); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "something is wrong with the submitted data : " + err.Error()})
	}

	if org.Type == "" {
		return c.Status(400).JSON(fiber.Map{"msg": "please select the type as either university or company"})
	}

	id := c.Locals("user_id")
	org.UserID = id.(uint)

	if err := db.Create(&org).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to add organization"})
	}

	return c.Status(201).JSON(fiber.Map{"msg": org.Type + " created successfully", "data": org})
}
