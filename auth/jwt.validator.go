package auth

import (
	"shortener/configs"
	"shortener/models"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

const (
	tokenIssuer      = "url-shortener-admin"
	minSigningKeyLen = 32
)

var secretKey []byte
var SigningMethod = jwt.SigningMethodHS256
var KeyDuration int

var parser = jwt.NewParser(
	jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	jwt.WithExpirationRequired(),
	jwt.WithIssuedAt(),
	jwt.WithIssuer(tokenIssuer),
)

func InitTokenParams() {
	if len(configs.AppConfig.JwtSigningKey) < minSigningKeyLen {
		panic("JWT_SIGNING_KEY must be set and at least 32 bytes long")
	}
	secretKey = []byte(configs.AppConfig.JwtSigningKey)
	SigningMethod = jwt.SigningMethodHS256
	KeyDuration = configs.AppConfig.JwtExpiryTimeHours
}

func CreateTokenForUser(u *models.UserLoginResponseDto) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(SigningMethod, jwt.MapClaims{
		"iss": tokenIssuer,
		"sub": strconv.FormatUint(u.Id, 10),
		"iat": now.Unix(),
		"exp": now.Add(time.Hour * time.Duration(KeyDuration)).Unix(),
	})
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ValidateToken(token string) (models.UserDto, error) {
	claims := jwt.MapClaims{}
	_, err := parser.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil {
		return models.UserDto{}, err
	}

	sub, err := claims.GetSubject()
	if err != nil {
		return models.UserDto{}, err
	}
	if sub == "" {
		return models.UserDto{}, jwt.ErrTokenInvalidSubject
	}
	userId, err := strconv.ParseUint(sub, 10, 64)
	if err != nil || userId == 0 {
		return models.UserDto{}, jwt.ErrTokenInvalidSubject
	}
	return models.UserDto{Id: userId}, nil
}

func ValidateAuthHeader(ctx *fiber.Ctx) error {
	header := ctx.Get("Authorization")
	if header == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header is missing",
		})
	}

	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authorization header must use the Bearer scheme",
		})
	}
	user, err := ValidateToken(token)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}
	ctx.Locals("user", user)
	return ctx.Next()
}
