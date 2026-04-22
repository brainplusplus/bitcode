package cache

import (
	"context"
	"time"
)

type Cache interface {
	Get(key string) (any, bool)
	Set(key string, value any, ttl time.Duration)
	Delete(key string)
	Clear()
}

type CacheConfig struct {
	Driver   string // "memory" (default), "redis"
	RedisURL string
}

func NewCache(cfg CacheConfig) Cache {
	if cfg.Driver == "redis" && cfg.RedisURL != "" {
		redis, err := NewRedisCache(context.Background(), cfg.RedisURL)
		if err == nil {
			return redis
		}
	}
	return NewMemoryCache()
}
