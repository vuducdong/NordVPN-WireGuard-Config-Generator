package main

import (
	"os"
	"time"

	"nordgen/internal/handler"
	"nordgen/internal/middleware"
	"nordgen/internal/store"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

func main() {
	store.Core.Init()

	allowOrigins := []string{}
	if origin := os.Getenv("FRONTEND_ORIGIN"); origin != "" {
		allowOrigins = append(allowOrigins, origin)
	}

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
