package admin

import (
	"fmt"
	"strconv"

	"shortener/internal/httpapi"
	"shortener/internal/shortlink"

	"github.com/gofiber/fiber/v2"
)

// @Summary Create a short URL
// @Tags URLs
// @Accept json
// @Produce json
// @Param body body shortlink.Input true "Target URL"
// @Success 201 {object} shortlink.View
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 429 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /urls [post]
func (h *Handler) createLink(ctx *fiber.Ctx, userID uint64) error {
	var in shortlink.Input
	if err := ctx.BodyParser(&in); err != nil {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "Invalid request body")
	}
	view, err := h.Links.Create(ctx.UserContext(), userID, in.URL)
	if err != nil {
		return serviceError(ctx, err, "create short URL")
	}
	return ctx.Status(fiber.StatusCreated).JSON(view)
}

// @Summary List URLs
// @Description List the user's short URLs, newest first, a page at a time. Pass next_cursor from the previous page as cursor to get the next one; next_cursor is null on the last page.
// @Tags URLs
// @Produce json
// @Param limit query int false "Page size, 1-100" default(50)
// @Param cursor query string false "next_cursor from the previous page"
// @Success 200 {object} shortlink.Page
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /urls [get]
func (h *Handler) listLinks(ctx *fiber.Ctx, userID uint64) error {
	limit := defaultPageSize
	if v := ctx.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxPageSize {
			return httpapi.Error(ctx, fiber.StatusBadRequest, fmt.Sprintf("limit must be between 1 and %d", maxPageSize))
		}
		limit = n
	}
	var cursor uint64
	if v := ctx.Query("cursor"); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil || n == 0 {
			return httpapi.Error(ctx, fiber.StatusBadRequest, "Invalid cursor")
		}
		cursor = n
	}
	page, err := h.Links.List(ctx.UserContext(), userID, limit, cursor)
	if err != nil {
		return serviceError(ctx, err, "fetch URLs")
	}
	return ctx.JSON(page)
}

// @Summary Get a URL
// @Tags URLs
// @Produce json
// @Param id path int true "URL ID"
// @Success 200 {object} shortlink.View
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /urls/{id} [get]
func (h *Handler) getLink(ctx *fiber.Ctx, userID uint64) error {
	id, ok := parseID(ctx)
	if !ok {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "Invalid URL ID")
	}
	view, err := h.Links.Get(ctx.UserContext(), id, userID)
	if err != nil {
		return serviceError(ctx, err, "fetch URL")
	}
	return ctx.JSON(view)
}

// @Summary Update a URL's target
// @Tags URLs
// @Accept json
// @Produce json
// @Param id path int true "URL ID"
// @Param body body shortlink.Input true "New target URL"
// @Success 200 {object} shortlink.View
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /urls/{id} [put]
func (h *Handler) updateLink(ctx *fiber.Ctx, userID uint64) error {
	id, ok := parseID(ctx)
	if !ok {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "Invalid URL ID")
	}
	var in shortlink.Input
	if err := ctx.BodyParser(&in); err != nil {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "Invalid request body")
	}
	view, err := h.Links.Update(ctx.UserContext(), id, userID, in.URL)
	if err != nil {
		return serviceError(ctx, err, "update URL")
	}
	return ctx.JSON(view)
}

// @Summary Delete a URL
// @Tags URLs
// @Param id path int true "URL ID"
// @Success 204
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /urls/{id} [delete]
func (h *Handler) deleteLink(ctx *fiber.Ctx, userID uint64) error {
	id, ok := parseID(ctx)
	if !ok {
		return httpapi.Error(ctx, fiber.StatusBadRequest, "Invalid URL ID")
	}
	if err := h.Links.Delete(ctx.UserContext(), id, userID); err != nil {
		return serviceError(ctx, err, "delete URL")
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}
