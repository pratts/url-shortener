package redirect

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"

	"github.com/gofiber/fiber/v2"
)

// homePage is the static page served at "/". It has no scripts; its only
// style is the inline <style> block, which homeCSP allows by hash.
//
//go:embed home.html
var homePage []byte

// robotsTxt lets crawlers index the homepage but keeps them off short links,
// which would otherwise be followed (and counted as clicks).
const robotsTxt = "User-agent: *\nAllow: /$\nDisallow: /\n"

var homeCSP = buildHomeCSP(homePage)

// buildHomeCSP returns a policy that allows nothing except the page's inline
// <style> block (by its SHA-256) and the inline SVG favicon. It panics if the
// page has no style block, which a test would catch first.
func buildHomeCSP(page []byte) string {
	start := bytes.Index(page, []byte("<style>"))
	end := bytes.Index(page, []byte("</style>"))
	if start < 0 || end < start {
		panic("home.html must contain one <style> block")
	}
	sum := sha256.Sum256(page[start+len("<style>") : end])
	return "default-src 'none'; style-src 'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'; " +
		"img-src data:; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"
}

func (h *Handler) home(ctx *fiber.Ctx) error {
	ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	ctx.Set(fiber.HeaderContentSecurityPolicy, homeCSP)
	ctx.Set(fiber.HeaderXContentTypeOptions, "nosniff")
	ctx.Set(fiber.HeaderReferrerPolicy, "strict-origin-when-cross-origin")
	ctx.Set(fiber.HeaderCacheControl, "public, max-age=3600")
	return ctx.Send(homePage)
}

func (h *Handler) robots(ctx *fiber.Ctx) error {
	ctx.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
	ctx.Set(fiber.HeaderCacheControl, "public, max-age=86400")
	return ctx.SendString(robotsTxt)
}
