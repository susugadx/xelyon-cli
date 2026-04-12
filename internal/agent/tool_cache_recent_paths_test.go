package agent

import (
	"reflect"
	"testing"
	"time"
)

func TestToolCache_RecentFilePaths(t *testing.T) {
	base := time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)
	cache := &ToolCache{
		files: map[string]cacheEntry{
			"older.go":  {AccessedAt: base.Add(-2 * time.Minute)},
			"newer.go":  {AccessedAt: base.Add(-1 * time.Minute)},
			"newest.go": {AccessedAt: base},
		},
	}

	got := cache.RecentFilePaths(2)
	want := []string{"newest.go", "newer.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RecentFilePaths() = %v, want %v", got, want)
	}
}

func TestToolCache_RecentFilePaths_NilOrNonPositiveLimit(t *testing.T) {
	var nilCache *ToolCache
	if got := nilCache.RecentFilePaths(3); got != nil {
		t.Fatalf("nil cache RecentFilePaths() = %v, want nil", got)
	}

	cache := &ToolCache{files: map[string]cacheEntry{
		"main.go": {AccessedAt: time.Now()},
	}}
	if got := cache.RecentFilePaths(0); got != nil {
		t.Fatalf("RecentFilePaths(0) = %v, want nil", got)
	}
}
