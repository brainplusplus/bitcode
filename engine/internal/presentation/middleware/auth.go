package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-engine/engine/pkg/security"
)

func AuthMiddleware(jwtCfg security.JWTConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get("Authorization")
		if header == "" {
			return c.Status(401).JSON(fiber.Map{"error": "missing authorization header"})
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(401).JSON(fiber.Map{"error": "invalid authorization format"})
		}

		claims, err := security.ValidateToken(jwtCfg, parts[1])
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("username", claims.Username)
		c.Locals("roles", claims.Roles)
		c.Locals("groups", claims.Groups)
		c.Locals("claims", claims)
		if claims.ImpersonatedBy != "" {
			c.Locals("impersonated_by", claims.ImpersonatedBy)
		}

		return c.Next()
	}
}
