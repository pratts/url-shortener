// Package cache stores resolved short links and rate-limit counters in Redis.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"shortener/internal/shortlink"

	"github.com/redis/go-redis/v9"
)

// missTTL is how long a "no such code" answer is cached. It is short because
// the cache also absorbs lookups for codes that are about to be created.
const missTTL = time.Minute

// LinkCache maps short codes to shortlink entries.
type LinkCache struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewLinkCache(rdb *redis.Client, ttl time.Duration) *LinkCache {
	return &LinkCache{rdb: rdb, ttl: ttl}
}

func linkKey(code string) string {
	return "link:" + code
}

// Get returns the cached entry for code; ok is false on a miss. An entry that
// cannot be decoded is treated as a miss.
func (c *LinkCache) Get(ctx context.Context, code string) (shortlink.Entry, bool, error) {
	raw, err := c.rdb.Get(ctx, linkKey(code)).Bytes()
	if errors.Is(err, redis.Nil) {
		return shortlink.Entry{}, false, nil
	}
	if err != nil {
		return shortlink.Entry{}, false, err
	}
	var e shortlink.Entry
	if err := json.Unmarshal(raw, &e); err != nil {
		return shortlink.Entry{}, false, nil
	}
	return e, true, nil
}

func (c *LinkCache) Set(ctx context.Context, code string, e shortlink.Entry) error {
	raw, err := json.Marshal(e)
	if err != nil {
		return err
	}
	ttl := c.ttl
	if e.Missing {
		ttl = missTTL
	}
	return c.rdb.Set(ctx, linkKey(code), raw, ttl).Err()
}

func (c *LinkCache) Delete(ctx context.Context, code string) error {
	return c.rdb.Del(ctx, linkKey(code)).Err()
}

// LimiterStorage implements fiber.Storage on Redis so rate-limit counters are
// shared by every instance of a service.
type LimiterStorage struct {
	rdb    *redis.Client
	prefix string
}

func NewLimiterStorage(rdb *redis.Client, prefix string) *LimiterStorage {
	return &LimiterStorage{rdb: rdb, prefix: prefix}
}

// fiber.Storage has no context parameter, so these calls use Background.

func (s *LimiterStorage) Get(key string) ([]byte, error) {
	val, err := s.rdb.Get(context.Background(), s.prefix+key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	return val, err
}

func (s *LimiterStorage) Set(key string, val []byte, exp time.Duration) error {
	return s.rdb.Set(context.Background(), s.prefix+key, val, exp).Err()
}

func (s *LimiterStorage) Delete(key string) error {
	return s.rdb.Del(context.Background(), s.prefix+key).Err()
}

func (s *LimiterStorage) Reset() error {
	ctx := context.Background()
	iter := s.rdb.Scan(ctx, 0, s.prefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		if err := s.rdb.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

func (s *LimiterStorage) Close() error {
	return nil
}
