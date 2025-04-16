package configs

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type APP_CONFIG struct {
	RedirectPort       string
	AdminPort          string
	ApiUrl             string
	JwtSigningKey      string
	JwtExpiryTimeHours int
	CORSOriginList     string
}

var AppConfig APP_CONFIG

func GetEnv(key string) string {
	return os.Getenv(key)
}

func loadConfig() {
	if GetEnv("ENV") != "production" {
		err := godotenv.Load(".env.development")
		if err != nil {
			panic("Unable to load env file")
		}
	}

	PORT := GetEnv("PORT")
	if PORT == "" {
		PORT = "8085"
	}

	ADMIN_PORT := GetEnv("ADMIN_PORT")
	if ADMIN_PORT == "" {
		ADMIN_PORT = "8086"
	}

	API_URL := GetEnv("API_URL")
	if API_URL == "" {
		API_URL = "http://localhost:8080"
	}

	jwtExpiryTimeHours, err := strconv.Atoi(GetEnv("JWT_EXPIRY_TIME_HOURS"))
	if err != nil {
		jwtExpiryTimeHours = 3600 // Default to 1 hour
	}

	corsOriginList := GetEnv("CORS_ORIGINS")
	if corsOriginList == "" {
		corsOriginList = "*"
	}

	AppConfig = APP_CONFIG{
		RedirectPort:       PORT,
		ApiUrl:             API_URL,
		JwtSigningKey:      GetEnv("JWT_SIGNING_KEY"),
		JwtExpiryTimeHours: jwtExpiryTimeHours,
		AdminPort:          ADMIN_PORT,
		CORSOriginList:     corsOriginList,
	}
}

func InitConfig() {
	loadConfig()

	LoadRedisConfig()
	LoadPostgresConfig()
}
