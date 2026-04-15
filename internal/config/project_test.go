package config

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
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

func TestLoadProjectConfigWithFinalChecks(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `context: "test"
rules:
  - "rule 1"
final_checks:
  commands:
    - "go test ./..."
  timeout: 30
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
	if pc.FinalChecks == nil {
		t.Fatal("FinalChecks is nil, want non-nil")
	}
	if len(pc.FinalChecks.Commands) != 1 || pc.FinalChecks.Commands[0] != "go test ./..." {
		t.Errorf("FinalChecks.Commands = %v, want [\"go test ./...\"]", pc.FinalChecks.Commands)
	}
	if pc.FinalChecks.Timeout != 30 {
		t.Errorf("FinalChecks.Timeout = %d, want 30", pc.FinalChecks.Timeout)
	}
}

func TestLoadProjectConfigWithLegacyHooks(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `context: "test"
hooks:
  on_completion:
    - "go test ./..."
  timeout: 45
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
	if pc.FinalChecks == nil {
		t.Fatal("FinalChecks is nil, want non-nil")
	}
	if len(pc.FinalChecks.Commands) != 1 || pc.FinalChecks.Commands[0] != "go test ./..." {
		t.Errorf("FinalChecks.Commands = %v, want [\"go test ./...\"]", pc.FinalChecks.Commands)
	}
	if pc.FinalChecks.Timeout != 45 {
		t.Errorf("FinalChecks.Timeout = %d, want 45", pc.FinalChecks.Timeout)
	}
}

func TestLoadProjectConfig_FinalChecksWinsOverLegacyVerificationAndHooks(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `context: "test"
final_checks:
  commands:
    - "make test"
  timeout: 90
verification:
  commands:
    - "legacy verify"
  timeout: 60
hooks:
  on_completion:
    - "go test ./..."
  timeout: 45
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
	if pc.FinalChecks == nil {
		t.Fatal("FinalChecks is nil, want non-nil")
	}
	if len(pc.FinalChecks.Commands) != 1 || pc.FinalChecks.Commands[0] != "make test" {
		t.Errorf("FinalChecks.Commands = %v, want [\"make test\"]", pc.FinalChecks.Commands)
	}
	if pc.FinalChecks.Timeout != 90 {
		t.Errorf("FinalChecks.Timeout = %d, want 90", pc.FinalChecks.Timeout)
	}
}

func TestLoadProjectConfig_WithConditionalAndIgnore(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `context: "base context"
rules:
  - "base rule"
conditional:
  - name: "Agent"
    paths:
      - "internal/agent/**"
    rules:
      - "agent rule"
    context: "agent context"
ignore:
  patterns:
    - "coverage"
    - "*.gen.go"
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
	if len(pc.Conditional) != 1 {
		t.Fatalf("Conditional length = %d, want 1", len(pc.Conditional))
	}
	if got := pc.Conditional[0].Paths[0]; got != "internal/agent/**" {
		t.Fatalf("Conditional[0].Paths[0] = %q, want %q", got, "internal/agent/**")
	}
	if got := pc.Ignore.Patterns; len(got) != 2 || got[0] != "coverage" || got[1] != "*.gen.go" {
		t.Fatalf("Ignore.Patterns = %v", got)
	}
}

func TestSelectProjectPromptSelection(t *testing.T) {
	pc := &ProjectConfig{
		Context: "base context",
		Rules:   []string{"base rule"},
		Conditional: []ProjectConditionalBlock{
			{
				Name:    "Agent",
				Paths:   []string{"internal/agent/**"},
				Rules:   []string{"agent rule"},
				Context: "agent context",
			},
			{
				Name:    "Config",
				Paths:   []string{"internal/config/**"},
				Rules:   []string{"config rule"},
				Context: "config context",
			},
		},
	}

	got := SelectProjectPromptSelection(pc, "internal/agent/compress.go を直してください")
	if len(got.Rules) != 2 {
		t.Fatalf("Rules length = %d, want 2", len(got.Rules))
	}
	if got.Rules[0] != "base rule" || got.Rules[1] != "agent rule" {
		t.Fatalf("Rules = %v", got.Rules)
	}
	if len(got.Contexts) != 2 {
		t.Fatalf("Contexts length = %d, want 2", len(got.Contexts))
	}
	if got.Contexts[0] != "base context" {
		t.Fatalf("Contexts[0] = %q", got.Contexts[0])
	}
	if got.Contexts[1] != "### Agent\nagent context" {
		t.Fatalf("Contexts[1] = %q", got.Contexts[1])
	}
}

func TestResolveSharedIgnorePatterns(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ListDir.AdditionalIgnoreDirs = []string{"coverage"}
	cfg.ProjectMap.AdditionalIgnoreDirs = []string{"generated"}

	pc := &ProjectConfig{
		Ignore: ProjectIgnoreConfig{
			Patterns: []string{"*.gen.go", "coverage"},
		},
	}

	patterns := ResolveSharedIgnorePatterns(cfg, pc)
	for _, want := range []string{".git", "node_modules", "coverage", "generated", "*.gen.go"} {
		if !slices.Contains(patterns, want) {
			t.Fatalf("ResolveSharedIgnorePatterns() missing %q in %v", want, patterns)
		}
	}
}

func TestExtractReferencedProjectPaths(t *testing.T) {
	input := "internal/agent/compress.go:42 と `docs/config.md` を見てください"
	got := ExtractReferencedProjectPaths(input)
	want := []string{"internal/agent/compress.go", "docs/config.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractReferencedProjectPaths() = %v, want %v", got, want)
	}
}

func TestExtractReferencedProjectPathsForRoot_AbsoluteAndTopLevelFile(t *testing.T) {
	root := t.TempDir()
	absPath := filepath.Join(root, "cmd", "tool", "main.go")
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("all:\n\t@true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	input := absPath + ":12 と Makefile を見てください"
	got := ExtractReferencedProjectPathsForRoot(input, root)
	want := []string{"cmd/tool/main.go", "Makefile"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractReferencedProjectPathsForRoot() = %v, want %v", got, want)
	}
}

func TestSelectProjectPromptSelection_UsesProjectRootForAbsolutePath(t *testing.T) {
	root := t.TempDir()
	absPath := filepath.Join(root, "cmd", "tool", "main.go")
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	pc := &ProjectConfig{
		FilePath: filepath.Join(root, "xelyon.yaml"),
		Rules:    []string{"base rule"},
		Conditional: []ProjectConditionalBlock{
			{
				Name:  "CLI",
				Paths: []string{"cmd/**/*.go"},
				Rules: []string{"cli rule"},
			},
		},
	}

	got := SelectProjectPromptSelection(pc, absPath+":8 を修正してください")
	if !slices.Contains(got.Rules, "cli rule") {
		t.Fatalf("expected cli rule to be selected, got %v", got.Rules)
	}
}

func TestResolveFinalChecks_ProjectFinalChecksPriority(t *testing.T) {
	globalCfg := &Config{
		FinalChecks: FinalChecksConfig{
			Commands: []string{"global verify"},
			Timeout:  600,
		},
	}
	projectCfg := &ProjectConfig{
		FinalChecks: &FinalChecksConfig{
			Commands: []string{"project verify"},
			Timeout:  30,
		},
	}

	resolved := ResolveFinalChecks(globalCfg, projectCfg)
	if resolved == nil {
		t.Fatal("ResolveFinalChecks returned nil")
	}
	if len(resolved.Commands) != 1 || resolved.Commands[0] != "project verify" {
		t.Errorf("expected project final checks, got %v", resolved.Commands)
	}
	if resolved.Timeout != 30 {
		t.Errorf("Timeout = %d, want 30", resolved.Timeout)
	}
}

func TestResolveFinalChecks_GlobalFallback(t *testing.T) {
	globalCfg := &Config{
		FinalChecks: FinalChecksConfig{
			Commands: []string{"global verify"},
			Timeout:  600,
		},
	}
	projectCfg := &ProjectConfig{
		// FinalChecks is nil → should fall back to global
	}

	resolved := ResolveFinalChecks(globalCfg, projectCfg)
	if resolved == nil {
		t.Fatal("ResolveFinalChecks returned nil")
	}
	if len(resolved.Commands) != 1 || resolved.Commands[0] != "global verify" {
		t.Errorf("expected global final checks fallback, got %v", resolved.Commands)
	}
}

func TestResolveFinalChecks_BothEmpty(t *testing.T) {
	globalCfg := &Config{
		FinalChecks: FinalChecksConfig{}, // no commands
	}
	projectCfg := &ProjectConfig{} // no final checks

	resolved := ResolveFinalChecks(globalCfg, projectCfg)
	if resolved != nil {
		t.Errorf("expected nil when both are empty, got %v", resolved)
	}
}

func TestResolveFinalChecks_NilProject(t *testing.T) {
	globalCfg := &Config{
		FinalChecks: FinalChecksConfig{
			Commands: []string{"global verify"},
		},
	}

	resolved := ResolveFinalChecks(globalCfg, nil)
	if resolved == nil {
		t.Fatal("ResolveFinalChecks returned nil with nil project but global final checks")
	}
	if resolved.Commands[0] != "global verify" {
		t.Errorf("expected global final checks, got %v", resolved.Commands)
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

func TestSaveProjectConfig_WithFinalChecks(t *testing.T) {
	dir := t.TempDir()
	pc := &ProjectConfig{
		Context: "test",
		Rules:   []string{"rule 1"},
		FinalChecks: &FinalChecksConfig{
			Commands: []string{"go test ./..."},
			Timeout:  120,
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
	if loaded.FinalChecks == nil {
		t.Fatal("FinalChecks is nil after save/load")
	}
	if len(loaded.FinalChecks.Commands) != 1 || loaded.FinalChecks.Commands[0] != "go test ./..." {
		t.Errorf("FinalChecks.Commands = %v", loaded.FinalChecks.Commands)
	}
	if loaded.FinalChecks.Timeout != 120 {
		t.Errorf("FinalChecks.Timeout = %d, want 120", loaded.FinalChecks.Timeout)
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
