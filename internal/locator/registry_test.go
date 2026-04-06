package locator

import (
	"fmt"
	"sync"
	"testing"
)

func TestRegisterAndResolve(t *testing.T) {
	reg := NewRegistry()

	loc := Location{FilePath: "cmd/main.go", Line: 42, EndLine: 55, Name: "func main"}
	id := reg.Register(loc)

	if id != "[L1]" {
		t.Errorf("expected [L1], got %s", id)
	}

	got, ok := reg.Resolve(id)
	if !ok {
		t.Fatal("Resolve returned false")
	}
	if got.FilePath != loc.FilePath || got.Line != loc.Line || got.EndLine != loc.EndLine || got.Name != loc.Name {
		t.Errorf("Resolve mismatch: got %+v, want %+v", got, loc)
	}
}

func TestRegisterIdempotent(t *testing.T) {
	reg := NewRegistry()

	loc := Location{FilePath: "pkg/util.go", Line: 10}
	id1 := reg.Register(loc)
	id2 := reg.Register(loc)

	if id1 != id2 {
		t.Errorf("idempotent violation: %s != %s", id1, id2)
	}
}

func TestRegisterDistinctResolvedPathsDoNotDedup(t *testing.T) {
	reg := NewRegistry()

	id1 := reg.Register(Location{FilePath: "target.go", ResolvedPath: "/repo/target.go", Line: 3})
	id2 := reg.Register(Location{FilePath: "target.go", ResolvedPath: "/repo/sub/target.go", Line: 3})

	if id1 == id2 {
		t.Fatalf("expected distinct IDs for same display path with different resolved paths, got %s", id1)
	}
}

func TestRegisterMultipleLocations(t *testing.T) {
	reg := NewRegistry()

	loc1 := Location{FilePath: "a.go", Line: 1}
	loc2 := Location{FilePath: "b.go", Line: 2}
	loc3 := Location{FilePath: "c.go", Line: 3}

	id1 := reg.Register(loc1)
	id2 := reg.Register(loc2)
	id3 := reg.Register(loc3)

	if id1 != "[L1]" || id2 != "[L2]" || id3 != "[L3]" {
		t.Errorf("unexpected IDs: %s, %s, %s", id1, id2, id3)
	}
}

func TestResolveMulti(t *testing.T) {
	reg := NewRegistry()

	loc1 := Location{FilePath: "a.go", Line: 10}
	loc2 := Location{FilePath: "b.go", Line: 20}
	loc3 := Location{FilePath: "c.go", Line: 30}
	reg.Register(loc1)
	reg.Register(loc2)
	reg.Register(loc3)

	locs := reg.ResolveMulti("[L1, L3]")
	if len(locs) != 2 {
		t.Fatalf("expected 2 locations, got %d", len(locs))
	}
	if locs[0].FilePath != "a.go" || locs[1].FilePath != "c.go" {
		t.Errorf("unexpected locations: %+v", locs)
	}
}

func TestResolveMultiWithInvalid(t *testing.T) {
	reg := NewRegistry()

	reg.Register(Location{FilePath: "a.go", Line: 1})

	// L1 は有効、L99 は無効 → L1 のみ返る
	locs := reg.ResolveMulti("[L1, L99]")
	if len(locs) != 1 {
		t.Fatalf("expected 1 location, got %d", len(locs))
	}
	if locs[0].FilePath != "a.go" {
		t.Errorf("unexpected location: %+v", locs[0])
	}
}

func TestResolveNonexistent(t *testing.T) {
	reg := NewRegistry()

	_, ok := reg.Resolve("[L1]")
	if ok {
		t.Error("expected false for nonexistent ID")
	}
}

func TestResolveInvalidFormats(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Location{FilePath: "a.go", Line: 1})

	tests := []string{
		"X5",
		"",
		"[L0]",
		"abc",
		"L",
		"Lx",
		"[]",
	}
	for _, input := range tests {
		_, ok := reg.Resolve(input)
		if ok {
			t.Errorf("expected false for invalid ID %q", input)
		}
	}
}

func TestResolveBracketedAndBare(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Location{FilePath: "a.go", Line: 1})

	// 角括弧付き
	loc1, ok1 := reg.Resolve("[L1]")
	if !ok1 {
		t.Fatal("expected [L1] to resolve")
	}

	// 角括弧なし
	loc2, ok2 := reg.Resolve("L1")
	if !ok2 {
		t.Fatal("expected L1 to resolve")
	}

	if loc1.FilePath != loc2.FilePath {
		t.Error("bracketed and bare should resolve to same location")
	}
}

func TestResolveLowercase(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Location{FilePath: "a.go", Line: 1})

	_, ok := reg.Resolve("l1")
	if !ok {
		t.Error("expected lowercase l1 to resolve")
	}
}

func TestConcurrentRegister(t *testing.T) {
	reg := NewRegistry()

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			loc := Location{FilePath: fmt.Sprintf("file%d.go", n), Line: n}
			reg.Register(loc)
		}(i)
	}
	wg.Wait()

	// エントリ数が100であることを確認
	reg.mu.RLock()
	count := len(reg.entries)
	reg.mu.RUnlock()
	if count != 100 {
		t.Errorf("expected 100 entries, got %d", count)
	}
}

func TestResolveMultiEmpty(t *testing.T) {
	reg := NewRegistry()

	locs := reg.ResolveMulti("")
	if locs != nil {
		t.Errorf("expected nil for empty input, got %v", locs)
	}

	locs = reg.ResolveMulti("[]")
	if locs != nil {
		t.Errorf("expected nil for empty brackets, got %v", locs)
	}
}
