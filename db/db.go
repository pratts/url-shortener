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
	// Replaced by uidx_shortened_urls_short_code and idx_shortened_urls_owner_id.
	dropLegacyIndexes(db, &models.ShortenedURL{}, "idx_shortened_urls_short_code", "idx_shortened_urls_created_by")
	DBObj = db
}

func InitUrlRedictDb() {
	if err := DBObj.AutoMigrate(&models.UrlRedirect{}); err != nil {
		panic(fmt.Sprintf("database migration failed: %v", err))
	}
	// Replaced by idx_url_redirects_url_time.
	dropLegacyIndexes(DBObj, &models.UrlRedirect{}, "idx_url_redirects_short_url_id")
}

// dropLegacyIndexes removes indexes that newer ones make redundant. It runs
// after AutoMigrate has created the replacements.
func dropLegacyIndexes(db *gorm.DB, model interface{}, names ...string) {
	for _, name := range names {
		if db.Migrator().HasIndex(model, name) {
			if err := db.Migrator().DropIndex(model, name); err != nil {
				panic(fmt.Sprintf("database migration failed dropping %s: %v", name, err))
			}
		}
	}
}

// IsUniqueViolation reports whether err is a Postgres unique-constraint
// violation on a constraint or index whose name contains column.
func IsUniqueViolation(err error, column string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, column)
}
