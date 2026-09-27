// Package ratelimit provides the request limits used by the HTTP services.
package ratelimit

import (
	"fmt"
	"strings"
	"time"

	"shortener/internal/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// Limits builds limiters whose counters live in a shared storage, so limits
// hold across every instance of a service.
type Limits struct {
	storage  fiber.Storage
	disabled bool
}

// New returns enforcing limits. A nil storage keeps counters in memory.
func New(storage fiber.Storage) *Limits {
	return &Limits{storage: storage}
}

// Disabled returns limits that let every request through. It exists for local
// development and end-to-end tests; configuration refuses it in production.
func Disabled() *Limits {
	return &Limits{disabled: true}
}

// Enabled reports whether the limits are enforced.
func (l *Limits) Enabled() bool {
	return !l.disabled
}

func allow(ctx *fiber.Ctx) error {
	return ctx.Next()
}

// build returns the limiter for cfg, or a pass-through when limits are off.
func (l *Limits) build(cfg limiter.Config) fiber.Handler {
	if l.disabled {
		return allow
	}
	return limiter.New(cfg)
}

func limitReached(ctx *fiber.Ctx) error {
	return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
		"error": "Too many requests, please try again later",
	})
}

func (l *Limits) sliding(max int, window time.Duration, key func(*fiber.Ctx) string) limiter.Config {
	return limiter.Config{
		Max:               max,
		Expiration:        window,
		KeyGenerator:      key,
		LimitReached:      limitReached,
		Storage:           l.storage,
		LimiterMiddleware: limiter.SlidingWindow{},
	}
}

// LoginByIP caps all login attempts from one client IP.
func (l *Limits) LoginByIP() fiber.Handler {
	return l.build(l.sliding(20, 15*time.Minute, func(ctx *fiber.Ctx) string {
		return "login:ip:" + ctx.IP()
	}))
}

// LoginByAccount caps failed login attempts per email, regardless of source IP.
func (l *Limits) LoginByAccount() fiber.Handler {
	cfg := l.sliding(5, 15*time.Minute, func(ctx *fiber.Ctx) string {
		var body struct {
			Email string `json:"email"`
		}
		_ = ctx.BodyParser(&body)
		return "login:account:" + strings.ToLower(strings.TrimSpace(body.Email))
	})
	cfg.SkipSuccessfulRequests = true
	return l.build(cfg)
}

// RegisterByIP caps account registrations from one client IP.
func (l *Limits) RegisterByIP() fiber.Handler {
	return l.build(l.sliding(5, time.Hour, func(ctx *fiber.Ctx) string {
		return "register:ip:" + ctx.IP()
	}))
}

// PerUser caps requests per authenticated user. It must run after the auth
// middleware.
func (l *Limits) PerUser(name string, max int, window time.Duration) fiber.Handler {
	return l.build(l.sliding(max, window, func(ctx *fiber.Ctx) string {
		id, _ := auth.UserID(ctx)
		return fmt.Sprintf("%s:user:%d", name, id)
	}))
}

// RedirectByIP caps redirects per client IP. Counters are kept in memory
// rather than Redis so the redirect hot path does not pay two extra Redis
// round trips per request; the limit therefore applies per instance.
func (l *Limits) RedirectByIP() fiber.Handler {
	return l.build(limiter.Config{
		Max:          120,
		Expiration:   time.Minute,
		KeyGenerator: func(ctx *fiber.Ctx) string { return ctx.IP() },
		LimitReached: limitReached,
	})
}
