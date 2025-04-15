package configs

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type APP_CONFIG struct {
	Port               string
	AdminPort          string
	ApiUrl             string
	JwtSigningKey      string
	JwtExpiryTimeHours int
}

var AppConfig APP_CONFIG

func GetEnv(key string) string {
	return os.Getenv(key)
}

func loadConfigFile() {
	if GetEnv("ENV") == "production" {
		return
	}
	err := godotenv.Load(".env")
	if err != nil {
		panic("Unable to load env file")
	}
}

func loadDefaultConfig() {
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

	AppConfig = APP_CONFIG{
		Port:               PORT,
		ApiUrl:             API_URL,
		JwtSigningKey:      GetEnv("JWT_SIGNING_KEY"),
		JwtExpiryTimeHours: jwtExpiryTimeHours,
		AdminPort:          ADMIN_PORT,
	}
}

func InitConfig() {
	loadConfigFile()
	loadDefaultConfig()

	LoadRedisConfig()
	LoadPostgresConfig()
}
