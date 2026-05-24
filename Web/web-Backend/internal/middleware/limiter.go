package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

func NewLimiter(limit int, window time.Duration, errMsg string) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        limit,
		Expiration: window,
		KeyGenerator: func(c fiber.Ctx) string {
			if clientIP := c.Get("X-Client-IP"); clientIP != "" {
				return clientIP
			}
			return c.IP()
		},
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": errMsg,
			})
		},
	})
}
