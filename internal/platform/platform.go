// Package platform opens connections to external services.
package platform

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"time"

	"shortener/internal/config"
	"shortener/migrations"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewLogger returns a JSON or text slog logger writing to w.
func NewLogger(format string, w io.Writer) *slog.Logger {
	if format == "json" {
		return slog.New(slog.NewJSONHandler(w, nil))
	}
	return slog.New(slog.NewTextHandler(w, nil))
}

// OpenPostgres connects to Postgres with a bounded pool and verifies the
// connection.
func OpenPostgres(cfg config.Postgres) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		// Callers log unexpected errors themselves; GORM's own logging would
		// also report expected not-found lookups and unique violations.
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("connecting to postgres at %s:%d: %w", cfg.Host, cfg.Port, err)
	}
	return db, nil
}

// OpenPostgresCurrent connects to Postgres and fails if migrations are pending.
func OpenPostgresCurrent(cfg config.Postgres) (*gorm.DB, error) {
	db, err := OpenPostgres(cfg)
	if err != nil {
		return nil, err
	}
	sqlDB, _ := db.DB()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := migrations.EnsureCurrent(ctx, sqlDB); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}

// OpenRedis connects to Redis with bounded timeouts and verifies the
// connection. A slow Redis then degrades requests instead of stalling them.
func OpenRedis(cfg config.Redis) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		PoolTimeout:  2 * time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("connecting to redis at %s: %w", cfg.Addr(), err)
	}
	return rdb, nil
}

// WarnOnUnboundedRedis logs a warning when Redis can grow without limit or
// may evict keys that have no TTL. Every key this app writes has a TTL, so
// maxmemory with volatile-lru is the intended setup. Managed Redis services
// often forbid CONFIG GET; that is not treated as a problem.
func WarnOnUnboundedRedis(ctx context.Context, rdb *redis.Client, log *slog.Logger) {
	settings, err := rdb.ConfigGet(ctx, "maxmemory*").Result()
	if err != nil {
		log.Debug("could not read redis memory settings", "err", err)
		return
	}
	if limit, _ := strconv.ParseInt(settings["maxmemory"], 10, 64); limit == 0 {
		log.Warn("redis has no maxmemory limit; set maxmemory and maxmemory-policy volatile-lru")
	}
	if policy := settings["maxmemory-policy"]; policy == "noeviction" {
		log.Warn("redis maxmemory-policy is noeviction; writes will fail when memory is full", "policy", policy)
	}
}
