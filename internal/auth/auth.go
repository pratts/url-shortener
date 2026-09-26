// Package auth issues and verifies access tokens.
package auth

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

const (
	issuer       = "url-shortener-admin"
	minKeyLength = 32
)

var ErrWeakKey = errors.New("JWT signing key must be at least 32 bytes")

// TokenService signs and verifies HS256 access tokens.
type TokenService struct {
	key    []byte
	ttl    time.Duration
	parser *jwt.Parser
	now    func() time.Time
}

func NewTokenService(key string, ttl time.Duration) (*TokenService, error) {
	if len(key) < minKeyLength {
		return nil, ErrWeakKey
	}
	return &TokenService{
		key: []byte(key),
		ttl: ttl,
		parser: jwt.NewParser(
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithExpirationRequired(),
			jwt.WithIssuedAt(),
			jwt.WithIssuer(issuer),
		),
		now: time.Now,
	}, nil
}

// Issue returns a signed token for userID.
func (s *TokenService) Issue(userID uint64) (string, error) {
	now := s.now()
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": issuer,
		"sub": strconv.FormatUint(userID, 10),
		"iat": now.Unix(),
		"exp": now.Add(s.ttl).Unix(),
	}).SignedString(s.key)
}

// Verify checks token and returns the user ID it was issued for.
func (s *TokenService) Verify(token string) (uint64, error) {
	claims := jwt.MapClaims{}
	if _, err := s.parser.ParseWithClaims(token, claims, func(*jwt.Token) (interface{}, error) {
		return s.key, nil
	}); err != nil {
		return 0, err
	}
	sub, err := claims.GetSubject()
	if err != nil {
		return 0, err
	}
	userID, err := strconv.ParseUint(sub, 10, 64)
	if err != nil || userID == 0 {
		return 0, jwt.ErrTokenInvalidSubject
	}
	return userID, nil
}

type userIDKey struct{}

// Middleware rejects requests without a valid Bearer token and stores the
// caller's user ID for UserID.
func (s *TokenService) Middleware() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		header := ctx.Get(fiber.HeaderAuthorization)
		if header == "" {
			return unauthorized(ctx, "Authorization header is missing")
		}
		scheme, token, found := strings.Cut(header, " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
			return unauthorized(ctx, "Authorization header must use the Bearer scheme")
		}
		userID, err := s.Verify(token)
		if err != nil {
			return unauthorized(ctx, "Invalid token")
		}
		ctx.Locals(userIDKey{}, userID)
		return ctx.Next()
	}
}

// UserID returns the authenticated user's ID; ok is false when the request did
// not pass through Middleware.
func UserID(ctx *fiber.Ctx) (id uint64, ok bool) {
	id, ok = ctx.Locals(userIDKey{}).(uint64)
	return id, ok
}

func unauthorized(ctx *fiber.Ctx, msg string) error {
	return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": msg})
}
