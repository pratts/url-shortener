// Package redirect serves short links.
package redirect

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"shortener/internal/analytics"
	"shortener/internal/httpapi"
	"shortener/internal/ratelimit"
	"shortener/internal/shortlink"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
)

// maxAgentLength caps the stored User-Agent; headers can be several KB.
const maxAgentLength = 512

// Resolver looks up where a short code points.
type Resolver interface {
	Resolve(ctx context.Context, code string) (shortlink.Entry, error)
}

// ClickRecorder queues a click without blocking.
type ClickRecorder interface {
	Record(analytics.Click) bool
}

// Handler resolves short codes and records clicks.
type Handler struct {
	Links  Resolver
	Clicks ClickRecorder
}

func (h *Handler) Register(app fiber.Router) {
	app.Get("/:code", ratelimit.RedirectByIP(), h.redirect)
}

func (h *Handler) redirect(ctx *fiber.Ctx) error {
	code := ctx.Params("code")
	entry, err := h.Links.Resolve(ctx.UserContext(), code)
	if errors.Is(err, shortlink.ErrNotFound) {
		return httpapi.Error(ctx, fiber.StatusNotFound, "URL not found")
	}
	if err != nil {
		slog.ErrorContext(ctx.UserContext(), "resolving short code", "code", code, "err", err)
		return httpapi.Error(ctx, fiber.StatusInternalServerError, "Failed to resolve URL")
	}

	// Fiber's request strings point into buffers reused after the handler
	// returns, and the click is written later, so copy them.
	h.Clicks.Record(analytics.Click{
		ShortURLID: entry.LinkID,
		CreatedBy:  entry.Owner,
		ShortCode:  utils.CopyString(code),
		Agent:      cleanAgent(ctx.Get(fiber.HeaderUserAgent)),
		IPAddress:  utils.CopyString(ctx.IP()),
	})
	return ctx.Redirect(entry.Target, fiber.StatusFound)
}

// cleanAgent returns a copy of a User-Agent that Postgres will accept: valid
// UTF-8, no NUL bytes, at most maxAgentLength bytes. Clicks are inserted in
// batches, so one bad value would otherwise lose the whole batch.
func cleanAgent(agent string) string {
	agent = strings.ToValidUTF8(agent, "")
	agent = strings.ReplaceAll(agent, "\x00", "")
	if len(agent) > maxAgentLength {
		agent = strings.ToValidUTF8(agent[:maxAgentLength], "")
	}
	return utils.CopyString(agent)
}
