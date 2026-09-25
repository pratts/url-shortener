package ratelimit

import (
	"fmt"
	"shortener/cache"
	"shortener/models"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func limitReached(ctx *fiber.Ctx) error {
	return ctx.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
		"error": "Too many requests, please try again later",
	})
}

// LoginByIP caps all login attempts from one client IP.
func LoginByIP() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        20,
		Expiration: 15 * time.Minute,
		KeyGenerator: func(ctx *fiber.Ctx) string {
			return "login:ip:" + ctx.IP()
		},
		LimitReached:      limitReached,
		Storage:           cache.NewLimiterStorage("rl:"),
		LimiterMiddleware: limiter.SlidingWindow{},
	})
}

// LoginByAccount caps failed login attempts per email, regardless of source IP.
func LoginByAccount() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        5,
		Expiration: 15 * time.Minute,
		KeyGenerator: func(ctx *fiber.Ctx) string {
			var body models.UserLoginDto
			_ = ctx.BodyParser(&body)
			return "login:account:" + strings.ToLower(strings.TrimSpace(body.Email))
		},
		SkipSuccessfulRequests: true,
		LimitReached:           limitReached,
		Storage:                cache.NewLimiterStorage("rl:"),
		LimiterMiddleware:      limiter.SlidingWindow{},
	})
}

// PerUser caps requests per authenticated user. It must run after
// auth.ValidateAuthHeader.
func PerUser(name string, max int, window time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: window,
		KeyGenerator: func(ctx *fiber.Ctx) string {
			user, _ := ctx.Locals("user").(models.UserDto)
			return fmt.Sprintf("%s:user:%d", name, user.Id)
		},
		LimitReached:      limitReached,
		Storage:           cache.NewLimiterStorage("rl:"),
		LimiterMiddleware: limiter.SlidingWindow{},
	})
}

// RedirectByIP caps redirects per client IP. It keeps counters in memory
// rather than Redis so the redirect hot path does not pay two extra Redis
// round trips per request; limits therefore apply per instance.
func RedirectByIP() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:          120,
		Expiration:   time.Minute,
		KeyGenerator: func(ctx *fiber.Ctx) string { return ctx.IP() },
		LimitReached: limitReached,
	})
}
