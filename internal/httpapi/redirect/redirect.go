// Package redirect serves short links.
package redirect

import (
	"context"
	"errors"
	"log"

	"shortener/internal/analytics"
	"shortener/internal/httpapi"
	"shortener/internal/ratelimit"
	"shortener/internal/shortlink"

	"github.com/gofiber/fiber/v2"
)

// Handler resolves short codes and records clicks.
type Handler struct {
	Links  *shortlink.Service
	Clicks *analytics.Recorder
}

func (h *Handler) Register(app fiber.Router) {
	app.Get("/:code", ratelimit.RedirectByIP(), h.redirect)
}

func (h *Handler) redirect(ctx *fiber.Ctx) error {
	code := ctx.Params("code")
	target, err := h.Links.Resolve(ctx.UserContext(), code)
	if errors.Is(err, shortlink.ErrNotFound) {
		return httpapi.Error(ctx, fiber.StatusNotFound, "URL not found")
	}
	if err != nil {
		log.Printf("resolving %s: %v", code, err)
		return httpapi.Error(ctx, fiber.StatusInternalServerError, "Failed to resolve URL")
	}
	click := analytics.Click{
		ShortCode: code,
		Agent:     ctx.Get(fiber.HeaderUserAgent),
		IPAddress: ctx.IP(),
	}
	go h.record(click)
	return ctx.Redirect(target, fiber.StatusFound)
}

// record looks up the link's owner and stores the click. It runs after the
// response has been sent.
func (h *Handler) record(click analytics.Click) {
	ctx := context.Background()
	link, err := h.Links.ByCode(ctx, click.ShortCode)
	if err != nil {
		log.Printf("recording click on %s: %v", click.ShortCode, err)
		return
	}
	click.ShortURLID = link.ID
	click.CreatedBy = link.CreatedBy
	if err := h.Clicks.Record(ctx, click); err != nil {
		log.Printf("recording click on %s: %v", click.ShortCode, err)
	}
}
