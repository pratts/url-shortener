package configs

import "strconv"

type POSTGRES_CONFIG struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	Schema   string `json:"schema"`
}

var PgConfig POSTGRES_CONFIG

func LoadPostgresConfig() {
	port, err := strconv.Atoi(GetEnv("DB_PORT"))
	if err != nil {
		panic("Invalid DB_PORT value")
	}
	PgConfig = POSTGRES_CONFIG{
		Host:     GetEnv("DB_HOST"),
		Port:     port,
		Username: GetEnv("DB_USERNAME"),
		Password: GetEnv("DB_PASSWORD"),
		Database: GetEnv("DB_DATABASE"),
		Schema:   GetEnv("DB_SCHEMA"),
	}
}
