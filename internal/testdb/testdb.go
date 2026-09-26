// Package testdb connects integration tests to real Postgres and Redis. Tests
// using it are skipped unless SHORTENER_TEST_DB names a Postgres database they
// may freely modify. Connection settings default to a local server with the
// current user and no password; override them with SHORTENER_TEST_DB_HOST,
// _PORT, _USER and _PASSWORD, and SHORTENER_TEST_REDIS_HOST/_PORT/_DB.
package testdb

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"shortener/internal/config"
	"shortener/internal/platform"
	"shortener/migrations"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Config returns the connection settings for the test database, skipping the
// test when none is configured.
func Config(t *testing.T) config.Postgres {
	t.Helper()
	name := os.Getenv("SHORTENER_TEST_DB")
	if name == "" {
		t.Skip("SHORTENER_TEST_DB not set; skipping integration test")
	}
	return config.Postgres{
		Host:     env("SHORTENER_TEST_DB_HOST", "localhost"),
		Port:     envInt("SHORTENER_TEST_DB_PORT", 5432),
		User:     env("SHORTENER_TEST_DB_USER", os.Getenv("USER")),
		Password: os.Getenv("SHORTENER_TEST_DB_PASSWORD"),
		Database: name,
		SSLMode:  "disable",
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return n
	}
	return def
}

// Postgres opens the test database, applies migrations and empties every table.
func Postgres(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := platform.OpenPostgres(Config(t))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })
	if _, err := migrations.Up(context.Background(), sqlDB); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("TRUNCATE users, shortened_urls, url_redirects RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatal(err)
	}
	return db
}

// Redis connects to Redis DB SHORTENER_TEST_REDIS_DB (default 13).
func Redis(t *testing.T) *redis.Client {
	t.Helper()
	Config(t)
	rdb, err := platform.OpenRedis(config.Redis{
		Host: env("SHORTENER_TEST_REDIS_HOST", "localhost"),
		Port: envInt("SHORTENER_TEST_REDIS_PORT", 6379),
		DB:   envInt("SHORTENER_TEST_REDIS_DB", 13),
		TTL:  time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rdb.Close() })
	return rdb
}

// CreateUser inserts a user and returns its ID, for tests that need an owner.
func CreateUser(t *testing.T, db *gorm.DB, email string) uint64 {
	t.Helper()
	var id uint64
	err := db.Raw("INSERT INTO users (email, password, name) VALUES (?, 'x', 'Test') RETURNING id", email).Scan(&id).Error
	if err != nil {
		t.Fatal(err)
	}
	return id
}
