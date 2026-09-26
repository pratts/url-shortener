package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"shortener/internal/config"
	"shortener/internal/platform"

	"github.com/gofiber/fiber/v2"
)

func testApp(t *testing.T, logs io.Writer) *fiber.App {
	t.Helper()
	return NewApp(config.HTTP{RequestTimeout: 50 * time.Millisecond}, platform.NewLogger("json", logs))
}

func decode(t *testing.T, body io.Reader) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestHealthAndReadiness(t *testing.T) {
	var logs bytes.Buffer
	app := testApp(t, &logs)
	healthy := true
	RegisterHealth(app, map[string]Check{
		"postgres": func(context.Context) error { return nil },
		"redis": func(context.Context) error {
			if healthy {
				return nil
			}
			return errors.New("connection refused")
		},
	})

	resp, _ := app.Test(httptest.NewRequest("GET", "/healthz", nil))
	if resp.StatusCode != 200 {
		t.Fatalf("healthz: %d", resp.StatusCode)
	}
	resp, _ = app.Test(httptest.NewRequest("GET", "/readyz", nil))
	if resp.StatusCode != 200 {
		t.Fatalf("readyz with healthy deps: %d", resp.StatusCode)
	}
	healthy = false
	resp, _ = app.Test(httptest.NewRequest("GET", "/readyz", nil))
	body := decode(t, resp.Body)
	checks := body["checks"].(map[string]interface{})
	if resp.StatusCode != fiber.StatusServiceUnavailable || checks["redis"] != "connection refused" || checks["postgres"] != "ok" {
		t.Fatalf("readyz with redis down: %d %v", resp.StatusCode, body)
	}
}

func TestPanicBecomesJSON500(t *testing.T) {
	var logs bytes.Buffer
	app := testApp(t, &logs)
	app.Get("/boom", func(*fiber.Ctx) error { panic("secret internal detail") })
	resp, _ := app.Test(httptest.NewRequest("GET", "/boom", nil))
	body := decode(t, resp.Body)
	if resp.StatusCode != 500 || body["error"] != "Internal server error" {
		t.Fatalf("got %d %v", resp.StatusCode, body)
	}
	if !strings.Contains(logs.String(), "secret internal detail") {
		t.Fatal("the panic was not logged")
	}
}

func TestRequestDeadlineAndAccessLog(t *testing.T) {
	var logs bytes.Buffer
	app := testApp(t, &logs)
	app.Get("/slow", func(c *fiber.Ctx) error {
		deadline, ok := c.UserContext().Deadline()
		if !ok || time.Until(deadline) > 50*time.Millisecond {
			return c.SendStatus(fiber.StatusTeapot)
		}
		<-c.UserContext().Done()
		return Error(c, fiber.StatusGatewayTimeout, "timed out")
	})
	resp, _ := app.Test(httptest.NewRequest("GET", "/slow", nil), 2000)
	if resp.StatusCode != fiber.StatusGatewayTimeout {
		t.Fatalf("got %d; the request context has no deadline", resp.StatusCode)
	}
	requestID := resp.Header.Get(fiber.HeaderXRequestID)
	if requestID == "" {
		t.Fatal("no request ID header")
	}

	var entry map[string]interface{}
	for _, line := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
		json.Unmarshal([]byte(line), &entry)
		if entry["msg"] == "request" {
			break
		}
	}
	if entry["path"] != "/slow" || entry["status"] != float64(504) || entry["request_id"] != requestID || entry["level"] != "ERROR" {
		t.Fatalf("access log entry %v", entry)
	}
}
