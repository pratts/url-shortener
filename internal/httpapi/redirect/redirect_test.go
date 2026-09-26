package redirect

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"shortener/internal/analytics"
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
	(&Handler{Links: r, Clicks: clicks}).Register(app)
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
