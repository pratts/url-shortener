package configs

import "strconv"

type REDIS_CONFIG struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	Database int    `json:"database"`
	TTL      int    `json:"ttl"`
}

var RedisConfig REDIS_CONFIG

func LoadRedisConfig() {
	port, err := strconv.Atoi(GetEnv("REDIS_PORT"))
	if err != nil {
		panic("Invalid REDIS_PORT value")
	}

	ttl, err := strconv.Atoi(GetEnv("REDIS_TTL"))
	if err != nil {
		panic("Invalid REDIS_TTL value")
	}

	db := 0
	redisDb := GetEnv("REDIS_DB")
	if redisDb != "" {
		rdb, err := strconv.Atoi(redisDb)
		if err != nil {
			panic("Invalid REDIS_DB value")
		}
		db = rdb
	} else {
		db = 0
	}
	RedisConfig = REDIS_CONFIG{
		Host:     GetEnv("REDIS_HOST"),
		Port:     port,
		Password: GetEnv("REDIS_PASSWORD"),
		Database: db,
		TTL:      ttl,
	}
}
