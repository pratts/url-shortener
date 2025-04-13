package main

import (
	"fmt"
	"shortener/configs"
	"shortener/models"
	urls "shortener/urls"
	users "shortener/users"

	"github.com/gofiber/fiber/v2"
)

func main() {
	fmt.Println("Starting the application...")
	configs.InitConfig()
	models.InitDb()
	fmt.Println("Database initialized successfully")

	fmt.Println("Initializing URL and User services...")
	app := fiber.New()
	users.Init(app)
	urls.Init(app)
	fmt.Println("URL and User services initialized successfully")

	app.All("*", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Endpoint not found",
		})
	})

	fmt.Printf("Starting server on port %v...\n", configs.AppConfig.Port)
	if err := app.Listen(fmt.Sprintf(":%v", configs.AppConfig.Port)); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
