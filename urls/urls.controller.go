package urls

import (
	"fmt"
	"shortener/models"
	"strconv"

	"shortener/auth"

	"github.com/gofiber/fiber/v2"
)

func InitUrlRoutes() func(router fiber.Router) {
	fmt.Println("Initializing URL routes")
	return func(router fiber.Router) {
		router.Use(auth.ValidateAuthHeader)
		router.Post("/", createShortCode)
		router.Get("/", getAllUrlDetails)
		router.Get("/:id", getUrlDetails)
		router.Put("/:id", updateUrl)
		router.Delete("/:id", deleteUrl)
	}
}

// @Summary Create a short URL
// @Description Create a short URL for a given long URL
// @Tags URLs
// @Accept json
// @Produce json
// @Param urlInput body models.UrlInput true "URL Input"
// @Success 201 {object} models.UrlDto
// @Failure 400 {object} map[string]interface{}
// @Router /urls [post]
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

// @Summary Get all URLs
// @Description Get all short URLs created by the user
// @Tags URLs
// @Produce json
// @Success 200 {array} models.UrlDto
// @Failure 500 {object} map[string]interface{}
// @Router /urls [get]
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

// @Summary Get URL details
// @Description Get details of a specific short URL by ID
// @Tags URLs
// @Produce json
// @Param id path int true "URL ID"
// @Success 200 {object} models.UrlDto
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /urls/{id} [get]
func getUrlDetails(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Code is required",
		})
	}

	user := ctx.Locals("user")
	userId := user.(models.UserDto).Id

	urlId, err := strconv.Atoi(id)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid URL ID",
		})
	}
	urlDetails, err := GetUrlDetails(uint64(urlId), userId)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "URL not found",
		})
	}
	return ctx.Status(fiber.StatusOK).JSON(urlDetails)
}

// @Summary Update a URL
// @Description Update the long URL for a given short URL ID
// @Tags URLs
// @Accept json
// @Produce json
// @Param id path int true "URL ID"
// @Param urlInput body models.UrlInput true "URL Input"
// @Success 200 {object} models.UrlDto
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /urls/{id} [put]
func updateUrl(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
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

	urlId, err := strconv.Atoi(id)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid URL ID",
		})
	}
	urlDetails, err := UpdateUrl(uint64(urlId), urlInput, userId)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "URL not found",
		})
	}
	return ctx.Status(fiber.StatusOK).JSON(urlDetails)
}

// @Summary Delete a URL
// @Description Delete a short URL by ID
// @Tags URLs
// @Param id path int true "URL ID"
// @Success 204
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /urls/{id} [delete]
func deleteUrl(ctx *fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Code is required",
		})
	}

	user := ctx.Locals("user")
	userId := user.(models.UserDto).Id

	urlId, err := strconv.Atoi(id)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid URL ID",
		})
	}
	err = DeleteUrl(uint64(urlId), userId)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "URL not found",
		})
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
