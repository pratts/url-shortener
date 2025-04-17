package main

import (
	"fmt"
	"shortener/auth"
	"shortener/cache"
	"shortener/configs"
	"shortener/db"
	redirect "shortener/redirect"

	"github.com/gofiber/fiber/v2"
)

func main() {
	fmt.Println("Starting the application...")
	configs.InitConfig()
	db.InitDb()
	db.InitUrlRedictDb()
	cache.InitCache()
	auth.InitTokenParams()
	fmt.Println("Database initialized successfully")

	fmt.Println("Initializing URL and User services...")
	app := fiber.New()
	app.Get("/:code", redirect.RedirectUrl)
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
