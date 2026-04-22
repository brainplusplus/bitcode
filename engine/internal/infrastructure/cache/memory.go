package cache

import (
	"sync"
	"time"
)

type cacheEntry struct {
	value     any
	expiresAt time.Time
}

type MemoryCache struct {
	data map[string]cacheEntry
	mu   sync.RWMutex
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		data: make(map[string]cacheEntry),
	}
}

func (c *MemoryCache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok {
		return nil, false
	}
	if !entry.expiresAt.IsZero() && time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.value, true
}

func (c *MemoryCache) Set(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}
	c.data[key] = cacheEntry{value: value, expiresAt: expiresAt}
}

func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]cacheEntry)
}
