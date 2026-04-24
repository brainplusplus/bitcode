package middleware

import (
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type IPWhitelistConfig struct {
	Enabled    bool
	AllowedIPs []string
	AdminOnly  bool
}

func DefaultIPWhitelistConfig() IPWhitelistConfig {
	return IPWhitelistConfig{
		Enabled:    false,
		AllowedIPs: []string{},
		AdminOnly:  true,
	}
}

func IPWhitelistMiddleware(cfg IPWhitelistConfig) fiber.Handler {
	allowed := buildIPSet(cfg.AllowedIPs)
	cidrs := buildCIDRs(cfg.AllowedIPs)

	return func(c *fiber.Ctx) error {
		if !cfg.Enabled || len(cfg.AllowedIPs) == 0 {
			return c.Next()
		}

		if cfg.AdminOnly && !strings.HasPrefix(c.Path(), "/admin") {
			return c.Next()
		}

		clientIP := c.IP()

		if allowed[clientIP] {
			return c.Next()
		}

		ip := net.ParseIP(clientIP)
		if ip != nil {
			for _, cidr := range cidrs {
				if cidr.Contains(ip) {
					return c.Next()
				}
			}
		}

		return c.Status(403).JSON(fiber.Map{
			"error": "access denied: IP not allowed",
		})
	}
}

func buildIPSet(ips []string) map[string]bool {
	set := make(map[string]bool)
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip == "" || strings.Contains(ip, "/") {
			continue
		}
		set[ip] = true
	}
	return set
}

func buildCIDRs(ips []string) []*net.IPNet {
	var cidrs []*net.IPNet
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if !strings.Contains(ip, "/") {
			continue
		}
		_, cidr, err := net.ParseCIDR(ip)
		if err == nil {
			cidrs = append(cidrs, cidr)
		}
	}
	return cidrs
}
