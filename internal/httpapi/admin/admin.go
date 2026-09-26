// Package admin serves the admin API: accounts and link management.
package admin

import (
	"errors"
	"log/slog"
	"strconv"
	"time"

	"shortener/internal/auth"
	"shortener/internal/httpapi"
	"shortener/internal/ratelimit"
	"shortener/internal/shortlink"
	"shortener/internal/user"

	"github.com/gofiber/fiber/v2"
)

const (
	defaultPageSize = 50
	maxPageSize     = 100
)

// Handler serves the admin API.
type Handler struct {
	Links               *shortlink.Service
	Users               *user.Service
	Tokens              *auth.TokenService
	Limits              *ratelimit.Limits
	RegistrationEnabled bool
}

// Register mounts the API routes on router (normally /api/v1).
func (h *Handler) Register(router fiber.Router) {
	requireAuth := h.Tokens.Middleware()

	users := router.Group("/users")
	users.Post("/login", h.Limits.LoginByIP(), h.Limits.LoginByAccount(), h.login)
	if h.RegistrationEnabled {
		users.Post("/register", h.Limits.RegisterByIP(), h.register)
	}
	users.Get("/me", requireAuth, withUser(h.getMe))
	users.Patch("/me", requireAuth, withUser(h.updateMe))

	urls := router.Group("/urls", requireAuth)
	urls.Post("/", h.Limits.PerUser("create-url", 30, time.Minute), withUser(h.createLink))
	urls.Get("/", withUser(h.listLinks))
	urls.Get("/:id", withUser(h.getLink))
	urls.Put("/:id", withUser(h.updateLink))
	urls.Delete("/:id", withUser(h.deleteLink))
}

// withUser passes the authenticated user's ID to fn. It must run behind the
// auth middleware; without one the request is rejected rather than served
// anonymously.
func withUser(fn func(ctx *fiber.Ctx, userID uint64) error) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		id, ok := auth.UserID(ctx)
		if !ok {
			return httpapi.Error(ctx, fiber.StatusUnauthorized, "Not authenticated")
		}
		return fn(ctx, id)
	}
}

// parseID reads the :id route parameter as a positive integer.
func parseID(ctx *fiber.Ctx) (uint64, bool) {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 64)
	return id, err == nil && id > 0
}

// serviceError maps a service error to a response. Unexpected errors are
// logged and reported as 500 with a generic message.
func serviceError(ctx *fiber.Ctx, err error, action string) error {
	var fields user.ValidationErrors
	switch {
	case errors.As(err, &fields):
		return httpapi.ValidationError(ctx, fields)
	case errors.Is(err, shortlink.ErrInvalidTarget):
		return httpapi.Error(ctx, fiber.StatusBadRequest, err.Error())
	case errors.Is(err, shortlink.ErrNotFound):
		return httpapi.Error(ctx, fiber.StatusNotFound, "URL not found")
	case errors.Is(err, user.ErrNotFound):
		return httpapi.Error(ctx, fiber.StatusNotFound, "User not found")
	case errors.Is(err, user.ErrEmailTaken):
		return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":  err.Error(),
			"fields": fiber.Map{"email": "is already registered"},
		})
	case errors.Is(err, user.ErrInvalidCredentials):
		return httpapi.Error(ctx, fiber.StatusUnauthorized, "Invalid email or password")
	case errors.Is(err, user.ErrCurrentPassword):
		return httpapi.Error(ctx, fiber.StatusForbidden, err.Error())
	}
	slog.ErrorContext(ctx.UserContext(), "request failed", "action", action,
		"request_id", ctx.GetRespHeader(fiber.HeaderXRequestID), "err", err)
	return httpapi.Error(ctx, fiber.StatusInternalServerError, "Failed to "+action)
}
