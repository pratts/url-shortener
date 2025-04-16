package urls

import (
	"fmt"
	"shortener/cache"

	"github.com/gofiber/fiber/v2"
)

func RedirectUrl(ctx *fiber.Ctx) error {
	code := ctx.Params("code")

	val, err := cache.GetFromCache(code)
	if err == nil && val != "" {
		fmt.Println("Cache hit for code:", code)
		return ctx.Status(fiber.StatusSeeOther).Redirect(val)
	}
	originalURL, err := Expand(code)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "URL not found",
		})
	}

	if err := cache.SetToCache(code, originalURL); err != nil {
		fmt.Println("Error setting cache:", err)
	}
	return ctx.Status(fiber.StatusSeeOther).Redirect(originalURL)
}
