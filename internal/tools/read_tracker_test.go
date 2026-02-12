package tools

import (
	"sync"
	"testing"
)

func TestReadTracker_MarkAndIsRead(t *testing.T) {
	rt := NewReadTracker()
	rt.MarkRead("/tmp/test.go")

	if !rt.IsRead("/tmp/test.go") {
		t.Error("Expected /tmp/test.go to be marked as read")
	}
	if rt.IsRead("/tmp/other.go") {
		t.Error("Expected /tmp/other.go to NOT be marked as read")
	}
}

func TestReadTracker_IsRead_Unread(t *testing.T) {
	rt := NewReadTracker()

	if rt.IsRead("/some/unread/file.go") {
		t.Error("New tracker should not have any paths marked as read")
	}
}

func TestReadTracker_Reset(t *testing.T) {
	rt := NewReadTracker()
	rt.MarkRead("/tmp/test.go")
	rt.MarkRead("/tmp/other.go")

	rt.Reset()

	if rt.IsRead("/tmp/test.go") {
		t.Error("After Reset, /tmp/test.go should not be marked as read")
	}
	if rt.IsRead("/tmp/other.go") {
		t.Error("After Reset, /tmp/other.go should not be marked as read")
	}
}

func TestReadTracker_MultiplePaths(t *testing.T) {
	rt := NewReadTracker()
	paths := []string{"/a/b.go", "/c/d.go", "/e/f.go"}
	for _, p := range paths {
		rt.MarkRead(p)
	}
	for _, p := range paths {
		if !rt.IsRead(p) {
			t.Errorf("Expected %s to be marked as read", p)
		}
	}
}

func TestReadTracker_ConcurrentAccess(t *testing.T) {
	rt := NewReadTracker()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			path := "/tmp/concurrent_" + string(rune('A'+n%26)) + ".go"
			rt.MarkRead(path)
			_ = rt.IsRead(path)
		}(i)
	}
	wg.Wait()
}

func TestGlobalReadTracker_Initialized(t *testing.T) {
	if GlobalReadTracker == nil {
		t.Fatal("GlobalReadTracker should be initialized at package level")
	}
}
