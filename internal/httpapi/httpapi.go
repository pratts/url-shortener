// Package httpapi holds what the HTTP services share: app setup and JSON
// error responses.
package httpapi

import (
	"shortener/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// NewApp returns a Fiber app with proxy handling and panic recovery.
func NewApp(cfg config.HTTP) *fiber.App {
	app := fiber.New(fiber.Config{
		ProxyHeader:             cfg.ProxyHeader,
		EnableTrustedProxyCheck: len(cfg.TrustedProxies) > 0,
		TrustedProxies:          cfg.TrustedProxies,
		EnableIPValidation:      true,
	})
	app.Use(recover.New())
	return app
}

// NotFound answers every request that matched no route.
func NotFound(ctx *fiber.Ctx) error {
	return Error(ctx, fiber.StatusNotFound, "Endpoint not found")
}

// Error writes {"error": msg} with the given status.
func Error(ctx *fiber.Ctx, status int, msg string) error {
	return ctx.Status(status).JSON(fiber.Map{"error": msg})
}

// ValidationError writes a 400 with per-field problems under "fields".
func ValidationError(ctx *fiber.Ctx, fields map[string]string) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error":  "Validation failed",
		"fields": fields,
	})
}
