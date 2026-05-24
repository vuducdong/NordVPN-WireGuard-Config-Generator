package main

import (
	"os"
	"strings"
	"time"

	"nordgen/internal/handler"
	"nordgen/internal/middleware"
	"nordgen/internal/store"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

var cloudflareProxyCIDRs = []string{
	"103.21.244.0/22",
	"103.22.200.0/22",
	"103.31.4.0/22",
	"104.16.0.0/13",
	"104.24.0.0/14",
	"108.162.192.0/18",
	"131.0.72.0/22",
	"141.101.64.0/18",
	"162.158.0.0/15",
	"172.64.0.0/13",
	"173.245.48.0/20",
	"188.114.96.0/20",
	"190.93.240.0/20",
	"197.234.240.0/22",
	"198.41.128.0/17",
	"2400:cb00::/32",
	"2606:4700::/32",
	"2803:f800::/32",
	"2405:b500::/32",
	"2405:8100::/32",
	"2a06:98c0::/29",
	"2c0f:f248::/32",
}

func resolveTrustedProxies() []string {
	raw := os.Getenv("TRUSTED_PROXY_CIDRS")
	if raw == "" {
		return cloudflareProxyCIDRs
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return cloudflareProxyCIDRs
	}
	return out
}

func main() {
	store.Core.Init()

	allowOrigins := []string{}
	if origin := os.Getenv("FRONTEND_ORIGIN"); origin != "" {
		allowOrigins = append(allowOrigins, origin)
	}

	app := fiber.New(fiber.Config{
		BodyLimit:          4 * 1024 * 1024,
		ReadTimeout:        10 * time.Second,
		WriteTimeout:       30 * time.Second,
		IdleTimeout:        120 * time.Second,
		TrustProxy:         true,
		TrustProxyConfig:   fiber.TrustProxyConfig{Proxies: resolveTrustedProxies()},
		ProxyHeader:        fiber.HeaderXForwardedFor,
		EnableIPValidation: true,
		JSONEncoder:        sonic.Marshal,
		JSONDecoder:        sonic.Unmarshal,
	})

	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: allowOrigins,
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "If-None-Match"},
		MaxAge:       86400,
	}))

	api := app.Group("/api")

	stdLimiter := middleware.NewLimiter(100, 1*time.Minute, "Rate limit exceeded")
	heavyLimiter := middleware.NewLimiter(5, 1*time.Minute, "Rate limit exceeded for batch generation")

	api.Get("/servers", stdLimiter, handler.GetServers)
	api.Post("/key", stdLimiter, handler.ExchangeToken)

	api.Post("/config", stdLimiter, handler.GenerateConfig("text"))
	api.Post("/config/download", stdLimiter, handler.GenerateConfig("file"))
	api.Post("/config/qr", stdLimiter, handler.GenerateConfig("qr"))

	api.Post("/config/batch", heavyLimiter, handler.GenerateBatch)

	app.Listen(":3000", fiber.ListenConfig{
		DisableStartupMessage: true,
	})
}
