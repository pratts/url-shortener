package urls

import (
	"shortener/models"

	"shortener/auth"

	"github.com/gofiber/fiber/v2"
)

func InitUrlRoutes() func(router fiber.Router) {
	return func(router fiber.Router) {
		router.Post("/", auth.ValidateAuthHeader, shorten)
		router.Get("/:code", expand)
	}
}

func shorten(ctx *fiber.Ctx) error {
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
	originalURL, err := Expand(code)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "URL not found",
		})
	}

	return ctx.Status(fiber.StatusSeeOther).Redirect(originalURL)
}
