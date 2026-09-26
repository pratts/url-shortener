// Command redirect serves short links.
package main

import (
	"context"
	"log/slog"
	"os"

	"shortener/internal/analytics"
	"shortener/internal/cache"
	"shortener/internal/config"
	"shortener/internal/httpapi"
	"shortener/internal/httpapi/redirect"
	"shortener/internal/platform"
	"shortener/internal/shortlink"
)

func main() {
	cfg, err := config.LoadRedirect()
	if err != nil {
		fatal(slog.Default(), err)
	}
	log := platform.NewLogger(cfg.HTTP.LogFormat, os.Stderr).With("service", "redirect")
	slog.SetDefault(log)

	db, err := platform.OpenPostgresCurrent(cfg.Postgres)
	if err != nil {
		fatal(log, err)
	}
	sqlDB, _ := db.DB()
	rdb, err := platform.OpenRedis(cfg.Redis)
	if err != nil {
		fatal(log, err)
	}
	platform.WarnOnUnboundedRedis(context.Background(), rdb, log)

	links, err := shortlink.NewService(
		shortlink.NewGormRepository(db),
		cache.NewLinkCache(rdb, cfg.Redis.TTL),
		cfg.ShortURLBase,
	)
	if err != nil {
		fatal(log, err)
	}
	clicks := analytics.NewBatcher(analytics.NewGormStore(db), analytics.DefaultBatcherConfig)

	app := httpapi.NewApp(cfg.HTTP, log)
	// Health routes are registered before /:code so they take precedence.
	httpapi.RegisterHealth(app, map[string]httpapi.Check{
		"postgres": sqlDB.PingContext,
		"redis":    func(ctx context.Context) error { return rdb.Ping(ctx).Err() },
	})
	(&redirect.Handler{Links: links, Clicks: clicks}).Register(app)
	app.Use(httpapi.NotFound)

	err = httpapi.Serve(app, cfg.HTTP.Port, log,
		func(ctx context.Context) error {
			err := clicks.Close(ctx)
			log.Info("click writer stopped", "dropped", clicks.Dropped(), "failed", clicks.Failed())
			return err
		},
		func(context.Context) error { return rdb.Close() },
		func(context.Context) error { return sqlDB.Close() },
	)
	if err != nil {
		fatal(log, err)
	}
}

func fatal(log *slog.Logger, err error) {
	log.Error("redirect service failed", "err", err)
	os.Exit(1)
}
