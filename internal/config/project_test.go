package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectConfigFromYAML(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `context: "Test project context"
rules:
  - "Run tests before commit"
  - "Use go fmt"
`
	if err := os.WriteFile(filepath.Join(dir, "xelyon.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to temp dir
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	pc := LoadProjectConfig()
	if pc == nil {
		t.Fatal("LoadProjectConfig() returned nil, want non-nil")
	}
	if pc.Context != "Test project context" {
		t.Errorf("Context = %q, want %q", pc.Context, "Test project context")
	}
	if len(pc.Rules) != 2 {
		t.Fatalf("Rules length = %d, want 2", len(pc.Rules))
	}
	if pc.Rules[0] != "Run tests before commit" {
		t.Errorf("Rules[0] = %q, want %q", pc.Rules[0], "Run tests before commit")
	}
	if pc.FilePath != filepath.Join(dir, "xelyon.yaml") {
		t.Errorf("FilePath = %q, want %q", pc.FilePath, filepath.Join(dir, "xelyon.yaml"))
	}
}

func TestLoadProjectConfigNeitherExists(t *testing.T) {
	dir := t.TempDir()

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	pc := LoadProjectConfig()
	if pc != nil {
		t.Errorf("expected nil when no xelyon.yaml exists, got %+v", pc)
	}
}

func TestLoadProjectConfigXELYONMDIgnored(t *testing.T) {
	dir := t.TempDir()
	// XELYON.md のみ存在 → 無視される
	if err := os.WriteFile(filepath.Join(dir, "XELYON.md"), []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	pc := LoadProjectConfig()
	if pc != nil {
		t.Errorf("expected nil when only XELYON.md exists (should be ignored), got %+v", pc)
	}
}

func TestLoadProjectConfigInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "xelyon.yaml"), []byte(":\ninvalid: [yaml\n"), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	pc := LoadProjectConfig()
	if pc != nil {
		t.Error("expected nil for invalid YAML, got non-nil")
	}
}

func TestLoadProjectConfigWithHooks(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `context: "test"
rules:
  - "rule 1"
hooks:
  on_completion:
    - "go test ./..."
  timeout: 30
  max_retry: 5
`
	if err := os.WriteFile(filepath.Join(dir, "xelyon.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	pc := LoadProjectConfig()
	if pc == nil {
		t.Fatal("LoadProjectConfig() returned nil")
	}
	if pc.Hooks == nil {
		t.Fatal("Hooks is nil, want non-nil")
	}
	if len(pc.Hooks.OnCompletion) != 1 || pc.Hooks.OnCompletion[0] != "go test ./..." {
		t.Errorf("Hooks.OnCompletion = %v, want [\"go test ./...\"]", pc.Hooks.OnCompletion)
	}
	if pc.Hooks.Timeout != 30 {
		t.Errorf("Hooks.Timeout = %d, want 30", pc.Hooks.Timeout)
	}
	if pc.Hooks.MaxRetry != 5 {
		t.Errorf("Hooks.MaxRetry = %d, want 5", pc.Hooks.MaxRetry)
	}
}

func TestResolveHooks_ProjectHooksPriority(t *testing.T) {
	globalCfg := &Config{
		Hooks: HooksConfig{
			OnCompletion: []string{"global hook"},
			Timeout:      60,
		},
	}
	projectCfg := &ProjectConfig{
		Hooks: &HooksConfig{
			OnCompletion: []string{"project hook"},
			Timeout:      30,
		},
	}

	resolved := ResolveHooks(globalCfg, projectCfg)
	if resolved == nil {
		t.Fatal("ResolveHooks returned nil")
	}
	if len(resolved.OnCompletion) != 1 || resolved.OnCompletion[0] != "project hook" {
		t.Errorf("expected project hook, got %v", resolved.OnCompletion)
	}
	if resolved.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", resolved.Timeout)
	}
}

func TestResolveHooks_GlobalFallback(t *testing.T) {
	globalCfg := &Config{
		Hooks: HooksConfig{
			OnCompletion: []string{"global hook"},
			Timeout:      60,
		},
	}
	projectCfg := &ProjectConfig{
		// Hooks is nil → should fall back to global
	}

	resolved := ResolveHooks(globalCfg, projectCfg)
	if resolved == nil {
		t.Fatal("ResolveHooks returned nil")
	}
	if len(resolved.OnCompletion) != 1 || resolved.OnCompletion[0] != "global hook" {
		t.Errorf("expected global hook fallback, got %v", resolved.OnCompletion)
	}
}

func TestResolveHooks_BothEmpty(t *testing.T) {
	globalCfg := &Config{
		Hooks: HooksConfig{}, // no on_completion
	}
	projectCfg := &ProjectConfig{} // no hooks

	resolved := ResolveHooks(globalCfg, projectCfg)
	if resolved != nil {
		t.Errorf("expected nil when both are empty, got %v", resolved)
	}
}

func TestResolveHooks_NilProject(t *testing.T) {
	globalCfg := &Config{
		Hooks: HooksConfig{
			OnCompletion: []string{"global hook"},
		},
	}

	resolved := ResolveHooks(globalCfg, nil)
	if resolved == nil {
		t.Fatal("ResolveHooks returned nil with nil project but global hooks")
	}
	if resolved.OnCompletion[0] != "global hook" {
		t.Errorf("expected global hook, got %v", resolved.OnCompletion)
	}
}

func TestSaveProjectConfig(t *testing.T) {
	dir := t.TempDir()
	pc := &ProjectConfig{
		Context:  "Test project",
		Rules:    []string{"rule 1", "rule 2"},
		FilePath: filepath.Join(dir, "xelyon.yaml"),
	}

	if err := SaveProjectConfig(pc); err != nil {
		t.Fatalf("SaveProjectConfig() error: %v", err)
	}

	// 再読み込みして一致確認
	loaded, err := loadProjectConfigFromYAML(pc.FilePath)
	if err != nil {
		t.Fatalf("loadProjectConfigFromYAML() error: %v", err)
	}
	if loaded.Context != "Test project" {
		t.Errorf("Context = %q, want %q", loaded.Context, "Test project")
	}
	if len(loaded.Rules) != 2 || loaded.Rules[0] != "rule 1" {
		t.Errorf("Rules = %v, want [rule 1, rule 2]", loaded.Rules)
	}
}

func TestSaveProjectConfig_WithHooks(t *testing.T) {
	dir := t.TempDir()
	pc := &ProjectConfig{
		Context: "test",
		Rules:   []string{"rule 1"},
		Hooks: &HooksConfig{
			OnCompletion: []string{"go test ./..."},
			Timeout:      120,
			MaxRetry:     5,
		},
		FilePath: filepath.Join(dir, "xelyon.yaml"),
	}

	if err := SaveProjectConfig(pc); err != nil {
		t.Fatalf("SaveProjectConfig() error: %v", err)
	}

	loaded, err := loadProjectConfigFromYAML(pc.FilePath)
	if err != nil {
		t.Fatalf("loadProjectConfigFromYAML() error: %v", err)
	}
	if loaded.Hooks == nil {
		t.Fatal("Hooks is nil after save/load")
	}
	if len(loaded.Hooks.OnCompletion) != 1 || loaded.Hooks.OnCompletion[0] != "go test ./..." {
		t.Errorf("Hooks.OnCompletion = %v", loaded.Hooks.OnCompletion)
	}
	if loaded.Hooks.Timeout != 120 {
		t.Errorf("Hooks.Timeout = %d, want 120", loaded.Hooks.Timeout)
	}
	if loaded.Hooks.MaxRetry != 5 {
		t.Errorf("Hooks.MaxRetry = %d, want 5", loaded.Hooks.MaxRetry)
	}
}

func TestSaveProjectConfig_DefaultPath(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	_ = os.Chdir(dir)

	pc := &ProjectConfig{
		Context: "default path test",
		Rules:   []string{"rule"},
	}

	if err := SaveProjectConfig(pc); err != nil {
		t.Fatalf("SaveProjectConfig() error: %v", err)
	}

	// ./xelyon.yaml に保存されていることを確認
	if _, err := os.Stat(filepath.Join(dir, "xelyon.yaml")); err != nil {
		t.Fatalf("xelyon.yaml not created: %v", err)
	}

	loaded, err := loadProjectConfigFromYAML(filepath.Join(dir, "xelyon.yaml"))
	if err != nil {
		t.Fatalf("loadProjectConfigFromYAML() error: %v", err)
	}
	if loaded.Context != "default path test" {
		t.Errorf("Context = %q, want %q", loaded.Context, "default path test")
	}
}

func TestFindFileUpward(t *testing.T) {
	// Create nested directories: parent/child/grandchild
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	grandchild := filepath.Join(child, "grandchild")
	os.MkdirAll(grandchild, 0755)

	// Place file in parent
	if err := os.WriteFile(filepath.Join(parent, "target.txt"), []byte("found"), 0644); err != nil {
		t.Fatal(err)
	}

	// Search from grandchild
	result := findFileUpward(grandchild, "target.txt")
	expected := filepath.Join(parent, "target.txt")
	if result != expected {
		t.Errorf("findFileUpward() = %q, want %q", result, expected)
	}

	// Search for non-existent file
	result = findFileUpward(grandchild, "nonexistent.txt")
	if result != "" {
		t.Errorf("findFileUpward() = %q, want empty for non-existent file", result)
	}
}
