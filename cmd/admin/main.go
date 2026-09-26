// Command admin serves the admin API.
//
// @title                       URL Shortener API
// @version                     1.0
// @description                 Admin API for creating and managing short URLs.
// @contact.name                Prateek Sharma
// @contact.email               prateeksharma.2801@gmail.com
// @BasePath                    /api/v1
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 "Bearer <token>" from POST /users/login
package main

import (
	"context"
	"log/slog"
	"os"

	_ "shortener/docs"
	"shortener/internal/auth"
	"shortener/internal/cache"
	"shortener/internal/config"
	"shortener/internal/httpapi"
	"shortener/internal/httpapi/admin"
	"shortener/internal/platform"
	"shortener/internal/ratelimit"
	"shortener/internal/shortlink"
	"shortener/internal/user"

	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
)

func main() {
	cfg, err := config.LoadAdmin()
	if err != nil {
		fatal(slog.Default(), err)
	}
	log := platform.NewLogger(cfg.HTTP.LogFormat, os.Stderr).With("service", "admin")
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

	tokens, err := auth.NewTokenService(cfg.JWTSigningKey, cfg.JWTTTL)
	if err != nil {
		fatal(log, err)
	}
	links, err := shortlink.NewService(
		shortlink.NewGormRepository(db),
		cache.NewLinkCache(rdb, cfg.Redis.TTL),
		cfg.ShortURLBase,
	)
	if err != nil {
		fatal(log, err)
	}

	handler := &admin.Handler{
		Links:               links,
		Users:               user.NewService(user.NewGormRepository(db)),
		Tokens:              tokens,
		Limits:              ratelimit.New(cache.NewLimiterStorage(rdb, "rl:")),
		RegistrationEnabled: cfg.RegistrationEnabled,
	}

	app := httpapi.NewApp(cfg.HTTP, log)
	httpapi.RegisterHealth(app, map[string]httpapi.Check{
		"postgres": sqlDB.PingContext,
		"redis":    func(ctx context.Context) error { return rdb.Ping(ctx).Err() },
	})
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowCredentials: true,
		AllowHeaders:     "Authorization,Content-Type,Accept",
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
	}))
	api := app.Group("/api/v1")
	handler.Register(api)
	if cfg.EnableSwagger {
		api.Get("/swagger/*", swagger.HandlerDefault)
	}
	app.Use(httpapi.NotFound)

	err = httpapi.Serve(app, cfg.HTTP.Port, log,
		func(context.Context) error { return rdb.Close() },
		func(context.Context) error { return sqlDB.Close() },
	)
	if err != nil {
		fatal(log, err)
	}
}

func fatal(log *slog.Logger, err error) {
	log.Error("admin API failed", "err", err)
	os.Exit(1)
}
