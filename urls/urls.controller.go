package urls

import (
	"fmt"

	"shortener/models"

	"github.com/gofiber/fiber/v2"
)

func Init(app *fiber.App) {
	initUrls(app)
}

func initUrls(app *fiber.App) {
	app.Get("/c/:code", expand)
	app.Post("/c", shorten)
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

	response := Shorten(urlInput.URL)
	return ctx.Status(fiber.StatusCreated).JSON(response)
}

func expand(ctx *fiber.Ctx) error {
	code := ctx.Params("code")
	fmt.Println("expand called with code:", code)
	originalURL, err := Expand(code)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "URL not found",
		})
	}

	return ctx.Status(fiber.StatusSeeOther).Redirect(originalURL)
}
