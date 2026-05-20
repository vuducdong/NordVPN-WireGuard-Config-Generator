package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

func extractRefererHost(referer string) string {
	r := referer
	if idx := strings.Index(r, "://"); idx != -1 {
		r = r[idx+3:]
	}
	if idx := strings.IndexByte(r, '/'); idx != -1 {
		r = r[:idx]
	}
	if idx := strings.IndexByte(r, ':'); idx != -1 {
		r = r[:idx]
	}
	return r
}

func hostPortEquals(origin, host string) bool {
	return len(origin) > len(host) && origin[:len(host)] == host && origin[len(host)] == ':'
}

func OriginGuard(c fiber.Ctx) error {
	host := c.Hostname()
	origin := c.Get("Origin")
	referer := c.Get("Referer")

	if origin != "" {
		cleanOrg := origin
		if strings.HasPrefix(cleanOrg, "https://") {
			cleanOrg = cleanOrg[8:]
		} else if strings.HasPrefix(cleanOrg, "http://") {
			cleanOrg = cleanOrg[7:]
		}
		if cleanOrg != host && !hostPortEquals(cleanOrg, host) {
			return c.Status(403).JSON(fiber.Map{"error": "Forbidden Origin"})
		}
	}

	if referer != "" {
		if extractRefererHost(referer) != host {
			return c.Status(403).JSON(fiber.Map{"error": "Forbidden Referer"})
		}
	}

	return c.Next()
}
