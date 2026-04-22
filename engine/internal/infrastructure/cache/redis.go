package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisCache(ctx context.Context, url string) (*RedisCache, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &RedisCache{client: client, ctx: ctx}, nil
}

func (c *RedisCache) Get(key string) (any, bool) {
	val, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		return nil, false
	}

	var result any
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return val, true
	}
	return result, true
}

func (c *RedisCache) Set(key string, value any, ttl time.Duration) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	c.client.Set(c.ctx, key, string(data), ttl)
}

func (c *RedisCache) Delete(key string) {
	c.client.Del(c.ctx, key)
}

func (c *RedisCache) Clear() {
	c.client.FlushDB(c.ctx)
}
