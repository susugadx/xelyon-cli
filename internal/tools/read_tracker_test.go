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

// --- 行範囲テスト ---

func TestReadTracker_MarkReadRange(t *testing.T) {
	rt := NewReadTracker()
	rt.MarkReadRange("/tmp/file.go", 51, 88)

	// 開始行
	if !rt.IsReadLine("/tmp/file.go", 51) {
		t.Error("Start line 51 should be read")
	}
	// 範囲内
	if !rt.IsReadLine("/tmp/file.go", 58) {
		t.Error("Line 58 (within range) should be read")
	}
	// 終了行
	if !rt.IsReadLine("/tmp/file.go", 88) {
		t.Error("End line 88 should be read")
	}
	// 範囲外・直前
	if rt.IsReadLine("/tmp/file.go", 50) {
		t.Error("Line 50 (before range) should NOT be read")
	}
	// 範囲外・直後
	if rt.IsReadLine("/tmp/file.go", 89) {
		t.Error("Line 89 (after range) should NOT be read")
	}
	// ファイル全体は未読
	if rt.IsRead("/tmp/file.go") {
		t.Error("File should NOT be fully marked as read (only range)")
	}
}

func TestReadTracker_MergeRanges(t *testing.T) {
	rt := NewReadTracker()
	rt.MarkReadRange("/tmp/file.go", 51, 88)
	rt.MarkReadRange("/tmp/file.go", 80, 120)

	// マージ後 51-120
	if !rt.IsReadLine("/tmp/file.go", 100) {
		t.Error("Line 100 should be read after merge (51-88 + 80-120 → 51-120)")
	}
	if !rt.IsReadRange("/tmp/file.go", 51, 120) {
		t.Error("Range 51-120 should be covered after merge")
	}
	if rt.IsReadRange("/tmp/file.go", 51, 121) {
		t.Error("Range 51-121 should NOT be covered (only up to 120)")
	}
}

func TestReadTracker_FullFileOverride(t *testing.T) {
	rt := NewReadTracker()
	rt.MarkReadRange("/tmp/file.go", 51, 88)

	// 全体既読前
	if rt.IsReadLine("/tmp/file.go", 120) {
		t.Error("Line 120 should NOT be read before MarkRead")
	}

	// MarkRead で全体既読化
	rt.MarkRead("/tmp/file.go")

	// 全体既読が優先
	if !rt.IsReadLine("/tmp/file.go", 120) {
		t.Error("Line 120 should be read after MarkRead (full file override)")
	}
	if !rt.IsReadRange("/tmp/file.go", 1, 9999) {
		t.Error("Any range should be covered after MarkRead (full file override)")
	}
}

func TestReadTracker_InvalidateClearsBoth(t *testing.T) {
	rt := NewReadTracker()
	rt.MarkRead("/tmp/file.go")
	rt.MarkReadRange("/tmp/file.go", 10, 20)

	rt.InvalidateFile("/tmp/file.go")

	if rt.IsRead("/tmp/file.go") {
		t.Error("After InvalidateFile, IsRead should be false")
	}
	if rt.IsReadLine("/tmp/file.go", 15) {
		t.Error("After InvalidateFile, IsReadLine should be false")
	}
}

func TestReadTracker_AdjacentMerge(t *testing.T) {
	rt := NewReadTracker()
	rt.MarkReadRange("/tmp/file.go", 10, 20)
	rt.MarkReadRange("/tmp/file.go", 21, 30)

	// 隣接マージ: 10-20 + 21-30 → 10-30
	if !rt.IsReadRange("/tmp/file.go", 10, 30) {
		t.Error("Adjacent ranges should be merged (10-20 + 21-30 → 10-30)")
	}

	// 内部確認: ranges のスライス長が 1 であること
	rt.mu.RLock()
	rangeCount := len(rt.ranges["/tmp/file.go"])
	rt.mu.RUnlock()
	if rangeCount != 1 {
		t.Errorf("Expected 1 merged range, got %d", rangeCount)
	}
}
