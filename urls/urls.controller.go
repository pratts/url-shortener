package urls

import (
	"shortener/models"

	"shortener/auth"

	"github.com/gofiber/fiber/v2"
)

func InitUrlRoutes() func(router fiber.Router) {
	return func(router fiber.Router) {
		router.Use(auth.ValidateAuthHeader)
		router.Post("/", createShortCode)
		router.Get("/", getAllUrlDetails)
		router.Get("/:code", getUrlDetails)
		router.Put("/:code", updateUrl)
		router.Delete("/:code", deleteUrl)
	}
}

func createShortCode(ctx *fiber.Ctx) error {
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

	user := ctx.Locals("user")
	userId := user.(models.UserDto).Id
	response := CreateShortCode(urlInput.URL, userId)
	return ctx.Status(fiber.StatusCreated).JSON(response)
}

func getAllUrlDetails(ctx *fiber.Ctx) error {
	user := ctx.Locals("user")
	userId := user.(models.UserDto).Id
	urls, err := GetAllShortCodes(userId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch URLs",
		})
	}
	return ctx.Status(fiber.StatusOK).JSON(urls)
}

func getUrlDetails(ctx *fiber.Ctx) error {
	code := ctx.Params("code")
	if code == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Code is required",
		})
	}

	user := ctx.Locals("user")
	userId := user.(models.UserDto).Id
	urlDetails, err := GetUrlDetails(code, userId)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "URL not found",
		})
	}
	return ctx.Status(fiber.StatusOK).JSON(urlDetails)
}

func updateUrl(ctx *fiber.Ctx) error {
	code := ctx.Params("code")
	if code == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Code is required",
		})
	}

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

	user := ctx.Locals("user")
	userId := user.(models.UserDto).Id
	urlDetails, err := UpdateUrl(code, urlInput, userId)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "URL not found",
		})
	}
	return ctx.Status(fiber.StatusOK).JSON(urlDetails)
}

func deleteUrl(ctx *fiber.Ctx) error {
	code := ctx.Params("code")
	if code == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Code is required",
		})
	}

	user := ctx.Locals("user")
	userId := user.(models.UserDto).Id
	err := DeleteUrl(code, userId)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "URL not found",
		})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
