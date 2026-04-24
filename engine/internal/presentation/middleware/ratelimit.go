package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

type RateLimitConfig struct {
	Enabled    bool
	Max        int
	Window     time.Duration
	AuthMax    int
	AuthWindow time.Duration
}

func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:    true,
		Max:        100,
		Window:     1 * time.Minute,
		AuthMax:    5,
		AuthWindow: 1 * time.Minute,
	}
}

func RateLimitMiddleware(cfg RateLimitConfig) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        cfg.Max,
		Expiration: cfg.Window,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			retryAfter := int(cfg.Window.Seconds())
			c.Set("Retry-After", fmt.Sprintf("%d", retryAfter))
			return c.Status(429).JSON(fiber.Map{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter,
			})
		},
	})
}

func AuthRateLimitMiddleware(cfg RateLimitConfig) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        cfg.AuthMax,
		Expiration: cfg.AuthWindow,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			retryAfter := int(cfg.AuthWindow.Seconds())
			c.Set("Retry-After", fmt.Sprintf("%d", retryAfter))
			return c.Status(429).JSON(fiber.Map{
				"error":       "too many attempts, please try again later",
				"retry_after": retryAfter,
			})
		},
	})
}
