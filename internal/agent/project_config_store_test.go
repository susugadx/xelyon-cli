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

	loadCalls := 0
	store := NewProjectConfigStoreWithDeps(defaultProjectConfigStoreMaxEntries, func(cwd string) *config.ProjectConfig {
		loadCalls++
		return config.LoadProjectConfigForDir(cwd)
	})
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

	loadCalls := 0
	store := NewProjectConfigStoreWithDeps(defaultProjectConfigStoreMaxEntries, func(cwd string) *config.ProjectConfig {
		loadCalls++
		return config.LoadProjectConfigForDir(cwd)
	})
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

func TestProjectConfigStore_LoadForCWD_UsesRequestedWorkspace(t *testing.T) {
	processCWD := withTempWorkdir(t)
	targetCWD := t.TempDir()

	if err := os.WriteFile(filepath.Join(processCWD, "xelyon.yaml"), []byte("context: process\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(process) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetCWD, "xelyon.yaml"), []byte("context: target\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}

	loadCalls := 0
	store := NewProjectConfigStoreWithDeps(defaultProjectConfigStoreMaxEntries, func(cwd string) *config.ProjectConfig {
		loadCalls++
		return config.LoadProjectConfigForDir(cwd)
	})

	targetCfg := store.LoadForCWD(targetCWD)
	if targetCfg == nil || targetCfg.Context != "target" {
		t.Fatalf("LoadForCWD(target) = %#v, want context=target", targetCfg)
	}
	if loadCalls != 1 {
		t.Fatalf("loadCalls after first target load = %d, want 1", loadCalls)
	}

	targetCached := store.LoadForCWD(targetCWD)
	if targetCached == nil || targetCached.Context != "target" {
		t.Fatalf("LoadForCWD(target) cached = %#v, want context=target", targetCached)
	}
	if loadCalls != 1 {
		t.Fatalf("loadCalls should remain 1 on cache hit, got %d", loadCalls)
	}

	processCfg := store.Load()
	if processCfg == nil || processCfg.Context != "process" {
		t.Fatalf("Load() = %#v, want context=process", processCfg)
	}
	if loadCalls != 2 {
		t.Fatalf("loadCalls after loading process cwd = %d, want 2", loadCalls)
	}
}
