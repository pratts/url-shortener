package auth

import (
	"net/http/httptest"
	"shortener/configs"
	"shortener/models"
	"strconv"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

const testKey = "0123456789abcdef0123456789abcdef"

func setup(t *testing.T) {
	t.Helper()
	configs.AppConfig.JwtSigningKey = testKey
	configs.AppConfig.JwtExpiryTimeHours = 1
	InitTokenParams()
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
	return jwt.MapClaims{
		"iss": tokenIssuer,
		"sub": "42",
		"iat": now.Unix(),
		"exp": now.Add(time.Hour).Unix(),
	}
}

func TestInitTokenParamsRejectsWeakKey(t *testing.T) {
	for _, key := range []string{"", "short-key"} {
		configs.AppConfig.JwtSigningKey = key
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("expected panic for signing key of length %d", len(key))
				}
			}()
			InitTokenParams()
		}()
	}
}

func TestRoundTrip(t *testing.T) {
	setup(t)
	token, err := CreateTokenForUser(&models.UserLoginResponseDto{Id: 42})
	if err != nil {
		t.Fatal(err)
	}
	user, err := ValidateToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if user.Id != 42 {
		t.Fatalf("got user %d, want 42", user.Id)
	}
}

func TestValidateTokenRejects(t *testing.T) {
	setup(t)

	noExp := validClaims()
	delete(noExp, "exp")
	expired := validClaims()
	expired["exp"] = time.Now().Add(-time.Minute).Unix()
	wrongIss := validClaims()
	wrongIss["iss"] = "someone-else"
	noIss := validClaims()
	delete(noIss, "iss")
	badSub := validClaims()
	badSub["sub"] = "-1"
	zeroSub := validClaims()
	zeroSub["sub"] = "0"

	cases := map[string]string{
		"alg none":       sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims()),
		"HS512":          sign(t, jwt.SigningMethodHS512, []byte(testKey), validClaims()),
		"wrong key":      sign(t, jwt.SigningMethodHS256, []byte("another-key-another-key-another-k"), validClaims()),
		"empty key":      sign(t, jwt.SigningMethodHS256, []byte(""), validClaims()),
		"missing exp":    sign(t, jwt.SigningMethodHS256, []byte(testKey), noExp),
		"expired":        sign(t, jwt.SigningMethodHS256, []byte(testKey), expired),
		"wrong issuer":   sign(t, jwt.SigningMethodHS256, []byte(testKey), wrongIss),
		"missing issuer": sign(t, jwt.SigningMethodHS256, []byte(testKey), noIss),
		"negative sub":   sign(t, jwt.SigningMethodHS256, []byte(testKey), badSub),
		"zero sub":       sign(t, jwt.SigningMethodHS256, []byte(testKey), zeroSub),
		"garbage":        "not-a-token",
	}
	for name, token := range cases {
		if _, err := ValidateToken(token); err == nil {
			t.Errorf("%s: expected token to be rejected", name)
		}
	}
}

func TestValidateAuthHeader(t *testing.T) {
	setup(t)
	token, _ := CreateTokenForUser(&models.UserLoginResponseDto{Id: 7})

	app := fiber.New()
	app.Get("/", ValidateAuthHeader, func(c *fiber.Ctx) error {
		return c.SendString(strconv.FormatUint(c.Locals("user").(models.UserDto).Id, 10))
	})

	cases := map[string]int{
		"":                   fiber.StatusUnauthorized,
		token:                fiber.StatusUnauthorized,
		"Basic " + token:     fiber.StatusUnauthorized,
		"Bearer ":            fiber.StatusUnauthorized,
		"Bearer " + token:    fiber.StatusOK,
		"bearer " + token:    fiber.StatusOK,
		"Bearer not-a-token": fiber.StatusUnauthorized,
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
