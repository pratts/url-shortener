// Package testdb connects integration tests to real Postgres and Redis. Tests
// using it are skipped unless SHORTENER_TEST_DB names a Postgres database
// (on localhost, as $USER) that they may freely modify.
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
		Host: "localhost", Port: 5432, User: os.Getenv("USER"),
		Database: name, SSLMode: "disable",
	}
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

// Redis connects to the Redis DB named by SHORTENER_TEST_REDIS_DB (default 13).
func Redis(t *testing.T) *redis.Client {
	t.Helper()
	Config(t)
	n, _ := strconv.Atoi(os.Getenv("SHORTENER_TEST_REDIS_DB"))
	if n == 0 {
		n = 13
	}
	rdb, err := platform.OpenRedis(config.Redis{Host: "localhost", Port: 6379, DB: n, TTL: time.Minute})
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
