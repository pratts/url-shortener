// Package platform opens connections to external services.
package platform

import (
	"context"
	"fmt"
	"time"

	"shortener/internal/config"
	"shortener/migrations"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// OpenPostgres connects to Postgres and verifies the connection.
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
		return nil, err
	}
	return db, nil
}

// OpenRedis connects to Redis and verifies the connection.
func OpenRedis(cfg config.Redis) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connecting to redis at %s: %w", cfg.Addr(), err)
	}
	return rdb, nil
}
