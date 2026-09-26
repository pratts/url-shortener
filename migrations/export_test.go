package migrations

import (
	"context"
	"database/sql"
)

// DownAll rolls back every migration. It exists only for tests.
func DownAll(ctx context.Context, db *sql.DB) error {
	p, err := provider(db)
	if err != nil {
		return err
	}
	_, err = p.DownTo(ctx, 0)
	return err
}
