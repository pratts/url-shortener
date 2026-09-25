package configs

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type APP_CONFIG struct {
	RedirectPort       string
	AdminPort          string
	ApiUrl             string
	JwtSigningKey      string
	JwtExpiryTimeHours int
	CORSOriginList     string
	EnableSwagger      bool
	// RegistrationEnabled exposes POST /users/register.
	RegistrationEnabled bool
	// ProxyHeader is the header holding the client IP (e.g. X-Forwarded-For)
	// when running behind a reverse proxy. Empty means use the socket address.
	ProxyHeader    string
	TrustedProxies []string
}

func IsProduction() bool {
	return GetEnv("ENV") == "production"
}

func splitList(value string) []string {
	var out []string
	for _, v := range strings.Split(value, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
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

	REDIRECT_PORT := GetEnv("REDIRECT_PORT")
	if REDIRECT_PORT == "" {
		REDIRECT_PORT = "8085"
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
	if err != nil || jwtExpiryTimeHours <= 0 {
		jwtExpiryTimeHours = 1
	}

	enableSwagger := !IsProduction()
	if v := GetEnv("ENABLE_SWAGGER"); v != "" {
		enableSwagger, _ = strconv.ParseBool(v)
	}

	registrationEnabled := true
	if v := GetEnv("REGISTRATION_ENABLED"); v != "" {
		registrationEnabled, err = strconv.ParseBool(v)
		if err != nil {
			panic("Invalid REGISTRATION_ENABLED value, must be true or false")
		}
	}

	proxyHeader := GetEnv("PROXY_HEADER")
	trustedProxies := splitList(GetEnv("TRUSTED_PROXIES"))
	if proxyHeader != "" && len(trustedProxies) == 0 {
		panic("TRUSTED_PROXIES must be set when PROXY_HEADER is set, otherwise client IPs can be spoofed")
	}

	corsOriginList := GetEnv("CORS_ORIGINS")
	if corsOriginList == "" {
		corsOriginList = "*"
	}

	AppConfig = APP_CONFIG{
		RedirectPort:        REDIRECT_PORT,
		ApiUrl:              API_URL,
		JwtSigningKey:       GetEnv("JWT_SIGNING_KEY"),
		JwtExpiryTimeHours:  jwtExpiryTimeHours,
		AdminPort:           ADMIN_PORT,
		CORSOriginList:      corsOriginList,
		EnableSwagger:       enableSwagger,
		RegistrationEnabled: registrationEnabled,
		ProxyHeader:         proxyHeader,
		TrustedProxies:      trustedProxies,
	}
}

func InitConfig() {
	loadConfig()

	LoadRedisConfig()
	LoadPostgresConfig()
}
