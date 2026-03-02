/*
 * Iptv-Proxy is a project to proxyfie an m3u file and to proxyfie an Xtream iptv service (client API).
 * Copyright (C) 2020  Pierre-Emmanuel Jacquier
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package server

import (
	"sync"
	"time"
)

const defaultXMLTVCacheMaxEntries = 100

type cachedResponse struct {
	payload     []byte
	contentType string
	storedAt    time.Time
}

type responseCache struct {
	ttl        time.Duration
	maxEntries int // 0 = no limit
	mu         sync.RWMutex
	entries    map[string]cachedResponse
}

func newResponseCache(ttl time.Duration, maxEntries int) *responseCache {
	if ttl <= 0 {
		return &responseCache{ttl: ttl}
	}
	if maxEntries <= 0 {
		maxEntries = defaultXMLTVCacheMaxEntries
	}
	return &responseCache{
		ttl:        ttl,
		maxEntries: maxEntries,
		entries:    make(map[string]cachedResponse),
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
	for c.maxEntries > 0 && len(c.entries) >= c.maxEntries {
		var oldestKey string
		var oldestTime time.Time
		for k, e := range c.entries {
			if oldestKey == "" || e.storedAt.Before(oldestTime) {
				oldestKey, oldestTime = k, e.storedAt
			}
		}
		if oldestKey != "" {
			delete(c.entries, oldestKey)
		} else {
			break
		}
	}
	c.entries[key] = cachedResponse{payload: buf, contentType: contentType, storedAt: time.Now()}
	c.mu.Unlock()
}
