package config

import (
	"strings"
	"testing"
	"time"
)

// setEnv sets a complete, valid admin environment, then applies overrides
// (an empty value unsets a variable).
func setEnv(t *testing.T, overrides map[string]string) {
	t.Helper()
	env := map[string]string{
		"ENV":             "production", // skip reading .env
		"DB_HOST":         "db.internal",
		"DB_USERNAME":     "app",
		"DB_PASSWORD":     "secret",
		"DB_DATABASE":     "shortener",
		"REDIS_HOST":      "cache.internal",
		"SHORT_URL_BASE":  "https://tidylnk.com/",
		"JWT_SIGNING_KEY": strings.Repeat("k", 32),
		"CORS_ORIGINS":    "https://admin.tidylnk.com",
	}
	for k, v := range overrides {
		env[k] = v
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
}

func TestLoadAdminDefaults(t *testing.T) {
	setEnv(t, nil)
	cfg, err := LoadAdmin()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Port != "8086" || cfg.Postgres.Port != 5432 || cfg.Redis.Port != 6379 {
		t.Errorf("ports: %+v", cfg)
	}
	if cfg.Postgres.SSLMode != "require" {
		t.Errorf("production should default to sslmode=require, got %q", cfg.Postgres.SSLMode)
	}
	if cfg.EnableSwagger || !cfg.RegistrationEnabled {
		t.Errorf("production defaults: swagger %v, registration %v", cfg.EnableSwagger, cfg.RegistrationEnabled)
	}
	if cfg.JWTTTL != time.Hour || cfg.Redis.TTL != time.Hour {
		t.Errorf("TTLs: jwt %v, redis %v", cfg.JWTTTL, cfg.Redis.TTL)
	}
	if cfg.ShortURLBase != "https://tidylnk.com" {
		t.Errorf("trailing slash not trimmed: %q", cfg.ShortURLBase)
	}
}

func TestLoadAdminReportsEveryProblem(t *testing.T) {
	setEnv(t, map[string]string{
		"DB_HOST":         "",
		"REDIS_HOST":      "",
		"JWT_SIGNING_KEY": "too-short",
		"CORS_ORIGINS":    "https://a.example, *",
		"SHORT_URL_BASE":  "tidylnk.com",
		"DB_SSLMODE":      "sometimes",
		"REDIS_TTL":       "0",
		"ADMIN_PORT":      "99999",
		"DB_PORT":         "abc",
		"ENABLE_SWAGGER":  "maybe",
		"PROXY_HEADER":    "X-Forwarded-For",
	})
	_, err := LoadAdmin()
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{
		"DB_HOST is required", "REDIS_HOST is required", "JWT_SIGNING_KEY must be at least 32 bytes",
		"CORS_ORIGINS must list explicit origins", "SHORT_URL_BASE must be an absolute http(s) URL",
		"DB_SSLMODE must be one of", "REDIS_TTL must be greater than 0", "ADMIN_PORT must be a port number",
		"DB_PORT must be an integer", "ENABLE_SWAGGER must be true or false", "TRUSTED_PROXIES must be set",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %q:\n%v", want, err)
		}
	}
}

func TestCORSOriginsRequiredForAdminOnly(t *testing.T) {
	setEnv(t, map[string]string{"CORS_ORIGINS": ""})
	if _, err := LoadAdmin(); err == nil || !strings.Contains(err.Error(), "CORS_ORIGINS is required") {
		t.Fatalf("admin: got %v", err)
	}
	if _, err := LoadRedirect(); err != nil {
		t.Fatalf("redirect should not need CORS_ORIGINS: %v", err)
	}
}

func TestRedirectDoesNotNeedJWTKey(t *testing.T) {
	setEnv(t, map[string]string{"JWT_SIGNING_KEY": ""})
	cfg, err := LoadRedirect()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Port != "8085" {
		t.Errorf("port %q", cfg.HTTP.Port)
	}
}

func TestDSNEscapesValues(t *testing.T) {
	cases := map[string]Postgres{
		"postgres://app:@localhost:5432/shortener?sslmode=disable": {
			Host: "localhost", Port: 5432, User: "app", Password: "", Database: "shortener", SSLMode: "disable",
		},
		"postgres://app:p%40ss%20w%2Frd@db:6543/shortener?sslmode=require": {
			Host: "db", Port: 6543, User: "app", Password: "p@ss w/rd", Database: "shortener", SSLMode: "require",
		},
	}
	for want, cfg := range cases {
		if got := cfg.DSN(); got != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
}

func TestPoolTimeoutAndLogSettings(t *testing.T) {
	setEnv(t, nil)
	cfg, err := LoadRedirect()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Postgres.MaxOpenConns != 20 || cfg.Postgres.MaxIdleConns != 10 || cfg.HTTP.RequestTimeout != 5*time.Second || cfg.HTTP.LogFormat != "json" {
		t.Fatalf("defaults: %+v %+v", cfg.Postgres, cfg.HTTP)
	}

	setEnv(t, map[string]string{"DB_MAX_OPEN_CONNS": "5", "DB_MAX_IDLE_CONNS": "8", "LOG_FORMAT": "xml", "REQUEST_TIMEOUT_SECONDS": "0"})
	_, err = LoadRedirect()
	for _, want := range []string{"DB_MAX_IDLE_CONNS (8) must not exceed DB_MAX_OPEN_CONNS (5)", "LOG_FORMAT must be json or text", "REQUEST_TIMEOUT_SECONDS must be greater than 0"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("error does not mention %q: %v", want, err)
		}
	}
}
