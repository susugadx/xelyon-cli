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

func TestLoadProjectConfigWithStableProviderHistoryReduction(t *testing.T) {
	for _, mode := range []ProviderHistoryReductionMode{
		ProviderHistoryReductionModeOff,
		ProviderHistoryReductionModeDryRun,
		ProviderHistoryReductionModeApply,
	} {
		t.Run(string(mode), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "xelyon.yaml")
			yamlContent := "provider_history_reduction:\n  mode: " + string(mode) + "\n  rehydrate_context: false\n"
			if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
				t.Fatal(err)
			}

			pc, err := loadProjectConfigFromYAML(path)
			if err != nil {
				t.Fatalf("loadProjectConfigFromYAML() error = %v", err)
			}
			if got := pc.ProviderHistoryReduction.Mode; got != mode {
				t.Fatalf("provider_history_reduction.mode = %q, want %q", got, mode)
			}
			if !pc.ProviderHistoryReduction.RehydrateContextSet || bool(pc.ProviderHistoryReduction.RehydrateContext) {
				t.Fatalf("provider_history_reduction.rehydrate_context = (%v, set=%v), want false set", pc.ProviderHistoryReduction.RehydrateContext, pc.ProviderHistoryReduction.RehydrateContextSet)
			}
		})
	}
}

func TestLoadProjectConfigWithStableProviderHistoryRawOutputArtifacts(t *testing.T) {
	tmp := t.TempDir()
	rawOutputRoot := filepath.Join(tmp, "rawoutputs")
	path := filepath.Join(tmp, "xelyon.yaml")
	yamlContent := strings.Join([]string{
		"provider_history_reduction:",
		"  mode: apply",
		"  raw_output_artifacts:",
		"    mode: apply",
		"    root: " + rawOutputRoot,
		"    max_artifact_bytes: 1024",
		"    session_quota_bytes: 4096",
		"    chunk_bytes: 512",
		"    active_context_budget_tokens: 256",
		"    active_context_budget_max_tokens: 512",
		"    retention: session",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	pc, err := loadProjectConfigFromYAML(path)
	if err != nil {
		t.Fatalf("loadProjectConfigFromYAML() error = %v", err)
	}
	raw := pc.ProviderHistoryReduction.RawOutputArtifacts
	if raw.Mode != ProviderHistoryRawOutputArtifactsModeApply ||
		raw.Root != rawOutputRoot ||
		raw.MaxArtifactBytes != 1024 ||
		raw.SessionQuotaBytes != 4096 ||
		raw.ChunkBytes != 512 ||
		raw.ActiveContextBudgetTokens != 256 ||
		raw.ActiveContextBudgetMaxTokens != 512 ||
		raw.Retention != ProviderHistoryRawOutputArtifactsRetentionSession {
		t.Fatalf("RawOutputArtifacts = %#v, want explicit project values", raw)
	}

	resolved, specified, err := ResolveProviderHistoryRawOutputArtifactsConfig(DefaultConfig(), pc)
	if err != nil {
		t.Fatalf("ResolveProviderHistoryRawOutputArtifactsConfig() error = %v", err)
	}
	if !specified || resolved != raw {
		t.Fatalf("ResolveProviderHistoryRawOutputArtifactsConfig() = (%#v, %v), want project override %#v", resolved, specified, raw)
	}
}

func TestResolveProviderHistoryRawOutputArtifactsRootEnvOverridesProjectAndGlobal(t *testing.T) {
	tmp := t.TempDir()
	globalRoot := filepath.Join(tmp, "global")
	projectRoot := filepath.Join(tmp, "project")
	envRoot := filepath.Join(tmp, "env")
	globalCfg := DefaultConfig()
	globalCfg.ProviderHistoryReduction.RawOutputArtifacts.Root = globalRoot
	projectCfg := &ProjectConfig{}
	projectCfg.ProviderHistoryReduction.RawOutputArtifacts.Root = projectRoot

	resolved, specified, err := ResolveProviderHistoryRawOutputArtifactsConfigWithEnv(globalCfg, projectCfg, func(key string) (string, bool) {
		if key != ProviderHistoryRawOutputArtifactRootEnvVar {
			return "", false
		}
		return envRoot, true
	})
	if err != nil {
		t.Fatalf("ResolveProviderHistoryRawOutputArtifactsConfigWithEnv() error = %v", err)
	}
	if !specified || resolved.Root != envRoot {
		t.Fatalf("resolved root = %q specified=%v, want env root %q", resolved.Root, specified, envRoot)
	}
}

func TestResolveProviderHistoryRawOutputArtifactsRootBlankEnvKeepsProjectRoot(t *testing.T) {
	tmp := t.TempDir()
	projectRoot := filepath.Join(tmp, "project")
	projectCfg := &ProjectConfig{}
	projectCfg.ProviderHistoryReduction.RawOutputArtifacts.Root = projectRoot

	resolved, specified, err := ResolveProviderHistoryRawOutputArtifactsConfigWithEnv(DefaultConfig(), projectCfg, func(key string) (string, bool) {
		if key != ProviderHistoryRawOutputArtifactRootEnvVar {
			return "", false
		}
		return "   ", true
	})
	if err != nil {
		t.Fatalf("ResolveProviderHistoryRawOutputArtifactsConfigWithEnv() error = %v", err)
	}
	if !specified || resolved.Root != projectRoot {
		t.Fatalf("resolved root = %q specified=%v, want project root %q", resolved.Root, specified, projectRoot)
	}
}

func TestResolveProviderHistoryRawOutputArtifactsRootBlankEnvKeepsGlobalRoot(t *testing.T) {
	tmp := t.TempDir()
	globalRoot := filepath.Join(tmp, "global")
	globalCfg := DefaultConfig()
	globalCfg.ProviderHistoryReduction.RawOutputArtifacts.Root = globalRoot

	resolved, _, err := ResolveProviderHistoryRawOutputArtifactsConfigWithEnv(globalCfg, nil, func(key string) (string, bool) {
		if key != ProviderHistoryRawOutputArtifactRootEnvVar {
			return "", false
		}
		return "", true
	})
	if err != nil {
		t.Fatalf("ResolveProviderHistoryRawOutputArtifactsConfigWithEnv() error = %v", err)
	}
	if resolved.Root != globalRoot {
		t.Fatalf("resolved root = %q, want global root %q", resolved.Root, globalRoot)
	}
}

func TestResolveProviderHistoryRawOutputArtifactsRootRejectsRelativeEnv(t *testing.T) {
	_, _, err := ResolveProviderHistoryRawOutputArtifactsConfigWithEnv(DefaultConfig(), nil, func(key string) (string, bool) {
		if key != ProviderHistoryRawOutputArtifactRootEnvVar {
			return "", false
		}
		return "relative/rawoutputs", true
	})
	if err == nil {
		t.Fatal("ResolveProviderHistoryRawOutputArtifactsConfigWithEnv() error = nil, want relative root rejection")
	}
	if !strings.Contains(err.Error(), "provider_history_reduction.raw_output_artifacts.root") {
		t.Fatalf("error = %q, want raw_output_artifacts.root", err.Error())
	}
}

func TestLoadProjectConfigRejectsStableProviderHistoryReductionAuto(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "xelyon.yaml")
	if err := os.WriteFile(path, []byte("provider_history_reduction:\n  mode: auto\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := loadProjectConfigFromYAML(path)
	if err == nil {
		t.Fatal("loadProjectConfigFromYAML() error = nil, want stable auto error")
	}
	want := `invalid provider history reduction mode "auto" (expected: off, dry_run, apply)`
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
			name:     "default dry-run is not explicit",
			wantMode: ProjectProviderHistoryReductionModeDryRun,
		},
		{
			name:     "project mode",
			project:  projectCfg,
			wantMode: ProjectProviderHistoryReductionModeDryRun,
			wantSet:  true,
		},
		{
			name: "stable project overrides experimental",
			project: &ProjectConfig{
				ProviderHistoryReduction: ProjectStableProviderHistoryReductionConfig{
					Mode: ProviderHistoryReductionModeApply,
				},
				Experimental: ProjectExperimentalConfig{
					ProviderHistoryReduction: ProjectProviderHistoryReductionConfig{
						Mode: ProjectProviderHistoryReductionModeDryRun,
					},
				},
			},
			wantMode: ProjectProviderHistoryReductionModeApply,
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
				RehydrateContext:    false,
				RehydrateContextSet: true,
			},
		},
	}
	projectExperimentalModeOnly := &ProjectConfig{
		Experimental: ProjectExperimentalConfig{
			ProviderHistoryReduction: ProjectProviderHistoryReductionConfig{
				Mode: ProjectProviderHistoryReductionModeApply,
			},
		},
	}
	projectStableModeOnly := &ProjectConfig{
		ProviderHistoryReduction: ProjectStableProviderHistoryReductionConfig{
			Mode: ProviderHistoryReductionModeApply,
		},
		Experimental: ProjectExperimentalConfig{
			ProviderHistoryReduction: ProjectProviderHistoryReductionConfig{
				Mode:                ProjectProviderHistoryReductionModeDryRun,
				RehydrateContext:    false,
				RehydrateContextSet: true,
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
			name: "default true",
			want: true,
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
			name:    "experimental mode only keeps legacy false default",
			project: projectExperimentalModeOnly,
		},
		{
			name:    "stable mode only ignores experimental and keeps stable true default",
			project: projectStableModeOnly,
			want:    true,
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
				Mode:                ProjectProviderHistoryReductionModeDryRun,
				RehydrateContext:    true,
				RehydrateContextSet: true,
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

func TestSaveProjectConfigWithStableProviderHistoryReductionFalseRoundTrip(t *testing.T) {
	dir := t.TempDir()
	pc := &ProjectConfig{
		Context: "test",
		ProviderHistoryReduction: ProjectStableProviderHistoryReductionConfig{
			Mode:                ProviderHistoryReductionModeApply,
			RehydrateContext:    false,
			RehydrateContextSet: true,
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
	for _, want := range []string{"provider_history_reduction:", "mode: apply", "rehydrate_context: false"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("saved xelyon.yaml missing %q:\n%s", want, string(data))
		}
	}

	loaded, err := loadProjectConfigFromYAML(pc.FilePath)
	if err != nil {
		t.Fatalf("loadProjectConfigFromYAML() error = %v", err)
	}
	if got := loaded.ProviderHistoryReduction.Mode; got != ProviderHistoryReductionModeApply {
		t.Fatalf("loaded provider_history_reduction.mode = %q", got)
	}
	if !loaded.ProviderHistoryReduction.RehydrateContextSet || bool(loaded.ProviderHistoryReduction.RehydrateContext) {
		t.Fatalf("loaded provider_history_reduction.rehydrate_context = (%v, set=%v), want false set", loaded.ProviderHistoryReduction.RehydrateContext, loaded.ProviderHistoryReduction.RehydrateContextSet)
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
