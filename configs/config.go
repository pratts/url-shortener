package configs

import (
	"os"

	"github.com/joho/godotenv"
)

type APP_CONFIG struct {
	Port   string
	ApiUrl string
}

var AppConfig APP_CONFIG

func GetEnv(key string) string {
	return os.Getenv(key)
}

func loadConfigFile() {
	err := godotenv.Load(".env")
	if err != nil {
		panic("Unable to load env file")
	}
}

func loadDefaultConfig() {
	PORT := GetEnv("PORT")
	if PORT == "" {
		PORT = "8080"
	}

	API_URL := GetEnv("API_URL")
	if API_URL == "" {
		API_URL = "http://localhost:8080"
	}

	AppConfig = APP_CONFIG{
		Port:   PORT,
		ApiUrl: API_URL,
	}
}

func InitConfig() {
	loadConfigFile()
	loadDefaultConfig()

	LoadRedisConfig()
	LoadPostgresConfig()
}
