package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateProjectConfigTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	if err := CreateProjectConfigTemplate("", false); err != nil {
		t.Fatalf("CreateProjectConfigTemplate() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(tmpDir, "xelyon.yaml"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), filepath.Base(tmpDir)+" - Project Configuration") {
		t.Fatalf("template does not include project name:\n%s", string(data))
	}

	if err := CreateProjectConfigTemplate("", false); !errors.Is(err, ErrProjectConfigExists) {
		t.Fatalf("CreateProjectConfigTemplate(existing) error = %v, want ErrProjectConfigExists", err)
	}
}

func TestCloneProjectConfigDeepCopiesSlices(t *testing.T) {
	original := &ProjectConfig{
		Rules: []string{"rule"},
		Conditional: []ProjectConditionalBlock{
			{Name: "go", Paths: []string{"*.go"}, Rules: []string{"test"}},
		},
		Ignore: ProjectIgnoreConfig{Patterns: []string{"dist"}},
		FinalChecks: &FinalChecksConfig{
			Commands: []string{"go test ./..."},
			Timeout:  60,
		},
	}

	clone := CloneProjectConfig(original)
	clone.Rules[0] = "changed"
	clone.Conditional[0].Paths[0] = "*.ts"
	clone.Conditional[0].Rules[0] = "lint"
	clone.Ignore.Patterns[0] = "node_modules"
	clone.FinalChecks.Commands[0] = "make test"

	if original.Rules[0] != "rule" {
		t.Fatalf("Rules shared backing array: %#v", original.Rules)
	}
	if original.Conditional[0].Paths[0] != "*.go" {
		t.Fatalf("Conditional.Paths shared backing array: %#v", original.Conditional[0].Paths)
	}
	if original.Conditional[0].Rules[0] != "test" {
		t.Fatalf("Conditional.Rules shared backing array: %#v", original.Conditional[0].Rules)
	}
	if original.Ignore.Patterns[0] != "dist" {
		t.Fatalf("Ignore.Patterns shared backing array: %#v", original.Ignore.Patterns)
	}
	if original.FinalChecks.Commands[0] != "go test ./..." {
		t.Fatalf("FinalChecks.Commands shared backing array: %#v", original.FinalChecks.Commands)
	}
}
