package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

type TenantConfig struct {
	Enabled  bool
	Strategy string
	Header   string
}

func DefaultTenantConfig() TenantConfig {
	return TenantConfig{
		Enabled:  false,
		Strategy: "header",
		Header:   "X-Tenant-ID",
	}
}

func TenantMiddleware(cfg TenantConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !cfg.Enabled {
			return c.Next()
		}

		var tenantID string

		switch cfg.Strategy {
		case "header":
			tenantID = c.Get(cfg.Header)
		case "subdomain":
			host := c.Hostname()
			parts := strings.Split(host, ".")
			if len(parts) >= 3 {
				tenantID = parts[0]
			}
		case "path":
			tenantID = c.Params("tenant")
		}

		if tenantID == "" {
			return c.Status(400).JSON(fiber.Map{"error": "tenant ID required"})
		}

		c.Locals("tenant_id", tenantID)
		return c.Next()
	}
}
