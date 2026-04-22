package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

func AuditMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		method := c.Method()
		if method == "POST" || method == "PUT" || method == "DELETE" || method == "PATCH" {
			userID, _ := c.Locals("user_id").(string)
			log.Printf("[AUDIT] %s %s by user=%s status=%d duration=%s",
				method, c.Path(), userID, c.Response().StatusCode(), time.Since(start))
		}

		return err
	}
}
