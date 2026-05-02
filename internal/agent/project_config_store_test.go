package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestProjectConfigStore_Load_ReusesCacheUntilFileChanges(t *testing.T) {
	workspace := withTempWorkdir(t)
	configPath := filepath.Join(workspace, "xelyon.yaml")
	if err := os.WriteFile(configPath, []byte("context: first\nrules:\n  - rule-a\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	oldLoader := loadProjectConfigFromDisk
	defer func() { loadProjectConfigFromDisk = oldLoader }()

	loadCalls := 0
	loadProjectConfigFromDisk = func() *config.ProjectConfig {
		loadCalls++
		return oldLoader()
	}

	store := NewProjectConfigStore()
	first := store.Load()
	second := store.Load()
	if loadCalls != 1 {
		t.Fatalf("expected second load to hit cache, loadCalls=%d", loadCalls)
	}
	if first == nil || second == nil {
		t.Fatalf("store should load project config, first=%v second=%v", first, second)
	}

	first.Context = "mutated"
	first.Rules[0] = "mutated-rule"
	third := store.Load()
	if third.Context == "mutated" || third.Rules[0] == "mutated-rule" {
		t.Fatalf("cache should return cloned config, got %#v", third)
	}

	time.Sleep(2 * time.Millisecond)
	if err := os.WriteFile(configPath, []byte("context: updated\nrules:\n  - rule-b\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(update) error = %v", err)
	}

	updated := store.Load()
	if loadCalls != 2 {
		t.Fatalf("expected reload after file change, loadCalls=%d", loadCalls)
	}
	if updated == nil || updated.Context != "updated" {
		t.Fatalf("updated config context = %#v, want updated", updated)
	}
}

func TestProjectConfigStore_Clear_InvalidatesCache(t *testing.T) {
	workspace := withTempWorkdir(t)
	configPath := filepath.Join(workspace, "xelyon.yaml")
	if err := os.WriteFile(configPath, []byte("context: first\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	oldLoader := loadProjectConfigFromDisk
	defer func() { loadProjectConfigFromDisk = oldLoader }()

	loadCalls := 0
	loadProjectConfigFromDisk = func() *config.ProjectConfig {
		loadCalls++
		return oldLoader()
	}

	store := NewProjectConfigStore()
	_ = store.Load()
	_ = store.Load()
	if loadCalls != 1 {
		t.Fatalf("expected cache hit before clear, loadCalls=%d", loadCalls)
	}

	store.Clear()
	_ = store.Load()
	if loadCalls != 2 {
		t.Fatalf("expected reload after clear, loadCalls=%d", loadCalls)
	}
}
