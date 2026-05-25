package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProjectConfigWithExperimentalProviderHistoryReductionModes(t *testing.T) {
	for _, mode := range []ProjectProviderHistoryReductionMode{
		ProjectProviderHistoryReductionModeOff,
		ProjectProviderHistoryReductionModeDryRun,
		ProjectProviderHistoryReductionModeApply,
		ProjectProviderHistoryReductionModeAuto,
	} {
		t.Run(string(mode), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "xelyon.yaml")
			yamlContent := "experimental:\n  provider_history_reduction:\n    mode: " + string(mode) + "\n"
			if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
				t.Fatal(err)
			}

			pc, err := loadProjectConfigFromYAML(path)
			if err != nil {
				t.Fatalf("loadProjectConfigFromYAML() error = %v", err)
			}
			if got := pc.Experimental.ProviderHistoryReduction.Mode; got != mode {
				t.Fatalf("provider_history_reduction.mode = %q, want %q", got, mode)
			}
		})
	}
}

func TestLoadProjectConfigWithInvalidExperimentalProviderHistoryReductionMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "xelyon.yaml")
	if err := os.WriteFile(path, []byte("experimental:\n  provider_history_reduction:\n    mode: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := loadProjectConfigFromYAML(path)
	if err == nil {
		t.Fatal("loadProjectConfigFromYAML() error = nil, want invalid mode error")
	}
	want := `invalid provider history reduction mode "x" (expected: off, dry_run, apply, auto)`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want containing %q", err.Error(), want)
	}
}

func TestLoadProjectConfigWithInvalidExperimentalProviderHistoryRehydrateContext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "xelyon.yaml")
	if err := os.WriteFile(path, []byte("experimental:\n  provider_history_reduction:\n    rehydrate_context: maybe\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := loadProjectConfigFromYAML(path)
	if err == nil {
		t.Fatal("loadProjectConfigFromYAML() error = nil, want invalid rehydrate_context error")
	}
	want := `invalid provider history rehydrate_context "maybe" (expected: true or false)`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want containing %q", err.Error(), want)
	}
}

func TestResolveProjectProviderHistoryReductionModePrecedence(t *testing.T) {
	projectCfg := &ProjectConfig{
		Experimental: ProjectExperimentalConfig{
			ProviderHistoryReduction: ProjectProviderHistoryReductionConfig{
				Mode: ProjectProviderHistoryReductionModeDryRun,
			},
		},
	}

	tests := []struct {
		name      string
		project   *ProjectConfig
		envValue  string
		envSet    bool
		wantMode  ProjectProviderHistoryReductionMode
		wantSet   bool
		wantError string
	}{
		{
			name:     "default off is not explicit",
			wantMode: ProjectProviderHistoryReductionModeOff,
		},
		{
			name:     "project mode",
			project:  projectCfg,
			wantMode: ProjectProviderHistoryReductionModeDryRun,
			wantSet:  true,
		},
		{
			name:     "env overrides project",
			project:  projectCfg,
			envValue: "apply",
			envSet:   true,
			wantMode: ProjectProviderHistoryReductionModeApply,
			wantSet:  true,
		},
		{
			name:     "env explicit off overrides project",
			project:  projectCfg,
			envValue: "off",
			envSet:   true,
			wantMode: ProjectProviderHistoryReductionModeOff,
			wantSet:  true,
		},
		{
			name:      "invalid env",
			project:   projectCfg,
			envValue:  "x",
			envSet:    true,
			wantError: `invalid provider history reduction mode "x" (expected: off, dry_run, apply, auto)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := func(key string) (string, bool) {
				if key != ProviderHistoryReductionEnvVar || !tt.envSet {
					return "", false
				}
				return tt.envValue, true
			}
			gotMode, gotSet, err := ResolveProjectProviderHistoryReductionMode(tt.project, lookup)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveProjectProviderHistoryReductionMode() error = %v", err)
			}
			if gotMode != tt.wantMode || gotSet != tt.wantSet {
				t.Fatalf("ResolveProjectProviderHistoryReductionMode() = (%q, %v), want (%q, %v)", gotMode, gotSet, tt.wantMode, tt.wantSet)
			}
		})
	}
}

func TestResolveProjectProviderHistoryRehydrateContextPrecedence(t *testing.T) {
	projectTrue := &ProjectConfig{
		Experimental: ProjectExperimentalConfig{
			ProviderHistoryReduction: ProjectProviderHistoryReductionConfig{
				RehydrateContext: true,
			},
		},
	}
	projectFalse := &ProjectConfig{
		Experimental: ProjectExperimentalConfig{
			ProviderHistoryReduction: ProjectProviderHistoryReductionConfig{
				RehydrateContext: false,
			},
		},
	}

	tests := []struct {
		name      string
		project   *ProjectConfig
		envValue  string
		envSet    bool
		want      bool
		wantError string
	}{
		{
			name: "default false",
		},
		{
			name:    "project true",
			project: projectTrue,
			want:    true,
		},
		{
			name:    "project false",
			project: projectFalse,
		},
		{
			name:     "env true overrides project false",
			project:  projectFalse,
			envValue: "1",
			envSet:   true,
			want:     true,
		},
		{
			name:     "env false overrides project true",
			project:  projectTrue,
			envValue: "false",
			envSet:   true,
		},
		{
			name:     "empty env is unset",
			project:  projectTrue,
			envValue: " ",
			envSet:   true,
			want:     true,
		},
		{
			name:      "invalid env",
			project:   projectTrue,
			envValue:  "maybe",
			envSet:    true,
			wantError: `invalid provider history rehydrate_context "maybe" (expected: 1, true, 0, false)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := func(key string) (string, bool) {
				if key != ProviderHistoryRehydrateContextEnvVar || !tt.envSet {
					return "", false
				}
				return tt.envValue, true
			}
			got, err := ResolveProjectProviderHistoryRehydrateContext(tt.project, lookup)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveProjectProviderHistoryRehydrateContext() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ResolveProjectProviderHistoryRehydrateContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveProjectConfigWithExperimentalProviderHistoryReductionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	pc := &ProjectConfig{
		Context: "test",
		Experimental: ProjectExperimentalConfig{
			ProviderHistoryReduction: ProjectProviderHistoryReductionConfig{
				Mode:             ProjectProviderHistoryReductionModeDryRun,
				RehydrateContext: true,
			},
		},
		FilePath: filepath.Join(dir, "xelyon.yaml"),
	}

	if err := SaveProjectConfig(pc); err != nil {
		t.Fatalf("SaveProjectConfig() error = %v", err)
	}
	data, err := os.ReadFile(pc.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"experimental:", "provider_history_reduction:", "mode: dry_run", "rehydrate_context: true"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("saved xelyon.yaml missing %q:\n%s", want, string(data))
		}
	}

	loaded, err := loadProjectConfigFromYAML(pc.FilePath)
	if err != nil {
		t.Fatalf("loadProjectConfigFromYAML() error = %v", err)
	}
	if got := loaded.Experimental.ProviderHistoryReduction.Mode; got != ProjectProviderHistoryReductionModeDryRun {
		t.Fatalf("loaded provider_history_reduction.mode = %q", got)
	}
	if got := bool(loaded.Experimental.ProviderHistoryReduction.RehydrateContext); !got {
		t.Fatalf("loaded provider_history_reduction.rehydrate_context = %v, want true", got)
	}
}

func TestSaveProjectConfigOmitsEmptyExperimentalSection(t *testing.T) {
	dir := t.TempDir()
	pc := &ProjectConfig{
		Context:  "test",
		FilePath: filepath.Join(dir, "xelyon.yaml"),
	}

	if err := SaveProjectConfig(pc); err != nil {
		t.Fatalf("SaveProjectConfig() error = %v", err)
	}
	data, err := os.ReadFile(pc.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "experimental:") {
		t.Fatalf("empty experimental section should be omitted:\n%s", string(data))
	}
}
