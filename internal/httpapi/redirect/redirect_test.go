package redirect

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"shortener/internal/analytics"
	"shortener/internal/ratelimit"
	"shortener/internal/shortlink"

	"github.com/gofiber/fiber/v2"
)

type fakeResolver struct {
	entries map[string]shortlink.Entry
	err     error
}

func (r fakeResolver) Resolve(_ context.Context, code string) (shortlink.Entry, error) {
	if r.err != nil {
		return shortlink.Entry{}, r.err
	}
	e, ok := r.entries[code]
	if !ok {
		return shortlink.Entry{}, shortlink.ErrNotFound
	}
	return e, nil
}

type fakeClicks struct {
	mu     sync.Mutex
	clicks []analytics.Click
}

func (c *fakeClicks) Record(click analytics.Click) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clicks = append(c.clicks, click)
	return true
}

func newApp(r Resolver, clicks ClickRecorder) *fiber.App {
	app := fiber.New()
	(&Handler{Links: r, Clicks: clicks, Limits: ratelimit.New(nil)}).Register(app)
	return app
}

func TestRedirectRecordsClick(t *testing.T) {
	clicks := &fakeClicks{}
	app := newApp(fakeResolver{entries: map[string]shortlink.Entry{
		"CODE001": {LinkID: 7, Owner: 3, Target: "https://example.com/x"},
	}}, clicks)

	req := httptest.NewRequest("GET", "/CODE001", nil)
	req.Header.Set("User-Agent", "test-agent")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusFound || resp.Header.Get("Location") != "https://example.com/x" {
		t.Fatalf("got %d to %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "private, max-age=0" {
		t.Fatalf("Cache-Control %q; redirects must not be cached or clicks go uncounted", cc)
	}
	if len(clicks.clicks) != 1 {
		t.Fatalf("recorded %d clicks, want 1", len(clicks.clicks))
	}
	c := clicks.clicks[0]
	if c.ShortURLID != 7 || c.CreatedBy != 3 || c.ShortCode != "CODE001" || c.Agent != "test-agent" || c.IPAddress == "" {
		t.Fatalf("click %+v", c)
	}
}

func TestRedirectErrors(t *testing.T) {
	clicks := &fakeClicks{}
	notFound := newApp(fakeResolver{}, clicks)
	resp, _ := notFound.Test(httptest.NewRequest("GET", "/NOPE000", nil))
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("unknown code: got %d, want 404", resp.StatusCode)
	}
	broken := newApp(fakeResolver{err: errors.New("connection refused")}, clicks)
	resp, _ = broken.Test(httptest.NewRequest("GET", "/CODE001", nil))
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("resolver failure: got %d, want 500", resp.StatusCode)
	}
	if len(clicks.clicks) != 0 {
		t.Fatalf("failed redirects recorded %d clicks", len(clicks.clicks))
	}
}

func TestCleanAgent(t *testing.T) {
	long := strings.Repeat("é", maxAgentLength) // 2 bytes each
	cases := map[string]string{
		"Mozilla/5.0":            "Mozilla/5.0",
		"bad\xff\xfebytes":       "badbytes",
		"nul\x00byte":            "nulbyte",
		strings.Repeat("a", 600): strings.Repeat("a", maxAgentLength),
	}
	for in, want := range cases {
		if got := cleanAgent(in); got != want {
			t.Errorf("cleanAgent(%q) = %q, want %q", in, got, want)
		}
	}
	got := cleanAgent(long)
	if len(got) > maxAgentLength || !utf8.ValidString(got) {
		t.Errorf("truncating multi-byte text gave %d bytes, valid UTF-8 %v", len(got), utf8.ValidString(got))
	}
}

func TestHomepage(t *testing.T) {
	clicks := &fakeClicks{}
	app := newApp(fakeResolver{}, clicks)
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	page := string(body)

	if resp.StatusCode != fiber.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("got %d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	for _, want := range []string{
		"hobby project",
		`href="https://admin.tidylnk.com"`,
		`href="https://github.com/pratts/url-shortener"`,
		`href="https://github.com/pratts/url-shortener-admin"`,
		`href="https://www.linkedin.com/in/prateeksharma28"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("homepage is missing %q", want)
		}
	}
	if strings.Contains(strings.ToLower(page), "<script") {
		t.Error("homepage must stay static: no scripts")
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options")
	}
	if len(clicks.clicks) != 0 {
		t.Fatalf("visiting the homepage recorded %d clicks", len(clicks.clicks))
	}
}

func TestHomepageCSPAllowsExactlyItsStyle(t *testing.T) {
	start := strings.Index(string(homePage), "<style>") + len("<style>")
	end := strings.Index(string(homePage), "</style>")
	sum := sha256.Sum256(homePage[start:end])
	want := "style-src 'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"

	app := newApp(fakeResolver{}, &fakeClicks{})
	resp, _ := app.Test(httptest.NewRequest("GET", "/", nil))
	csp := resp.Header.Get("Content-Security-Policy")
	for _, part := range []string{want, "default-src 'none'", "frame-ancestors 'none'"} {
		if !strings.Contains(csp, part) {
			t.Errorf("CSP %q is missing %q", csp, part)
		}
	}
	if strings.Contains(csp, "unsafe-inline") || strings.Contains(csp, "script-src") {
		t.Errorf("CSP must not allow inline code or scripts: %q", csp)
	}
	if strings.Count(string(homePage), "<style>") != 1 {
		t.Error("the CSP hash covers exactly one <style> block")
	}
}

func TestRobotsTxt(t *testing.T) {
	app := newApp(fakeResolver{}, &fakeClicks{})
	resp, _ := app.Test(httptest.NewRequest("GET", "/robots.txt", nil))
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != fiber.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/plain") {
		t.Fatalf("got %d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if string(body) != "User-agent: *\nAllow: /$\nDisallow: /\n" {
		t.Fatalf("robots.txt = %q", body)
	}
}

func TestHomepageIsNotRateLimited(t *testing.T) {
	app := newApp(fakeResolver{}, &fakeClicks{})
	for i := 0; i < 130; i++ {
		resp, _ := app.Test(httptest.NewRequest("GET", "/", nil))
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("request %d got %d; the redirect limit should not apply to the homepage", i+1, resp.StatusCode)
		}
	}
}
