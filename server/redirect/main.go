package main

import (
	"fmt"
	"shortener/cache"
	"shortener/configs"
	"shortener/db"
	"shortener/ratelimit"
	redirect "shortener/redirect"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	fmt.Println("Starting the application...")
	configs.InitConfig()
	db.InitDb()
	db.InitUrlRedictDb()
	cache.InitCache()
	fmt.Println("Database initialized successfully")

	fmt.Println("Initializing URL services...")
	app := fiber.New(fiber.Config{
		ProxyHeader:             configs.AppConfig.ProxyHeader,
		EnableTrustedProxyCheck: len(configs.AppConfig.TrustedProxies) > 0,
		TrustedProxies:          configs.AppConfig.TrustedProxies,
		EnableIPValidation:      true,
	})
	app.Use(recover.New())
	app.Get("/:code", ratelimit.RedirectByIP(), redirect.RedirectUrl)
	app.All("*", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Endpoint not found",
		})
	})

	fmt.Printf("Starting server on port %v...\n", configs.AppConfig.RedirectPort)
	if err := app.Listen(fmt.Sprintf(":%v", configs.AppConfig.RedirectPort)); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
