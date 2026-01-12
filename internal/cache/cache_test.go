package cache

import (
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(t time.Time) *fakeClock {
	return &fakeClock{now: t}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestCache_Disabled_NoOp(t *testing.T) {
	clk := newFakeClock(time.Unix(0, 0))
	c := New(Config{Enabled: false, Capacity: 10, DefaultTTL: time.Second}, clk)

	c.Set("k", []byte("v"), 0)
	if _, err := c.Get("k"); err == nil {
		t.Fatalf("expected ErrNotFound, got nil")
	}

	if got := c.Len(); got != 0 {
		t.Fatalf("expected len=0, got %d", got)
	}

	m := c.Metrics()
	if m.Misses == 0 {
		t.Fatalf("expected misses incremented")
	}
}

func TestCache_SetGet_RoundTrip_ClonesBytes(t *testing.T) {
	clk := newFakeClock(time.Unix(0, 0))
	c := New(Config{Enabled: true, Capacity: 10}, clk)

	b := []byte("hello")
	c.Set("k", b, 0)

	// mutate original slice; cache should not be affected
	b[0] = 'H'

	got, err := c.Get("k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("unexpected value: %q", string(got))
	}

	// mutate returned slice; cache should not be affected
	got[0] = 'X'
	got2, err := c.Get("k")
	if err != nil {
		t.Fatalf("Get2: %v", err)
	}
	if string(got2) != "hello" {
		t.Fatalf("unexpected value after mutating return: %q", string(got2))
	}
}

func TestCache_TTL_Expires(t *testing.T) {
	clk := newFakeClock(time.Unix(0, 0))
	c := New(Config{Enabled: true, Capacity: 10}, clk)

	c.Set("k", []byte("v"), 2*time.Second)

	if _, err := c.Get("k"); err != nil {
		t.Fatalf("expected hit before expiry, got %v", err)
	}

	clk.Add(2 * time.Second)
	if _, err := c.Get("k"); err == nil {
		t.Fatalf("expected ErrNotFound after expiry")
	}
}

func TestCache_DefaultTTL_AppliesWhenTTLZero(t *testing.T) {
	clk := newFakeClock(time.Unix(0, 0))
	c := New(Config{Enabled: true, Capacity: 10, DefaultTTL: time.Second}, clk)

	c.Set("k", []byte("v"), 0)
	clk.Add(999 * time.Millisecond)
	if _, err := c.Get("k"); err != nil {
		t.Fatalf("expected hit before default TTL expiry, got %v", err)
	}
	clk.Add(2 * time.Millisecond)
	if _, err := c.Get("k"); err == nil {
		t.Fatalf("expected ErrNotFound after default TTL expiry")
	}
}

func TestCache_DeleteAndPurge(t *testing.T) {
	clk := newFakeClock(time.Unix(0, 0))
	c := New(Config{Enabled: true, Capacity: 10}, clk)

	c.Set("a", []byte("1"), 0)
	c.Set("b", []byte("2"), 0)

	c.Delete("a")
	if _, err := c.Get("a"); err == nil {
		t.Fatalf("expected ErrNotFound after delete")
	}

	if got := c.Len(); got != 1 {
		t.Fatalf("expected len=1, got %d", got)
	}

	c.Purge()
	if got := c.Len(); got != 0 {
		t.Fatalf("expected len=0 after purge, got %d", got)
	}
}

func TestCache_LRU_Eviction(t *testing.T) {
	clk := newFakeClock(time.Unix(0, 0))
	c := New(Config{Enabled: true, Capacity: 2}, clk)

	c.Set("a", []byte("1"), 0)
	c.Set("b", []byte("2"), 0)

	// Touch "a" so "b" becomes LRU
	if _, err := c.Get("a"); err != nil {
		t.Fatalf("Get(a): %v", err)
	}

	c.Set("c", []byte("3"), 0) // should evict "b"

	if _, err := c.Get("b"); err == nil {
		t.Fatalf("expected b to be evicted")
	}
	if _, err := c.Get("a"); err != nil {
		t.Fatalf("expected a to remain, got %v", err)
	}
	if _, err := c.Get("c"); err != nil {
		t.Fatalf("expected c to exist, got %v", err)
	}
}

func TestCache_SetWithNegativeTTL_Deletes(t *testing.T) {
	clk := newFakeClock(time.Unix(0, 0))
	c := New(Config{Enabled: true, Capacity: 10}, clk)

	c.Set("k", []byte("v"), 0)
	c.Set("k", []byte("v2"), -1)
	if _, err := c.Get("k"); err == nil {
		t.Fatalf("expected ErrNotFound after negative TTL delete")
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	clk := newFakeClock(time.Unix(0, 0))
	c := New(Config{Enabled: true, Capacity: 128}, clk)

	var wg sync.WaitGroup
	workers := 50
	iters := 200
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		i := i
		go func() {
			defer wg.Done()
			key := "k"
			if i%2 == 0 {
				key = "k2"
			}
			for j := 0; j < iters; j++ {
				c.Set(key, []byte("v"), 0)
				_, _ = c.Get(key)
				if j%10 == 0 {
					c.Delete(key)
				}
			}
		}()
	}

	wg.Wait()

	// The goal is "no data race / no panic". Basic sanity:
	_ = c.Len()
	_ = c.Metrics()
}

func TestKeyBuilder_Deterministic(t *testing.T) {
	kb := NewKeyBuilder("p").Add("a", "1").Add("b", "2").AddHash("x", "hello")
	k1 := kb.String()
	k2 := NewKeyBuilder("p").Add("a", "1").Add("b", "2").AddHash("x", "hello").String()
	if k1 != k2 {
		t.Fatalf("expected deterministic key, got %q vs %q", k1, k2)
	}
}
