package bridge

import (
	"time"

	infracache "github.com/bitcode-framework/bitcode/internal/infrastructure/cache"
)

type cacheBridge struct {
	backend infracache.Cache
}

func newCacheBridge(backend infracache.Cache) *cacheBridge {
	return &cacheBridge{backend: backend}
}

func (c *cacheBridge) Get(key string) (any, error) {
	val, ok := c.backend.Get(key)
	if !ok {
		return nil, nil
	}
	return val, nil
}

func (c *cacheBridge) Set(key string, value any, opts *CacheSetOptions) error {
	ttl := time.Duration(0)
	if opts != nil && opts.TTL > 0 {
		ttl = time.Duration(opts.TTL) * time.Second
	}
	c.backend.Set(key, value, ttl)
	return nil
}

func (c *cacheBridge) Del(key string) error {
	c.backend.Delete(key)
	return nil
}
