package middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type TenantConfig struct {
	Enabled   bool
	Strategy  string // detection: "header", "subdomain", "path"
	Header    string
	Isolation string // isolation: "shared_table" (default), "shared_schema", "separate_db"
	Column    string // column name, default "tenant_id"
}

func DefaultTenantConfig() TenantConfig {
	return TenantConfig{
		Enabled:   false,
		Strategy:  "header",
		Header:    "X-Tenant-ID",
		Isolation: "shared_table",
		Column:    "tenant_id",
	}
}

func ValidateTenantConfig(cfg TenantConfig) error {
	if !cfg.Enabled {
		return nil
	}
	switch cfg.Isolation {
	case "shared_table", "":
		return nil
	case "shared_schema":
		return fmt.Errorf("tenant isolation 'shared_schema' is not yet implemented, use 'shared_table'")
	case "separate_db":
		return fmt.Errorf("tenant isolation 'separate_db' is not yet implemented, use 'shared_table'")
	default:
		return fmt.Errorf("unknown tenant isolation strategy: %s", cfg.Isolation)
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
