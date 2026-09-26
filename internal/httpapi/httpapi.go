// Package httpapi holds what the HTTP services share: app setup, health
// checks, graceful shutdown and JSON error responses.
package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"shortener/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

const shutdownTimeout = 15 * time.Second

// NewApp returns a Fiber app with server timeouts, proxy handling, panic
// recovery, request IDs, access logging and a per-request deadline.
func NewApp(cfg config.HTTP, log *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		ProxyHeader:             cfg.ProxyHeader,
		EnableTrustedProxyCheck: len(cfg.TrustedProxies) > 0,
		TrustedProxies:          cfg.TrustedProxies,
		EnableIPValidation:      true,
		ReadTimeout:             10 * time.Second,
		WriteTimeout:            10 * time.Second,
		IdleTimeout:             60 * time.Second,
		DisableStartupMessage:   true,
		ErrorHandler:            errorHandler(log),
	})
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(accessLog(log))
	app.Use(deadline(cfg.RequestTimeout))
	return app
}

// deadline gives each request a context that ends after timeout; handlers pass
// ctx.UserContext() to the database and cache.
func deadline(timeout time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), timeout)
		defer cancel()
		c.SetUserContext(ctx)
		return c.Next()
	}
}

func accessLog(log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		if err != nil {
			var fe *fiber.Error
			if errors.As(err, &fe) {
				status = fe.Code
			} else {
				status = fiber.StatusInternalServerError
			}
		}
		level := slog.LevelInfo
		if status >= 500 {
			level = slog.LevelError
		}
		log.LogAttrs(c.UserContext(), level, "request",
			slog.String("request_id", c.GetRespHeader(fiber.HeaderXRequestID)),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", c.IP()),
		)
		return err
	}
}

// errorHandler answers errors that escape handlers (including recovered
// panics) with JSON, without exposing internal details.
func errorHandler(log *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var fe *fiber.Error
		if errors.As(err, &fe) {
			return Error(c, fe.Code, fe.Message)
		}
		log.ErrorContext(c.UserContext(), "unhandled error",
			"request_id", c.GetRespHeader(fiber.HeaderXRequestID), "err", err)
		return Error(c, fiber.StatusInternalServerError, "Internal server error")
	}
}

// Check reports whether a dependency is usable.
type Check func(ctx context.Context) error

// RegisterHealth adds GET /healthz (the process is up) and GET /readyz (every
// dependency answers within a second; 503 otherwise).
func RegisterHealth(app *fiber.App, checks map[string]Check) {
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/readyz", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), time.Second)
		defer cancel()
		results := fiber.Map{}
		status := fiber.StatusOK
		for name, check := range checks {
			if err := check(ctx); err != nil {
				results[name] = err.Error()
				status = fiber.StatusServiceUnavailable
			} else {
				results[name] = "ok"
			}
		}
		return c.Status(status).JSON(fiber.Map{"checks": results})
	})
}

// Serve listens on port until SIGINT or SIGTERM, then stops accepting
// requests, waits for in-flight ones and runs cleanup in order.
func Serve(app *fiber.App, port string, log *slog.Logger, cleanup ...func(context.Context) error) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	listenErr := make(chan error, 1)
	go func() { listenErr <- app.Listen(":" + port) }()
	log.Info("listening", "port", port)

	select {
	case err := <-listenErr:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error("http shutdown", "err", err)
	}
	for _, fn := range cleanup {
		if err := fn(shutdownCtx); err != nil {
			log.Error("shutdown cleanup", "err", err)
		}
	}
	log.Info("stopped")
	return nil
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
