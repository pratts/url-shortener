package controllers

import (
	"fmt"
	"shortener/models"
	"shortener/services"

	"github.com/gofiber/fiber/v2"
)

func Init() {
	services.InitMap()
}

func InitUrls() {
	app := fiber.New()
	app.Get("/:code", expand)
	app.Post("/", shorten)

	fmt.Println("Server started at http://localhost:8080")

	app.Listen(":8080")
}

func shorten(ctx *fiber.Ctx) error {
	fmt.Println("shorten called")

	var urlInput models.UrlInput
	if err := ctx.BodyParser(&urlInput); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	if urlInput.URL == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "URL is required",
		})
	}

	response := services.Shorten(urlInput.URL)
	return ctx.Status(fiber.StatusCreated).JSON(response)
}

func expand(ctx *fiber.Ctx) error {
	code := ctx.Params("code")
	fmt.Println("expand called with code:", code)
	originalURL, err := services.Expand(code)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "URL not found",
		})
	}

	return ctx.Status(fiber.StatusSeeOther).Redirect(originalURL)
}
