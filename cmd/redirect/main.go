// Command redirect serves short links.
package main

import (
	"log"

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
	links, err := shortlink.NewService(
		shortlink.NewGormRepository(db),
		cache.NewLinkCache(rdb, cfg.Redis.TTL),
		cfg.ShortURLBase,
	)
	if err != nil {
		log.Fatal(err)
	}

	handler := &redirect.Handler{Links: links, Clicks: analytics.NewRecorder(db)}
	app := httpapi.NewApp(cfg.HTTP)
	handler.Register(app)
	app.Use(httpapi.NotFound)

	log.Printf("redirect service listening on :%s", cfg.HTTP.Port)
	log.Fatal(app.Listen(":" + cfg.HTTP.Port))
}
