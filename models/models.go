package models

import (
	"fmt"
	"shortener/configs"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var dbObj *gorm.DB

func InitDb() {
	dbConfig := configs.PgConfig
	connectionString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Username,
		dbConfig.Password,
		dbConfig.Database,
	)

	db, err := gorm.Open(postgres.Open(connectionString), &gorm.Config{})

	if err != nil {
		panic("failed to connect to the database")
	}

	db.AutoMigrate(&User{})
	db.AutoMigrate(&ShortenedURL{})
	dbObj = db
}
