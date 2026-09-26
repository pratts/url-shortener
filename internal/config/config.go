// Package config loads and validates each service's settings from the
// environment. Every problem is reported at once rather than failing on the
// first missing variable.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Postgres holds database connection settings.
type Postgres struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// DSN returns a URL-form connection string, so empty or special-character
// values are escaped correctly.
func (p Postgres) DSN() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(p.User, p.Password),
		Host:     net.JoinHostPort(p.Host, strconv.Itoa(p.Port)),
		Path:     "/" + p.Database,
		RawQuery: url.Values{"sslmode": {p.SSLMode}}.Encode(),
	}
	return u.String()
}

// Redis holds cache connection settings.
type Redis struct {
	Host     string
	Port     int
	Password string
	DB       int
	// TTL is how long a resolved short link stays cached.
	TTL time.Duration
}

func (r Redis) Addr() string {
	return net.JoinHostPort(r.Host, strconv.Itoa(r.Port))
}

// HTTP holds listener and reverse-proxy settings.
type HTTP struct {
	Port string
	// ProxyHeader holds the client IP (e.g. X-Forwarded-For) when behind a
	// reverse proxy. Empty means use the socket address.
	ProxyHeader    string
	TrustedProxies []string
}

// Admin is the admin API's configuration.
type Admin struct {
	HTTP     HTTP
	Postgres Postgres
	Redis    Redis
	// ShortURLBase is the public base URL of the redirect service, used to
	// build short links and to refuse links that point back at it.
	ShortURLBase        string
	JWTSigningKey       string
	JWTTTL              time.Duration
	CORSOrigins         string
	EnableSwagger       bool
	RegistrationEnabled bool
}

// Redirect is the redirect service's configuration.
type Redirect struct {
	HTTP         HTTP
	Postgres     Postgres
	Redis        Redis
	ShortURLBase string
}

const minJWTKeyLen = 32

var sslModes = map[string]bool{
	"disable": true, "allow": true, "prefer": true,
	"require": true, "verify-ca": true, "verify-full": true,
}

// LoadAdmin reads the admin API's configuration.
func LoadAdmin() (Admin, error) {
	l := newLoader()
	cfg := Admin{
		HTTP:                l.http("ADMIN_PORT", "8086"),
		Postgres:            l.postgres(),
		Redis:               l.redis(),
		ShortURLBase:        l.baseURL("SHORT_URL_BASE"),
		JWTSigningKey:       l.required("JWT_SIGNING_KEY"),
		JWTTTL:              time.Duration(l.positiveInt("JWT_EXPIRY_TIME_HOURS", 1)) * time.Hour,
		CORSOrigins:         l.required("CORS_ORIGINS"),
		EnableSwagger:       l.boolean("ENABLE_SWAGGER", !l.production),
		RegistrationEnabled: l.boolean("REGISTRATION_ENABLED", true),
	}
	if cfg.JWTSigningKey != "" && len(cfg.JWTSigningKey) < minJWTKeyLen {
		l.fail("JWT_SIGNING_KEY must be at least %d bytes", minJWTKeyLen)
	}
	for _, origin := range splitList(cfg.CORSOrigins) {
		if origin == "*" {
			// Credentials are allowed, so a wildcard would let any site call the API as the user.
			l.fail("CORS_ORIGINS must list explicit origins, not *")
		}
	}
	return cfg, l.err()
}

// LoadRedirect reads the redirect service's configuration.
func LoadRedirect() (Redirect, error) {
	l := newLoader()
	cfg := Redirect{
		HTTP:         l.http("REDIRECT_PORT", "8085"),
		Postgres:     l.postgres(),
		Redis:        l.redis(),
		ShortURLBase: l.baseURL("SHORT_URL_BASE"),
	}
	return cfg, l.err()
}

// LoadPostgres reads only the database settings, for command-line tools.
func LoadPostgres() (Postgres, error) {
	l := newLoader()
	cfg := l.postgres()
	return cfg, l.err()
}

type loader struct {
	production bool
	errs       []string
}

func newLoader() *loader {
	l := &loader{production: os.Getenv("ENV") == "production"}
	if !l.production {
		// Local development reads .env when present; real environment
		// variables always take precedence.
		if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
			l.fail("reading .env: %v", err)
		}
	}
	return l
}

func (l *loader) fail(format string, args ...interface{}) {
	l.errs = append(l.errs, fmt.Sprintf(format, args...))
}

func (l *loader) err() error {
	if len(l.errs) == 0 {
		return nil
	}
	return fmt.Errorf("invalid configuration:\n  - %s", strings.Join(l.errs, "\n  - "))
}

func (l *loader) optional(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func (l *loader) required(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		l.fail("%s is required", key)
	}
	return v
}

func (l *loader) integer(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		l.fail("%s must be an integer, got %q", key, v)
		return def
	}
	return n
}

func (l *loader) positiveInt(key string, def int) int {
	n := l.integer(key, def)
	if n <= 0 {
		l.fail("%s must be greater than 0", key)
	}
	return n
}

func (l *loader) port(key, def string) string {
	v := l.optional(key, def)
	if n, err := strconv.Atoi(v); err != nil || n < 1 || n > 65535 {
		l.fail("%s must be a port number, got %q", key, v)
	}
	return v
}

func (l *loader) boolean(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		l.fail("%s must be true or false, got %q", key, v)
		return def
	}
	return b
}

func (l *loader) baseURL(key string) string {
	v := strings.TrimRight(l.required(key), "/")
	if v == "" {
		return v
	}
	u, err := url.Parse(v)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		l.fail("%s must be an absolute http(s) URL, got %q", key, v)
	}
	return v
}

func (l *loader) http(portKey, defaultPort string) HTTP {
	cfg := HTTP{
		Port:           l.port(portKey, defaultPort),
		ProxyHeader:    l.optional("PROXY_HEADER", ""),
		TrustedProxies: splitList(os.Getenv("TRUSTED_PROXIES")),
	}
	if cfg.ProxyHeader != "" && len(cfg.TrustedProxies) == 0 {
		l.fail("TRUSTED_PROXIES must be set when PROXY_HEADER is set, otherwise client IPs can be spoofed")
	}
	return cfg
}

func (l *loader) postgres() Postgres {
	defaultSSL := "disable"
	if l.production {
		defaultSSL = "require"
	}
	cfg := Postgres{
		Host:     l.required("DB_HOST"),
		Port:     l.integer("DB_PORT", 5432),
		User:     l.required("DB_USERNAME"),
		Password: os.Getenv("DB_PASSWORD"),
		Database: l.required("DB_DATABASE"),
		SSLMode:  l.optional("DB_SSLMODE", defaultSSL),
	}
	if !sslModes[cfg.SSLMode] {
		l.fail("DB_SSLMODE must be one of disable, allow, prefer, require, verify-ca, verify-full; got %q", cfg.SSLMode)
	}
	return cfg
}

func (l *loader) redis() Redis {
	return Redis{
		Host:     l.required("REDIS_HOST"),
		Port:     l.integer("REDIS_PORT", 6379),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       l.integer("REDIS_DB", 0),
		TTL:      time.Duration(l.positiveInt("REDIS_TTL", 3600)) * time.Second,
	}
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
