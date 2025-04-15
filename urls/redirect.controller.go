package urls

import (
	"github.com/gofiber/fiber/v2"
)

func RedirectUrl(ctx *fiber.Ctx) error {
	code := ctx.Params("code")
	originalURL, err := Expand(code)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "URL not found",
		})
	}

	return ctx.Status(fiber.StatusSeeOther).Redirect(originalURL)
}
