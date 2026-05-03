package cacheutil

import "testing"

func TestLRU_EvictsOldest(t *testing.T) {
	cache := NewLRU[string, int](2)
	cache.Set("a", 1)
	cache.Set("b", 2)
	if _, ok := cache.Get("a"); !ok {
		t.Fatal("Get(a) should hit")
	}
	cache.Set("c", 3)

	if _, ok := cache.Peek("b"); ok {
		t.Fatal("b should be evicted as LRU")
	}
	if _, ok := cache.Peek("a"); !ok {
		t.Fatal("a should remain")
	}
	if _, ok := cache.Peek("c"); !ok {
		t.Fatal("c should remain")
	}
}

func TestLRU_Clear(t *testing.T) {
	cache := NewLRU[string, int](2)
	cache.Set("x", 1)
	cache.Clear()
	if cache.Len() != 0 {
		t.Fatalf("Len() = %d, want 0", cache.Len())
	}
}
