package project

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func Create(c *fiber.Ctx, db *gorm.DB) error {

	var project Project

	if err := c.BodyParser(&project); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to get project details"})
	}

	project.UserID = c.Locals("user_id").(uint)

	if err := db.Create(&project).Error; err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to add project"})
	}

	data := fiber.Map{
		"msg":  "project created successfully",
		"data": project,
	}

	return c.Status(201).JSON(data)

}
