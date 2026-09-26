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
	"log"

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
		log.Fatal(err)
	}
	db, err := platform.OpenPostgresCurrent(cfg.Postgres)
	if err != nil {
		log.Fatal(err)
	}
	rdb, err := platform.OpenRedis(cfg.Redis)
	if err != nil {
		log.Fatal(err)
	}
	tokens, err := auth.NewTokenService(cfg.JWTSigningKey, cfg.JWTTTL)
	if err != nil {
		log.Fatal(err)
	}
	links, err := shortlink.NewService(
		shortlink.NewGormRepository(db),
		cache.NewLinkCache(rdb, cfg.Redis.TTL),
		cfg.ShortURLBase,
	)
	if err != nil {
		log.Fatal(err)
	}

	handler := &admin.Handler{
		Links:               links,
		Users:               user.NewService(user.NewGormRepository(db)),
		Tokens:              tokens,
		Limits:              ratelimit.New(cache.NewLimiterStorage(rdb, "rl:")),
		RegistrationEnabled: cfg.RegistrationEnabled,
	}

	app := httpapi.NewApp(cfg.HTTP)
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

	log.Printf("admin API listening on :%s", cfg.HTTP.Port)
	log.Fatal(app.Listen(":" + cfg.HTTP.Port))
}
