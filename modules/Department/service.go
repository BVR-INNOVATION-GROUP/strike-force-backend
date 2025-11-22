package department

import (
	"fmt"

	organization "github.com/BVR-INNOVATION-GROUP/strike-force-backend/modules/Organization"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func Create(c *fiber.Ctx, db *gorm.DB) error {

	var department Department

	if err := c.BodyParser(&department); err != nil {
		return c.Status(400).JSON(fiber.Map{"msg": "invalid department details"})
	}

	OrgID := organization.FindById(db, c.Locals("user_id").(uint))

	if OrgID == 0 {
		return c.Status(400).JSON(fiber.Map{"msg": "failed to add department to organization"})
	}

	department.OrganizationID = OrgID
	department.Organization.ID = OrgID
	department.Organization.User.ID = c.Locals("user_id").(uint)
	department.Organization.UserID = c.Locals("user_id").(uint)

	if err := db.Create(&department).Error; err != nil {
		{
			return c.Status(400).JSON(fiber.Map{"msg": "failed to add department" + err.Error()})

		}
	}

	data := fiber.Map{
		"data": department,
		"msg":  "Department created successfully",
	}

	fmt.Println(data)

	return c.Status(201).JSON(data)

}
