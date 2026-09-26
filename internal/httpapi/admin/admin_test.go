package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"shortener/internal/auth"
	"shortener/internal/ratelimit"
	"shortener/internal/shortlink"
	"shortener/internal/user"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

type testServer struct {
	t   *testing.T
	app *fiber.App
}

func newTestServer(t *testing.T, links shortlink.Repository, registration bool) *testServer {
	t.Helper()
	linkSvc, err := shortlink.NewService(links, shortlink.NewMemoryCache(), "https://tidylnk.com")
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := auth.NewTokenService(strings.Repeat("k", 32), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{
		Links:               linkSvc,
		Users:               user.NewService(user.NewMemoryRepository(), user.WithPasswordCost(bcrypt.MinCost)),
		Tokens:              tokens,
		Limits:              ratelimit.New(nil), // nil storage: in-memory counters
		RegistrationEnabled: registration,
	}
	app := fiber.New()
	h.Register(app.Group("/api/v1"))
	return &testServer{t: t, app: app}
}

// do sends a request and returns the status and decoded JSON body.
func (s *testServer) do(method, path, token string, body interface{}) (int, map[string]interface{}) {
	s.t.Helper()
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := s.app.Test(req, -1)
	if err != nil {
		s.t.Fatal(err)
	}
	var out map[string]interface{}
	raw, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out
}

// signup registers and logs in, returning a token.
func (s *testServer) signup(email string) string {
	s.t.Helper()
	if status, body := s.do("POST", "/api/v1/users/register", "", map[string]string{
		"email": email, "name": "Test", "password": "correct horse",
	}); status != fiber.StatusCreated {
		s.t.Fatalf("register %s: %d %v", email, status, body)
	}
	status, body := s.do("POST", "/api/v1/users/login", "", map[string]string{
		"email": email, "password": "correct horse",
	})
	if status != fiber.StatusOK {
		s.t.Fatalf("login %s: %d %v", email, status, body)
	}
	return body["token"].(string)
}

func TestUserEndpoints(t *testing.T) {
	s := newTestServer(t, shortlink.NewMemoryRepository(), true)
	token := s.signup("alice@example.com")

	checks := []struct {
		name, method, path, token string
		body                      interface{}
		want                      int
	}{
		{"duplicate email", "POST", "/api/v1/users/register", "", map[string]string{"email": "ALICE@example.com", "name": "A", "password": "correct horse"}, 409},
		{"invalid registration", "POST", "/api/v1/users/register", "", map[string]string{}, 400},
		{"wrong password", "POST", "/api/v1/users/login", "", map[string]string{"email": "alice@example.com", "password": "nope nope"}, 401},
		{"missing credentials", "POST", "/api/v1/users/login", "", map[string]string{}, 400},
		{"me without token", "GET", "/api/v1/users/me", "", nil, 401},
		{"me", "GET", "/api/v1/users/me", token, nil, 200},
		{"empty update", "PATCH", "/api/v1/users/me", token, map[string]string{}, 400},
		{"password without current", "PATCH", "/api/v1/users/me", token, map[string]string{"password": "new password"}, 400},
		{"wrong current password", "PATCH", "/api/v1/users/me", token, map[string]string{"password": "new password", "current_password": "nope nope"}, 403},
		{"rename", "PATCH", "/api/v1/users/me", token, map[string]string{"name": "Alice"}, 200},
	}
	for _, c := range checks {
		if status, body := s.do(c.method, c.path, c.token, c.body); status != c.want {
			t.Errorf("%s: got %d %v, want %d", c.name, status, body, c.want)
		}
	}

	_, body := s.do("POST", "/api/v1/users/register", "", map[string]string{})
	if fields, _ := body["fields"].(map[string]interface{}); len(fields) != 3 {
		t.Errorf("registration errors should list 3 fields, got %v", body)
	}
}

func TestRegistrationDisabled(t *testing.T) {
	s := newTestServer(t, shortlink.NewMemoryRepository(), false)
	if status, _ := s.do("POST", "/api/v1/users/register", "", map[string]string{}); status != fiber.StatusNotFound && status != fiber.StatusMethodNotAllowed {
		t.Fatalf("register with registration disabled: got %d", status)
	}
}

func TestLinkEndpoints(t *testing.T) {
	s := newTestServer(t, shortlink.NewMemoryRepository(), true)
	alice := s.signup("alice@example.com")
	bob := s.signup("bob@example.com")

	status, created := s.do("POST", "/api/v1/urls", alice, map[string]string{"url": "https://example.com"})
	if status != fiber.StatusCreated || created["short_url"] == "" || created["created_at"] == "" {
		t.Fatalf("create: %d %v", status, created)
	}
	id := fmt.Sprint(created["id"])

	checks := []struct {
		name, method, path, token string
		body                      interface{}
		want                      int
	}{
		{"create without token", "POST", "/api/v1/urls", "", map[string]string{"url": "https://example.com"}, 401},
		{"create unsafe target", "POST", "/api/v1/urls", alice, map[string]string{"url": "javascript:alert(1)"}, 400},
		{"create own host", "POST", "/api/v1/urls", alice, map[string]string{"url": "https://tidylnk.com/x"}, 400},
		{"get", "GET", "/api/v1/urls/" + id, alice, nil, 200},
		{"get other user's", "GET", "/api/v1/urls/" + id, bob, nil, 404},
		{"update other user's", "PUT", "/api/v1/urls/" + id, bob, map[string]string{"url": "https://evil.example"}, 404},
		{"delete other user's", "DELETE", "/api/v1/urls/" + id, bob, nil, 404},
		{"get unknown", "GET", "/api/v1/urls/999", alice, nil, 404},
		{"bad id", "GET", "/api/v1/urls/abc", alice, nil, 400},
		{"zero id", "GET", "/api/v1/urls/0", alice, nil, 400},
		{"negative id", "DELETE", "/api/v1/urls/-1", alice, nil, 400},
		{"overflowing id", "GET", "/api/v1/urls/18446744073709551616", alice, nil, 400},
		{"limit too big", "GET", "/api/v1/urls?limit=101", alice, nil, 400},
		{"limit zero", "GET", "/api/v1/urls?limit=0", alice, nil, 400},
		{"bad cursor", "GET", "/api/v1/urls?cursor=abc", alice, nil, 400},
		{"update", "PUT", "/api/v1/urls/" + id, alice, map[string]string{"url": "https://example.org"}, 200},
		{"update unsafe", "PUT", "/api/v1/urls/" + id, alice, map[string]string{"url": "data:text/html,x"}, 400},
		{"delete", "DELETE", "/api/v1/urls/" + id, alice, nil, 204},
		{"delete again", "DELETE", "/api/v1/urls/" + id, alice, nil, 404},
	}
	for _, c := range checks {
		if status, body := s.do(c.method, c.path, c.token, c.body); status != c.want {
			t.Errorf("%s: got %d %v, want %d", c.name, status, body, c.want)
		}
	}
}

func TestListPaginationOverHTTP(t *testing.T) {
	s := newTestServer(t, shortlink.NewMemoryRepository(), true)
	token := s.signup("alice@example.com")
	for i := 0; i < 5; i++ {
		s.do("POST", "/api/v1/urls", token, map[string]string{"url": fmt.Sprintf("https://example.com/%d", i)})
	}

	var ids []interface{}
	path := "/api/v1/urls?limit=2"
	for pages := 0; ; pages++ {
		if pages > 3 {
			t.Fatal("pagination did not terminate")
		}
		status, page := s.do("GET", path, token, nil)
		if status != fiber.StatusOK {
			t.Fatalf("list: %d %v", status, page)
		}
		for _, item := range page["items"].([]interface{}) {
			ids = append(ids, item.(map[string]interface{})["id"])
		}
		next, ok := page["next_cursor"].(string)
		if !ok {
			if page["next_cursor"] != nil {
				t.Fatalf("next_cursor should be a string or null, got %v", page["next_cursor"])
			}
			break
		}
		path = "/api/v1/urls?limit=2&cursor=" + next
	}
	if fmt.Sprint(ids) != "[5 4 3 2 1]" {
		t.Fatalf("got ids %v, want [5 4 3 2 1]", ids)
	}
}

// brokenRepo fails every call, as a database outage would.
type brokenRepo struct{ shortlink.Repository }

var errDown = errors.New("connection refused")

func (brokenRepo) Create(context.Context, *shortlink.Link) error { return errDown }
func (brokenRepo) Get(context.Context, uint64, uint64) (shortlink.Link, error) {
	return shortlink.Link{}, errDown
}
func (brokenRepo) List(context.Context, uint64, int, uint64) ([]shortlink.Link, error) {
	return nil, errDown
}
func (brokenRepo) Delete(context.Context, uint64, uint64) (shortlink.Link, error) {
	return shortlink.Link{}, errDown
}

func TestDatabaseErrorsAre500(t *testing.T) {
	s := newTestServer(t, brokenRepo{}, true)
	token := s.signup("alice@example.com")
	for _, c := range []struct{ method, path string }{
		{"POST", "/api/v1/urls"},
		{"GET", "/api/v1/urls"},
		{"GET", "/api/v1/urls/1"},
		{"DELETE", "/api/v1/urls/1"},
	} {
		status, body := s.do(c.method, c.path, token, map[string]string{"url": "https://example.com"})
		if status != fiber.StatusInternalServerError {
			t.Errorf("%s %s with DB down: got %d, want 500", c.method, c.path, status)
		}
		if msg, _ := body["error"].(string); strings.Contains(msg, "connection refused") {
			t.Errorf("%s %s leaked the internal error: %q", c.method, c.path, msg)
		}
	}
}

func TestWithUserRejectsUnauthenticated(t *testing.T) {
	app := fiber.New()
	app.Get("/", withUser(func(ctx *fiber.Ctx, _ uint64) error { return ctx.SendStatus(200) }))
	resp, _ := app.Test(httptest.NewRequest("GET", "/", nil))
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("got %d, want 401", resp.StatusCode)
	}
}
