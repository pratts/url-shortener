package cache

import (
	"context"
	"fmt"
	"shortener/configs"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

var Rdb *redis.Client

func InitCache() {
	Rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", configs.RedisConfig.Host, configs.RedisConfig.Port),
		Password: configs.RedisConfig.Password, // no password set
		DB:       configs.RedisConfig.Database, // use default DB
	})
}

func GetFromCache(key string) (string, error) {
	val, err := Rdb.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

func SetToCache(key string, value string) error {
	err := Rdb.Set(ctx, key, value, 0).Err()
	if err != nil {
		return err
	}
	return nil
}

func DeleteFromCache(key string) error {
	err := Rdb.Del(ctx, key).Err()
	if err != nil {
		return err
	}
	return nil
}

func CacheSetWithExpiration(key string, value string, expiration int) error {
	err := Rdb.Set(ctx, key, value, time.Duration(expiration)*time.Second).Err()
	if err != nil {
		return err
	}
	return nil
}
