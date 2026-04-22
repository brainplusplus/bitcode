package middleware

import (
	"github.com/gofiber/fiber/v2"
)

type PermissionChecker interface {
	UserHasPermission(userID string, permission string) (bool, error)
}

func PermissionMiddleware(checker PermissionChecker, requiredPermissions []string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if len(requiredPermissions) == 0 {
			return c.Next()
		}

		userID, ok := c.Locals("user_id").(string)
		if !ok || userID == "" {
			return c.Status(401).JSON(fiber.Map{"error": "not authenticated"})
		}

		for _, perm := range requiredPermissions {
			allowed, err := checker.UserHasPermission(userID, perm)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": "permission check failed"})
			}
			if !allowed {
				return c.Status(403).JSON(fiber.Map{"error": "permission denied", "required": perm})
			}
		}

		return c.Next()
	}
}
