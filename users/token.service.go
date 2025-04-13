package users

import (
	"shortener/configs"
	"shortener/models"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey []byte
var SigningMethod = jwt.SigningMethodHS256
var KeyDuration int

func InitTokenParams() {
	secretKey = []byte(configs.AppConfig.JwtSigningKey)
	SigningMethod = jwt.SigningMethodHS256
	KeyDuration = configs.AppConfig.JwtExpiryTimeHours
}

func CreateTokenForUser(u *models.UserLoginResponseDto) (string, error) {
	token := jwt.NewWithClaims(SigningMethod, jwt.MapClaims{
		"sub": strconv.Itoa(int(u.Id)),
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour * time.Duration(KeyDuration)).Unix(),
	})
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ValidateToken(token string) (models.UserDto, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
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
	userId, err := strconv.Atoi(sub)
	if err != nil {
		return models.UserDto{}, err
	}
	if userId <= 0 {
		return models.UserDto{}, jwt.ErrTokenInvalidSubject
	}
	// Check if the token is expired
	exp, err := claims.GetExpirationTime()
	if err != nil {
		return models.UserDto{}, err
	}
	if exp.Before(time.Now()) {
		return models.UserDto{}, jwt.ErrTokenExpired
	}
	return models.UserDto{Id: uint64(userId)}, nil
}
