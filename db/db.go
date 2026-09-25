package db

import (
	"errors"
	"fmt"
	"shortener/configs"
	"shortener/models"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DBObj *gorm.DB

func InitDb() {
	dbConfig := configs.PgConfig
	connectionString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Username,
		dbConfig.Password,
		dbConfig.Database,
		dbConfig.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{})

	if err != nil {
		panic("failed to connect to the database")
	}

	// Fail fast: a partial migration leaves the service running against a
	// schema it cannot use.
	if err := db.AutoMigrate(&models.ShortenedURL{}, &models.User{}); err != nil {
		panic(fmt.Sprintf("database migration failed: %v", err))
	}
	DBObj = db
}

func InitUrlRedictDb() {
	if err := DBObj.AutoMigrate(&models.UrlRedirect{}); err != nil {
		panic(fmt.Sprintf("database migration failed: %v", err))
	}
}

// IsUniqueViolation reports whether err is a Postgres unique-constraint
// violation on a constraint or index whose name contains column.
func IsUniqueViolation(err error, column string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, column)
}
