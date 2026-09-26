// Package migrations holds the versioned database schema. Migrations are
// applied by cmd/migrate as a deploy step; services only check that the
// schema is current and refuse to start otherwise.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var files embed.FS

// ErrPending is returned by EnsureCurrent when migrations have not been applied.
var ErrPending = errors.New("database schema is out of date: run the migrate command first")

func provider(db *sql.DB) (*goose.Provider, error) {
	return goose.NewProvider(goose.DialectPostgres, db, files)
}

// Up applies all pending migrations and returns the resulting version.
func Up(ctx context.Context, db *sql.DB) (int64, error) {
	p, err := provider(db)
	if err != nil {
		return 0, err
	}
	if _, err := p.Up(ctx); err != nil {
		return 0, err
	}
	return p.GetDBVersion(ctx)
}

// Status returns the applied and latest available schema versions. It only
// reads from the database, so services can call it without DDL rights; a
// database that has never been migrated is at version 0.
func Status(ctx context.Context, db *sql.DB) (current, latest int64, err error) {
	p, err := provider(db)
	if err != nil {
		return 0, 0, err
	}
	sources := p.ListSources()
	if len(sources) > 0 {
		latest = sources[len(sources)-1].Version
	}
	var tracked bool
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('goose_db_version') IS NOT NULL").Scan(&tracked); err != nil {
		return 0, 0, err
	}
	if tracked {
		if err := db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version").Scan(&current); err != nil {
			return 0, 0, err
		}
	}
	return current, latest, nil
}

// EnsureCurrent returns ErrPending if any migration has not been applied.
func EnsureCurrent(ctx context.Context, db *sql.DB) error {
	current, latest, err := Status(ctx, db)
	if err != nil {
		return fmt.Errorf("checking schema version: %w", err)
	}
	if current < latest {
		return fmt.Errorf("%w (at version %d, latest is %d)", ErrPending, current, latest)
	}
	return nil
}
