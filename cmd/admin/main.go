package main

import (
	"fmt"
	"shortener/auth"
	"shortener/configs"
	"shortener/models"
	urls "shortener/urls"

	"github.com/gofiber/fiber/v2"
)

func main() {
	fmt.Println("Starting the application...")
	configs.InitConfig()
	models.InitDb()
	auth.InitTokenParams()
	fmt.Println("Database initialized successfully")

	fmt.Println("Initializing URL and User services...")
	app := fiber.New()
	app.Get("/:code", urls.RedirectUrl)
	app.All("*", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Endpoint not found",
		})
	})

	fmt.Printf("Starting server on port %v...\n", configs.AppConfig.AdminPort)
	if err := app.Listen(fmt.Sprintf(":%v", configs.AppConfig.AdminPort)); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
