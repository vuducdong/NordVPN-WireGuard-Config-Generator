package main

import (
	"embed"
	"time"

	"nordgen/internal/handler"
	"nordgen/internal/middleware"
	"nordgen/internal/store"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

//go:embed all:public
var publicFS embed.FS

func main() {
	store.Core.Init(publicFS)

	app := fiber.New(fiber.Config{
		BodyLimit:    4 * 1024 * 1024,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
		ProxyHeader:  "X-Forwarded-For",
		JSONEncoder:  sonic.Marshal,
		JSONDecoder:  sonic.Unmarshal,
	})

	app.Use(recover.New())
	app.Use(cors.New())

	api := app.Group("/api")
	api.Use(middleware.OriginGuard)

	stdLimiter := middleware.NewLimiter(100, 1*time.Minute, "Rate limit exceeded")
	heavyLimiter := middleware.NewLimiter(5, 1*time.Minute, "Rate limit exceeded for batch generation")

	api.Get("/servers", stdLimiter, handler.GetServers)
	api.Post("/key", stdLimiter, handler.ExchangeToken)

	api.Post("/config", stdLimiter, handler.GenerateConfig("text"))
	api.Post("/config/download", stdLimiter, handler.GenerateConfig("file"))
	api.Post("/config/qr", stdLimiter, handler.GenerateConfig("qr"))

	api.Post("/config/batch", heavyLimiter, handler.GenerateBatch)

	app.Use(handler.ServeFallback)

	app.Listen(":3000", fiber.ListenConfig{
		DisableStartupMessage: true,
	})
}
