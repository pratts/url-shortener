package urls

import (
	"errors"
	"fmt"
	"log"
	"shortener/models"
	"shortener/ratelimit"
	"strconv"
	"time"

	"shortener/auth"

	"github.com/gofiber/fiber/v2"
)

const (
	defaultPageSize = 50
	maxPageSize     = 100
)

func InitUrlRoutes() func(router fiber.Router) {
	fmt.Println("Initializing URL routes")
	return func(router fiber.Router) {
		router.Use(auth.ValidateAuthHeader)
		router.Post("/", ratelimit.PerUser("create-url", 30, time.Minute), createShortCode)
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
// @Failure 429 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /urls [post]
func createShortCode(ctx *fiber.Ctx) error {
	var urlInput models.UrlInput
	if err := ctx.BodyParser(&urlInput); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	target, err := ValidateTargetURL(urlInput.URL)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	user := ctx.Locals("user")
	userId := user.(models.UserDto).Id
	response, err := CreateShortCode(target, userId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create short URL",
		})
	}
	return ctx.Status(fiber.StatusCreated).JSON(response)
}

// @Summary List URLs
// @Description List the user's short URLs, newest first, a page at a time. Pass next_cursor from the previous page as cursor to get the next one; next_cursor is null on the last page.
// @Tags URLs
// @Produce json
// @Param limit query int false "Page size, 1-100" default(50)
// @Param cursor query string false "next_cursor from the previous page"
// @Success 200 {object} models.UrlPage
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /urls [get]
func getAllUrlDetails(ctx *fiber.Ctx) error {
	limit := defaultPageSize
	if v := ctx.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxPageSize {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("limit must be between 1 and %d", maxPageSize),
			})
		}
		limit = n
	}
	var cursor uint64
	if v := ctx.Query("cursor"); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil || n == 0 {
			return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid cursor",
			})
		}
		cursor = n
	}

	user := ctx.Locals("user")
	userId := user.(models.UserDto).Id
	urls, next, err := ListShortCodes(userId, limit, cursor)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch URLs",
		})
	}
	page := models.UrlPage{Items: urls}
	if next > 0 {
		cursor := strconv.FormatUint(next, 10)
		page.NextCursor = &cursor
	}
	return ctx.Status(fiber.StatusOK).JSON(page)
}

var errInvalidID = errors.New("invalid URL ID")

// parseID reads the :id route parameter as a positive integer.
func parseID(ctx *fiber.Ctx) (uint64, error) {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, errInvalidID
	}
	return id, nil
}

// lookupError responds 404 for a missing URL and 500 for anything else.
func lookupError(ctx *fiber.Ctx, err error, action string) error {
	if errors.Is(err, ErrUrlNotFound) {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "URL not found",
		})
	}
	log.Printf("failed to %s URL: %v", action, err)
	return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": fmt.Sprintf("Failed to %s URL", action),
	})
}

// @Summary Get URL details
// @Description Get details of a specific short URL by ID
// @Tags URLs
// @Produce json
// @Param id path int true "URL ID"
// @Success 200 {object} models.UrlDto
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /urls/{id} [get]
func getUrlDetails(ctx *fiber.Ctx) error {
	urlId, err := parseID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid URL ID",
		})
	}

	user := ctx.Locals("user")
	userId := user.(models.UserDto).Id
	urlDetails, err := GetUrlDetails(urlId, userId)
	if err != nil {
		return lookupError(ctx, err, "fetch")
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
// @Failure 500 {object} map[string]interface{}
// @Router /urls/{id} [put]
func updateUrl(ctx *fiber.Ctx) error {
	urlId, err := parseID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid URL ID",
		})
	}

	var urlInput models.UrlInput
	if err := ctx.BodyParser(&urlInput); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}
	target, err := ValidateTargetURL(urlInput.URL)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	urlInput.URL = target

	user := ctx.Locals("user")
	userId := user.(models.UserDto).Id
	urlDetails, err := UpdateUrl(urlId, urlInput, userId)
	if err != nil {
		return lookupError(ctx, err, "update")
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
// @Failure 500 {object} map[string]interface{}
// @Router /urls/{id} [delete]
func deleteUrl(ctx *fiber.Ctx) error {
	urlId, err := parseID(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid URL ID",
		})
	}

	user := ctx.Locals("user")
	userId := user.(models.UserDto).Id
	if err := DeleteUrl(urlId, userId); err != nil {
		return lookupError(ctx, err, "delete")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
