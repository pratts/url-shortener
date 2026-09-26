// Command migrate applies database migrations. Run it before starting or
// upgrading the services, which refuse to start on an out-of-date schema.
//
//	go run ./cmd/migrate          # apply pending migrations
//	go run ./cmd/migrate status   # show current and latest versions
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"shortener/internal/config"
	"shortener/internal/platform"
	"shortener/migrations"
)

func main() {
	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	cfg, err := config.LoadPostgres()
	if err != nil {
		log.Fatal(err)
	}
	db, err := platform.OpenPostgres(cfg)
	if err != nil {
		log.Fatal(err)
	}
	sqlDB, _ := db.DB()
	ctx := context.Background()

	switch cmd {
	case "up":
		version, err := migrations.Up(ctx, sqlDB)
		if err != nil {
			log.Fatalf("migration failed: %v", err)
		}
		fmt.Printf("schema is at version %d\n", version)
	case "status":
		current, latest, err := migrations.Status(ctx, sqlDB)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("current version %d, latest %d\n", current, latest)
		if current < latest {
			os.Exit(1)
		}
	default:
		log.Fatalf("unknown command %q (use up or status)", cmd)
	}
}
