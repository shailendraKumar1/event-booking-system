package cache

import (
	"sync"
	"time"

	"event-booking-system/internal/domain"
)

// EventCache is a TTL-based decorator cache for event reads (Decorator pattern).
// Thread-safe via RWMutex; eviction is lazy on read.
type EventCache struct {
	mu    sync.RWMutex
	ttl   time.Duration
	items map[string]cacheEntry
}

type cacheEntry struct {
	event   domain.Event
	expires time.Time
}

func NewEventCache(ttl time.Duration) *EventCache {
	return &EventCache{
		ttl:   ttl,
		items: make(map[string]cacheEntry),
	}
}

func (c *EventCache) Get(id string) (domain.Event, bool) {
	c.mu.RLock()
	entry, ok := c.items[id]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expires) {
		if ok {
			c.mu.Lock()
			delete(c.items, id)
			c.mu.Unlock()
		}
		return domain.Event{}, false
	}
	return entry.event, true
}

func (c *EventCache) Put(e domain.Event) {
	c.mu.Lock()
	c.items[e.ID] = cacheEntry{event: e, expires: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

func (c *EventCache) Invalidate(id string) {
	c.mu.Lock()
	delete(c.items, id)
	c.mu.Unlock()
}

func (c *EventCache) Clear() {
	c.mu.Lock()
	c.items = make(map[string]cacheEntry)
	c.mu.Unlock()
}
