package cache

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Clock abstracts time for testability.
//
// NOTE: Use RealClock in production.
// Tests can provide a fake clock.
type Clock interface {
	Now() time.Time
}

// RealClock is the production clock.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

// ErrNotFound is returned when a key is not in the cache (or expired).
var ErrNotFound = errors.New("cache: not found")

// Config defines cache behavior.
type Config struct {
	Enabled    bool
	Capacity   int           // max number of entries; <=0 means 0 (disabled behavior)
	DefaultTTL time.Duration // 0 means no expiration by default
}

// Metrics are best-effort counters.
// All fields are updated under lock.
type Metrics struct {
	Hits   uint64
	Misses uint64
}

// Cache is a thread-safe TTL + LRU cache.
//
// - If disabled (Config.Enabled=false or Capacity<=0), all operations are no-op and Get returns ErrNotFound.
// - TTL is optional per entry; 0 means "no expiration" for that entry.
// - DefaultTTL is applied when Set is called with ttl=0.
//
// This cache is in-memory only (no persistence).
type Cache struct {
	mu sync.Mutex

	cfg   Config
	clock Clock

	ll *list.List               // front=most recent
	m  map[string]*list.Element // key -> *list.Element

	metrics Metrics
}

type entry struct {
	key       string
	value     []byte
	expiresAt time.Time // zero means no expiration
}

// New creates a new cache.
//
// If cfg.Capacity is 0 or negative, the cache behaves as disabled.
func New(cfg Config, clock Clock) *Cache {
	if clock == nil {
		clock = RealClock{}
	}
	if cfg.Capacity < 0 {
		cfg.Capacity = 0
	}
	return &Cache{
		cfg:   cfg,
		clock: clock,
		ll:    list.New(),
		m:     make(map[string]*list.Element),
	}
}

// Enabled reports whether the cache is effectively enabled.
func (c *Cache) Enabled() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enabledLocked()
}

func (c *Cache) enabledLocked() bool {
	return c.cfg.Enabled && c.cfg.Capacity > 0
}

// Get returns the cached value for key.
// It returns ErrNotFound if missing or expired.
func (c *Cache) Get(key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabledLocked() {
		c.metrics.Misses++
		return nil, ErrNotFound
	}

	el, ok := c.m[key]
	if !ok {
		c.metrics.Misses++
		return nil, ErrNotFound
	}

	e := el.Value.(*entry)
	if c.isExpiredLocked(e) {
		c.removeElementLocked(el)
		c.metrics.Misses++
		return nil, ErrNotFound
	}

	c.ll.MoveToFront(el)
	c.metrics.Hits++
	return cloneBytes(e.value), nil
}

// Set stores value for key.
//
// ttl:
// - ttl > 0: expires after ttl
// - ttl == 0: uses Config.DefaultTTL; if that is also 0, no expiration
// - ttl < 0: treated as immediate expiration (effectively delete)
func (c *Cache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabledLocked() {
		return
	}

	if ttl < 0 {
		c.deleteLocked(key)
		return
	}
	if ttl == 0 {
		ttl = c.cfg.DefaultTTL
	}

	expiresAt := time.Time{}
	if ttl > 0 {
		expiresAt = c.clock.Now().Add(ttl)
	}

	if el, ok := c.m[key]; ok {
		e := el.Value.(*entry)
		e.value = cloneBytes(value)
		e.expiresAt = expiresAt
		c.ll.MoveToFront(el)
		return
	}

	e := &entry{
		key:       key,
		value:     cloneBytes(value),
		expiresAt: expiresAt,
	}
	el := c.ll.PushFront(e)
	c.m[key] = el
	c.evictIfNeededLocked()
}

// Delete removes key from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabledLocked() {
		return
	}
	c.deleteLocked(key)
}

func (c *Cache) deleteLocked(key string) {
	if el, ok := c.m[key]; ok {
		c.removeElementLocked(el)
	}
}

// Purge clears all entries.
func (c *Cache) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ll.Init()
	clear(c.m)
}

// Len returns the number of non-expired entries.
// It may lazily remove expired entries.
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabledLocked() {
		return 0
	}
	c.removeExpiredLocked()
	return len(c.m)
}

// Metrics returns a snapshot of cache hit/miss counters.
func (c *Cache) Metrics() Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.metrics
}

// removeExpiredLocked scans LRU order and removes any expired entries.
func (c *Cache) removeExpiredLocked() {
	now := c.clock.Now()
	for el := c.ll.Back(); el != nil; {
		prev := el.Prev()
		e := el.Value.(*entry)
		if !e.expiresAt.IsZero() && !now.Before(e.expiresAt) {
			c.removeElementLocked(el)
		}
		el = prev
	}
}

func (c *Cache) isExpiredLocked(e *entry) bool {
	if e.expiresAt.IsZero() {
		return false
	}
	now := c.clock.Now()
	return !now.Before(e.expiresAt)
}

func (c *Cache) evictIfNeededLocked() {
	// capacity might be changed in future; handle gracefully.
	cap := c.cfg.Capacity
	if cap <= 0 {
		return
	}

	// Opportunistic cleanup of expired entries first.
	c.removeExpiredLocked()

	for len(c.m) > cap {
		el := c.ll.Back()
		if el == nil {
			return
		}
		c.removeElementLocked(el)
	}
}

func (c *Cache) removeElementLocked(el *list.Element) {
	e := el.Value.(*entry)
	delete(c.m, e.key)
	c.ll.Remove(el)
}

func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp
}

// KeyBuilder helps create stable cache keys.
//
// This is intentionally minimal and does not assume any specific caching target.
// Callers can include provider/model/projectPath/etc. and a content hash.
//
// Example:
//
//	kb := cache.NewKeyBuilder("prompt")
//	key := kb.Add("provider", "openai").Add("model", "gpt-4o").AddHash("system_prompt", prompt).String()
//
// Output format is deterministic.
type KeyBuilder struct {
	prefix string
	parts  []string
}

func NewKeyBuilder(prefix string) *KeyBuilder {
	return &KeyBuilder{prefix: prefix}
}

func (k *KeyBuilder) Add(name, value string) *KeyBuilder {
	k.parts = append(k.parts, fmt.Sprintf("%s=%s", name, value))
	return k
}

func (k *KeyBuilder) AddHash(name string, content string) *KeyBuilder {
	h := sha256.Sum256([]byte(content))
	k.parts = append(k.parts, fmt.Sprintf("%s=sha256:%s", name, hex.EncodeToString(h[:])))
	return k
}

func (k *KeyBuilder) String() string {
	if len(k.parts) == 0 {
		return k.prefix
	}
	// Simple join without importing strings (keep deps minimal).
	out := k.prefix
	for _, p := range k.parts {
		out += "|" + p
	}
	return out
}
