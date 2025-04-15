package main

import (
	"fmt"
	"shortener/auth"
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
	auth.InitTokenParams()
	fmt.Println("Database initialized successfully")

	fmt.Println("Initializing URL and User services...")
	app := fiber.New()

	apiV1 := app.Group("/api/v1")
	apiV1.Route("/users", users.InitUserRoutes())
	apiV1.Route("/urls", urls.InitUrlRoutes())
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
