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
