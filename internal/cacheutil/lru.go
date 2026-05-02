package cacheutil

type lruEntry[V any] struct {
	value      V
	lastAccess uint64
}

// LRU は単純な clock-based LRU キャッシュ。
// 呼び出し側が必要なら外側で排他制御する。
type LRU[K comparable, V any] struct {
	entries map[K]lruEntry[V]
	max     int
	clock   uint64
}

// NewLRU は最大エントリ数付き LRU を初期化する。
// max<=0 の場合は既定値 1 を使う。
func NewLRU[K comparable, V any](max int) *LRU[K, V] {
	if max <= 0 {
		max = 1
	}
	return &LRU[K, V]{
		entries: make(map[K]lruEntry[V]),
		max:     max,
	}
}

// Peek は touch せずに値を返す。
func (c *LRU[K, V]) Peek(key K) (V, bool) {
	if c == nil || c.entries == nil {
		var zero V
		return zero, false
	}
	entry, ok := c.entries[key]
	if !ok {
		var zero V
		return zero, false
	}
	return entry.value, true
}

// Get は値を返しつつ touch する。
func (c *LRU[K, V]) Get(key K) (V, bool) {
	if c == nil || c.entries == nil {
		var zero V
		return zero, false
	}
	entry, ok := c.entries[key]
	if !ok {
		var zero V
		return zero, false
	}
	entry.lastAccess = c.nextClock()
	c.entries[key] = entry
	return entry.value, true
}

// Set は値を保存し、必要なら古い要素を eviction する。
func (c *LRU[K, V]) Set(key K, value V) {
	if c == nil {
		return
	}
	if c.entries == nil {
		c.entries = make(map[K]lruEntry[V])
	}
	c.entries[key] = lruEntry[V]{
		value:      value,
		lastAccess: c.nextClock(),
	}
	c.evictIfNeeded()
}

// Delete はキーを削除する。
func (c *LRU[K, V]) Delete(key K) {
	if c == nil || c.entries == nil {
		return
	}
	delete(c.entries, key)
}

// Clear は全エントリを削除する。
func (c *LRU[K, V]) Clear() {
	if c == nil {
		return
	}
	c.entries = make(map[K]lruEntry[V])
	c.clock = 0
}

// Len はキャッシュサイズを返す。
func (c *LRU[K, V]) Len() int {
	if c == nil || c.entries == nil {
		return 0
	}
	return len(c.entries)
}

func (c *LRU[K, V]) nextClock() uint64 {
	if c == nil {
		return 0
	}
	c.clock++
	return c.clock
}

func (c *LRU[K, V]) evictIfNeeded() {
	if c == nil || c.max <= 0 || len(c.entries) <= c.max {
		return
	}
	var oldestKey K
	var oldestAccess uint64
	first := true
	for key, entry := range c.entries {
		if first || entry.lastAccess < oldestAccess {
			oldestKey = key
			oldestAccess = entry.lastAccess
			first = false
		}
	}
	if !first {
		delete(c.entries, oldestKey)
	}
}
