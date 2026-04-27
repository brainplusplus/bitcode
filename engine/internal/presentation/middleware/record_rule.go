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

		if len(filters) > 0 {
			filters = interpolateFilters(filters, userID)
		}

		c.Locals("rls_filters", filters)
		return c.Next()
	}
}

func interpolateFilters(filters [][]any, userID string) [][]any {
	result := make([][]any, len(filters))
	for i, f := range filters {
		newF := make([]any, len(f))
		copy(newF, f)
		for j, v := range newF {
			if s, ok := v.(string); ok {
				if s == "{{user.id}}" {
					newF[j] = userID
				}
			}
		}
		result[i] = newF
	}
	return result
}
