package cache

import (
	"sync"
	"time"
)

type item struct {
	value     []byte
	expiresAt time.Time
}

type Store interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte)
}

type MemoryCache struct {
	ttl   time.Duration
	items map[string]item
	mu    sync.RWMutex
}

func NewMemory(ttl time.Duration) *MemoryCache {
	return &MemoryCache{
		ttl:   ttl,
		items: make(map[string]item),
	}
}

func (c *MemoryCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	it, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}

	if time.Now().After(it.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}

	return it.value, true
}

func (c *MemoryCache) Set(key string, value []byte) {
	c.mu.Lock()
	c.items[key] = item{value: value, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

