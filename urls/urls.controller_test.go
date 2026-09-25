package urls

import (
	"net/http/httptest"
	"shortener/configs"
	"shortener/models"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

// testApp serves the URL handlers with an authenticated user but no auth
// middleware or database, for checks that fail before any query runs.
func testApp() *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user", models.UserDto{Id: 1})
		return c.Next()
	})
	app.Get("/urls", getAllUrlDetails)
	app.Get("/urls/:id", getUrlDetails)
	app.Put("/urls/:id", updateUrl)
	app.Delete("/urls/:id", deleteUrl)
	return app
}

func TestRequestValidation(t *testing.T) {
	app := testApp()
	cases := []struct{ method, path string }{
		{"GET", "/urls/abc"},
		{"GET", "/urls/0"},
		{"GET", "/urls/-1"},
		{"GET", "/urls/18446744073709551616"}, // overflows uint64
		{"PUT", "/urls/-5"},
		{"DELETE", "/urls/1.5"},
		{"GET", "/urls?limit=0"},
		{"GET", "/urls?limit=101"},
		{"GET", "/urls?limit=abc"},
		{"GET", "/urls?cursor=0"},
		{"GET", "/urls?cursor=-3"},
		{"GET", "/urls?cursor=abc"},
	}
	for _, c := range cases {
		resp, err := app.Test(httptest.NewRequest(c.method, c.path, nil))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusBadRequest {
			t.Errorf("%s %s: got %d, want 400", c.method, c.path, resp.StatusCode)
		}
	}
}

func TestToDto(t *testing.T) {
	configs.AppConfig.ApiUrl = "https://tidylnk.com"
	ist := time.FixedZone("IST", 5*3600+1800)
	dto := toDto(models.ShortenedURL{
		Id:        7,
		CreatedBy: 3,
		LongURL:   "https://example.com",
		ShortCode: "abc1234",
		CreatedAt: time.Date(2026, 9, 26, 10, 0, 0, 0, ist),
		UpdatedAt: time.Date(2026, 9, 26, 11, 30, 0, 0, ist),
	})
	want := models.UrlDto{
		Id:        7,
		URL:       "https://example.com",
		ShortCode: "abc1234",
		ShortUrl:  "https://tidylnk.com/abc1234",
		CreatedAt: "2026-09-26T04:30:00Z",
		UpdatedAt: "2026-09-26T06:00:00Z",
		CreatedBy: 3,
	}
	if dto != want {
		t.Fatalf("got %+v\nwant %+v", dto, want)
	}
}
