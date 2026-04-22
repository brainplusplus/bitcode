package middleware

import (
	"github.com/gofiber/fiber/v2"
)

type RecordRuleEngine interface {
	GetFilters(userID string, modelName string, operation string) ([][]any, error)
}

func RecordRuleMiddleware(engine RecordRuleEngine, modelName string, operation string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("user_id").(string)
		if !ok || userID == "" {
			return c.Next()
		}

		filters, err := engine.GetFilters(userID, modelName, operation)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "record rule evaluation failed"})
		}

		c.Locals("rls_filters", filters)
		return c.Next()
	}
}
