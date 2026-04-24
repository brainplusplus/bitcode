package middleware

import (
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-engine/engine/internal/infrastructure/persistence"
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

func PersistentAuditMiddleware(auditRepo *persistence.AuditLogRepository, logReads bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		method := c.Method()
		path := c.Path()

		if shouldSkipPath(path) {
			return err
		}

		isWrite := method == "POST" || method == "PUT" || method == "DELETE" || method == "PATCH"
		if !isWrite && !logReads {
			return err
		}

		userID, _ := c.Locals("user_id").(string)
		durationMs := int(time.Since(start).Milliseconds())
		statusCode := c.Response().StatusCode()

		entry := persistence.AuditLogEntry{
			UserID:        userID,
			Action:        "request",
			IPAddress:     c.IP(),
			UserAgent:     truncate(c.Get("User-Agent"), 500),
			RequestMethod: method,
			RequestPath:   truncate(path, 500),
			StatusCode:    statusCode,
			DurationMs:    durationMs,
		}

		modelName, recordID := extractModelRecord(path)
		if modelName != "" {
			entry.ModelName = modelName
			entry.RecordID = recordID
		}

		auditRepo.WriteAsync(entry)

		return err
	}
}

func shouldSkipPath(path string) bool {
	if strings.HasPrefix(path, "/assets/") {
		return true
	}
	if strings.HasPrefix(path, "/favicon") {
		return true
	}
	if path == "/health" || path == "/admin/api/health" {
		return true
	}
	return false
}

func extractModelRecord(path string) (string, string) {
	if strings.HasPrefix(path, "/api/") {
		parts := strings.Split(strings.TrimPrefix(path, "/api/"), "/")
		if len(parts) >= 1 {
			model := parts[0]
			recordID := ""
			if len(parts) >= 2 {
				recordID = parts[1]
			}
			return model, recordID
		}
	}
	return "", ""
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
