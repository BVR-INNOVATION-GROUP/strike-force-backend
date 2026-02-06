package user

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func RegisterRoutes(app *fiber.App, db *gorm.DB) {

	user := app.Group("/user")

	// Public endpoints
	user.Post("/login", func(c *fiber.Ctx) error {
		return Login(c, db)
	})
	user.Post("/signup", func(c *fiber.Ctx) error {
		return SignUp(c, db)
	})
	user.Get("/verify", Verify)

	// Protected endpoints
	protected := user.Group("/", JWTProtect([]string{"*"}))

	protected.Get("/", func(c *fiber.Ctx) error {
		return GetCurrentUser(c, db)
	})

	// Update current user (uses token's user_id, no ID in URL)
	protected.Put("/", func(c *fiber.Ctx) error {
		return UpdateCurrentUser(c, db)
	})

	// Change password (authenticated user, uses token's user_id)
	protected.Put("/password", func(c *fiber.Ctx) error {
		return ChangePassword(c, db)
	})

	protected.Get("/all", func(c *fiber.Ctx) error {
		return GetAll(c, db)
	})

	// Register /search BEFORE /:id to ensure it's matched first
	protected.Get("/search", func(c *fiber.Ctx) error {
		return SearchUsers(c, db)
	})

	protected.Get("/:id", func(c *fiber.Ctx) error {
		return GetByID(c, db)
	})

	// Update user by ID (admin only)
	protected.Put("/:id", func(c *fiber.Ctx) error {
		return UpdateUser(c, db)
	})

	protected.Delete("/:id", func(c *fiber.Ctx) error {
		return DeleteUser(c, db)
	})

	// Group endpoints
	groups := protected.Group("/group")

	groups.Post("/", func(c *fiber.Ctx) error {
		return CreateGroup(c, db)
	})

	groups.Post("/add", func(c *fiber.Ctx) error {
		return AddToGroup(c, db)
	})

	groups.Post("/remove", func(c *fiber.Ctx) error {
		return RemoveFromGroup(c, db)
	})

	// User settings endpoints (in API v1)
	apiV1 := app.Group("/api/v1")
	settings := apiV1.Group("/users", JWTProtect([]string{"*"}))

	settings.Get("/:id/settings", func(c *fiber.Ctx) error {
		return GetUserSettings(c, db)
	})

	settings.Put("/:id/settings", func(c *fiber.Ctx) error {
		return UpdateUserSettings(c, db)
	})

	// Group endpoints in API v1
	apiV1Groups := apiV1.Group("/groups", JWTProtect([]string{"*"}))

	apiV1Groups.Get("/", func(c *fiber.Ctx) error {
		return GetAllGroups(c, db)
	})

	apiV1Groups.Get("/:id", func(c *fiber.Ctx) error {
		return GetGroupByID(c, db)
	})

	apiV1Groups.Put("/:id", func(c *fiber.Ctx) error {
		return UpdateGroup(c, db)
	})

	apiV1Groups.Delete("/:id", func(c *fiber.Ctx) error {
		return DeleteGroup(c, db)
	})
}
