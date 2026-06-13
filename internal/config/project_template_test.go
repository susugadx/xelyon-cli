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
	if !strings.Contains(string(data), "XELYON repo config for "+filepath.Base(tmpDir)) {
		t.Fatalf("template does not include project name:\n%s", string(data))
	}
	if strings.Contains(string(data), "AI 用コンテキスト") || strings.Contains(string(data), "context:") || strings.Contains(string(data), "rules:") {
		t.Fatalf("template should not recommend legacy context/rules guidance:\n%s", string(data))
	}

	if err := CreateProjectConfigTemplate("", false); !errors.Is(err, ErrProjectConfigExists) {
		t.Fatalf("CreateProjectConfigTemplate(existing) error = %v, want ErrProjectConfigExists", err)
	}
}

func TestCreateProjectAgentInstructionsTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "AGENTS.md")

	if err := CreateProjectAgentInstructionsTemplate(path); err != nil {
		t.Fatalf("CreateProjectAgentInstructionsTemplate() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "# AGENTS.md") {
		t.Fatalf("template missing AGENTS heading:\n%s", string(data))
	}

	if err := CreateProjectAgentInstructionsTemplate(path); !errors.Is(err, ErrProjectAgentInstructionsExists) {
		t.Fatalf("CreateProjectAgentInstructionsTemplate(existing) error = %v, want ErrProjectAgentInstructionsExists", err)
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
		Experimental: ProjectExperimentalConfig{
			ProviderHistoryReduction: ProjectProviderHistoryReductionConfig{
				Mode:                ProjectProviderHistoryReductionModeDryRun,
				RehydrateContext:    true,
				RehydrateContextSet: true,
			},
		},
	}

	clone := CloneProjectConfig(original)
	clone.Rules[0] = "changed"
	clone.Conditional[0].Paths[0] = "*.ts"
	clone.Conditional[0].Rules[0] = "lint"
	clone.Ignore.Patterns[0] = "node_modules"
	clone.FinalChecks.Commands[0] = "make test"
	clone.Experimental.ProviderHistoryReduction.Mode = ProjectProviderHistoryReductionModeApply
	clone.Experimental.ProviderHistoryReduction.RehydrateContext = false

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
	if original.Experimental.ProviderHistoryReduction.Mode != ProjectProviderHistoryReductionModeDryRun {
		t.Fatalf("Experimental.ProviderHistoryReduction.Mode = %q, want dry_run", original.Experimental.ProviderHistoryReduction.Mode)
	}
	if !bool(original.Experimental.ProviderHistoryReduction.RehydrateContext) {
		t.Fatalf("Experimental.ProviderHistoryReduction.RehydrateContext = false, want true")
	}
}
