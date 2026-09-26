package auth

import (
	"errors"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

const testKey = "0123456789abcdef0123456789abcdef"

func newService(t *testing.T) *TokenService {
	t.Helper()
	s, err := NewTokenService(testKey, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func sign(t *testing.T, method jwt.SigningMethod, key interface{}, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func validClaims() jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{"iss": issuer, "sub": "42", "iat": now.Unix(), "exp": now.Add(time.Hour).Unix()}
}

func TestNewTokenServiceRejectsWeakKey(t *testing.T) {
	for _, key := range []string{"", "short-key"} {
		if _, err := NewTokenService(key, time.Hour); !errors.Is(err, ErrWeakKey) {
			t.Errorf("key of length %d: got %v, want ErrWeakKey", len(key), err)
		}
	}
}

func TestIssueAndVerify(t *testing.T) {
	s := newService(t)
	token, err := s.Issue(42)
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.Verify(token)
	if err != nil || id != 42 {
		t.Fatalf("got %d, %v; want 42", id, err)
	}
}

func TestTokenExpiresAfterTTL(t *testing.T) {
	s := newService(t)
	s.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	token, _ := s.Issue(42)
	if _, err := s.Verify(token); !errors.Is(err, jwt.ErrTokenExpired) {
		t.Fatalf("got %v, want ErrTokenExpired", err)
	}
}

func TestVerifyRejects(t *testing.T) {
	s := newService(t)
	edit := func(f func(jwt.MapClaims)) jwt.MapClaims { c := validClaims(); f(c); return c }
	cases := map[string]string{
		"alg none":       sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims()),
		"HS512":          sign(t, jwt.SigningMethodHS512, []byte(testKey), validClaims()),
		"wrong key":      sign(t, jwt.SigningMethodHS256, []byte("another-key-another-key-another-k"), validClaims()),
		"empty key":      sign(t, jwt.SigningMethodHS256, []byte(""), validClaims()),
		"missing exp":    sign(t, jwt.SigningMethodHS256, []byte(testKey), edit(func(c jwt.MapClaims) { delete(c, "exp") })),
		"wrong issuer":   sign(t, jwt.SigningMethodHS256, []byte(testKey), edit(func(c jwt.MapClaims) { c["iss"] = "someone-else" })),
		"missing issuer": sign(t, jwt.SigningMethodHS256, []byte(testKey), edit(func(c jwt.MapClaims) { delete(c, "iss") })),
		"negative sub":   sign(t, jwt.SigningMethodHS256, []byte(testKey), edit(func(c jwt.MapClaims) { c["sub"] = "-1" })),
		"zero sub":       sign(t, jwt.SigningMethodHS256, []byte(testKey), edit(func(c jwt.MapClaims) { c["sub"] = "0" })),
		"garbage":        "not-a-token",
	}
	for name, token := range cases {
		if _, err := s.Verify(token); err == nil {
			t.Errorf("%s: expected token to be rejected", name)
		}
	}
}

func TestMiddleware(t *testing.T) {
	s := newService(t)
	token, _ := s.Issue(7)
	app := fiber.New()
	app.Get("/", s.Middleware(), func(c *fiber.Ctx) error {
		id, ok := UserID(c)
		if !ok {
			return c.SendStatus(fiber.StatusTeapot)
		}
		return c.SendString(strconv.FormatUint(id, 10))
	})

	cases := map[string]int{
		"":                   fiber.StatusUnauthorized,
		token:                fiber.StatusUnauthorized,
		"Basic " + token:     fiber.StatusUnauthorized,
		"Bearer ":            fiber.StatusUnauthorized,
		"Bearer not-a-token": fiber.StatusUnauthorized,
		"Bearer " + token:    fiber.StatusOK,
		"bearer " + token:    fiber.StatusOK,
	}
	for header, want := range cases {
		req := httptest.NewRequest("GET", "/", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != want {
			t.Errorf("header %q: got %d, want %d", header, resp.StatusCode, want)
		}
	}
}

func TestUserIDWithoutMiddleware(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		if _, ok := UserID(c); ok {
			return c.SendStatus(fiber.StatusOK)
		}
		return c.SendStatus(fiber.StatusUnauthorized)
	})
	resp, _ := app.Test(httptest.NewRequest("GET", "/", nil))
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("UserID reported a user without the middleware")
	}
}
