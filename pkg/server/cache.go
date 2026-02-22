package server

import (
	"sync"
	"time"
)

type cachedResponse struct {
	payload     []byte
	contentType string
	storedAt    time.Time
}

type responseCache struct {
	ttl     time.Duration
	mu      sync.RWMutex
	entries map[string]cachedResponse
}

func newResponseCache(ttl time.Duration) *responseCache {
	if ttl <= 0 {
		return &responseCache{ttl: ttl}
	}

	return &responseCache{
		ttl:     ttl,
		entries: make(map[string]cachedResponse),
	}
}

func (c *responseCache) Get(key string) (cachedResponse, bool) {
	if c == nil || c.ttl <= 0 {
		return cachedResponse{}, false
	}

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return cachedResponse{}, false
	}

	if time.Since(entry.storedAt) > c.ttl {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return cachedResponse{}, false
	}

	return entry, true
}

func (c *responseCache) Set(key string, payload []byte, contentType string) {
	if c == nil || c.ttl <= 0 {
		return
	}

	buf := make([]byte, len(payload))
	copy(buf, payload)

	c.mu.Lock()
	if c.entries == nil {
		c.entries = make(map[string]cachedResponse)
	}
	c.entries[key] = cachedResponse{
		payload:     buf,
		contentType: contentType,
		storedAt:    time.Now(),
	}
	c.mu.Unlock()
}
