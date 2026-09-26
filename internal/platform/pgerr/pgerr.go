// Package pgerr classifies Postgres errors.
package pgerr

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// IsUniqueViolation reports whether err is a unique-constraint violation on a
// constraint or index whose name contains column.
func IsUniqueViolation(err error, column string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, column)
}
