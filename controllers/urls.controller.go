package controllers

import (
	"fmt"
	"shortener/configs"
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

	fmt.Println(fmt.Sprintf("Server is running on port %v", configs.AppConfig.Port))

	app.Listen(fmt.Sprintf(":%v", configs.AppConfig.Port))
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
