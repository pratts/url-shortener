package cache

import (
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// LimiterStorage implements fiber.Storage on top of Rdb so rate-limit counters
// are shared by every instance of a service.
type LimiterStorage struct {
	prefix string
}

func NewLimiterStorage(prefix string) *LimiterStorage {
	return &LimiterStorage{prefix: prefix}
}

func (s *LimiterStorage) Get(key string) ([]byte, error) {
	val, err := Rdb.Get(ctx, s.prefix+key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	return val, err
}

func (s *LimiterStorage) Set(key string, val []byte, exp time.Duration) error {
	return Rdb.Set(ctx, s.prefix+key, val, exp).Err()
}

func (s *LimiterStorage) Delete(key string) error {
	return Rdb.Del(ctx, s.prefix+key).Err()
}

func (s *LimiterStorage) Reset() error {
	iter := Rdb.Scan(ctx, 0, s.prefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		if err := Rdb.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

func (s *LimiterStorage) Close() error {
	return nil
}
