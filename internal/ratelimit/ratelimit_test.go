package ratelimit

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

// statuses sends n requests and returns their status codes.
func statuses(t *testing.T, app *fiber.App, method, path, body string, n int) []int {
	t.Helper()
	out := make([]int, n)
	for i := range out {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatal(err)
		}
		out[i] = resp.StatusCode
	}
	return out
}

func count(codes []int, status int) int {
	n := 0
	for _, c := range codes {
		if c == status {
			n++
		}
	}
	return n
}

func TestEnabledLimitsBlockAfterMax(t *testing.T) {
	l := New(nil)
	if !l.Enabled() {
		t.Fatal("New should enforce limits")
	}
	app := fiber.New()
	app.Get("/", l.PerUser("test", 3, time.Minute), func(c *fiber.Ctx) error { return c.SendStatus(200) })

	codes := statuses(t, app, "GET", "/", "", 5)
	if count(codes, 200) != 3 || count(codes, fiber.StatusTooManyRequests) != 2 {
		t.Fatalf("got %v, want 3 allowed then 429s", codes)
	}
}

func TestDisabledLimitsLetEverythingThrough(t *testing.T) {
	l := Disabled()
	if l.Enabled() {
		t.Fatal("Disabled should not enforce limits")
	}
	app := fiber.New()
	ok := func(c *fiber.Ctx) error { return c.SendStatus(200) }
	app.Post("/login", l.LoginByIP(), l.LoginByAccount(), ok)
	app.Post("/register", l.RegisterByIP(), ok)
	app.Post("/urls", l.PerUser("create-url", 30, time.Minute), ok)
	app.Get("/:code", l.RedirectByIP(), ok)

	for _, c := range []struct {
		method, path string
		n            int
	}{
		{"POST", "/login", 30},
		{"POST", "/register", 10},
		{"POST", "/urls", 40},
		{"GET", "/abc1234", 150},
	} {
		codes := statuses(t, app, c.method, c.path, `{"email":"a@b.c"}`, c.n)
		if count(codes, 200) != c.n {
			t.Errorf("%s %s: got %d of %d allowed with limits off", c.method, c.path, count(codes, 200), c.n)
		}
	}
}

func TestLoginByAccountCountsOnlyFailures(t *testing.T) {
	app := fiber.New()
	app.Post("/login", New(nil).LoginByAccount(), func(c *fiber.Ctx) error {
		if strings.Contains(string(c.Body()), `"good"`) {
			return c.SendStatus(200)
		}
		return c.SendStatus(fiber.StatusUnauthorized)
	})

	good := statuses(t, app, "POST", "/login", `{"email":"a@b.c","password":"good"}`, 10)
	if count(good, 200) != 10 {
		t.Fatalf("successful logins were limited: %v", good)
	}
	bad := statuses(t, app, "POST", "/login", `{"email":"A@b.c ","password":"bad"}`, 6)
	if count(bad, fiber.StatusUnauthorized) != 5 || bad[5] != fiber.StatusTooManyRequests {
		t.Fatalf("got %v, want 5 failures then 429 (email normalized)", bad)
	}
	other := statuses(t, app, "POST", "/login", `{"email":"other@b.c","password":"bad"}`, 1)
	if other[0] != fiber.StatusUnauthorized {
		t.Fatalf("another account was blocked: %v", other)
	}
}

func TestRedirectByIPLimit(t *testing.T) {
	app := fiber.New()
	app.Get("/:code", New(nil).RedirectByIP(), func(c *fiber.Ctx) error { return c.SendStatus(302) })
	codes := statuses(t, app, "GET", "/abc1234", "", 121)
	if count(codes, 302) != 120 || codes[120] != fiber.StatusTooManyRequests {
		t.Fatalf("got %d redirects and final status %d, want 120 then 429", count(codes, 302), codes[120])
	}
}
