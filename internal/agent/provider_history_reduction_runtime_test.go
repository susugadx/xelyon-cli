package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestSyncProviderHistoryReductionModeFromProjectConfig(t *testing.T) {
	unsetProviderHistoryReductionEnv(t)

	runtime := NewAgentRuntime()
	projectCfg := &config.ProjectConfig{
		Experimental: config.ProjectExperimentalConfig{
			ProviderHistoryReduction: config.ProjectProviderHistoryReductionConfig{
				Mode: config.ProjectProviderHistoryReductionModeDryRun,
			},
		},
	}

	if err := syncProviderHistoryReductionModeFromProjectConfig(runtime, projectCfg); err != nil {
		t.Fatalf("syncProviderHistoryReductionModeFromProjectConfig() error = %v", err)
	}
	if runtime.Options.ProviderHistoryReductionMode != ProviderHistoryReductionDryRun || !runtime.Options.ProviderHistoryReductionModeSet {
		t.Fatalf("runtime provider history reduction mode = (%v, %v), want dry-run set", runtime.Options.ProviderHistoryReductionMode, runtime.Options.ProviderHistoryReductionModeSet)
	}

	if err := syncProviderHistoryReductionModeFromProjectConfig(runtime, nil); err != nil {
		t.Fatalf("syncProviderHistoryReductionModeFromProjectConfig(nil) error = %v", err)
	}
	if runtime.Options.ProviderHistoryReductionMode != ProviderHistoryReductionDisabled || runtime.Options.ProviderHistoryReductionModeSet {
		t.Fatalf("runtime provider history reduction mode after nil project = (%v, %v), want off unset", runtime.Options.ProviderHistoryReductionMode, runtime.Options.ProviderHistoryReductionModeSet)
	}
}

func TestSyncProviderHistoryReductionModeFromProjectConfigEnvOverride(t *testing.T) {
	t.Setenv(config.ProviderHistoryReductionEnvVar, "auto")
	runtime := NewAgentRuntime()
	projectCfg := &config.ProjectConfig{
		Experimental: config.ProjectExperimentalConfig{
			ProviderHistoryReduction: config.ProjectProviderHistoryReductionConfig{
				Mode: config.ProjectProviderHistoryReductionModeDryRun,
			},
		},
	}

	if err := syncProviderHistoryReductionModeFromProjectConfig(runtime, projectCfg); err != nil {
		t.Fatalf("syncProviderHistoryReductionModeFromProjectConfig() error = %v", err)
	}
	if runtime.Options.ProviderHistoryReductionMode != ProviderHistoryReductionAuto || !runtime.Options.ProviderHistoryReductionModeSet {
		t.Fatalf("runtime provider history reduction mode = (%v, %v), want auto set", runtime.Options.ProviderHistoryReductionMode, runtime.Options.ProviderHistoryReductionModeSet)
	}
}

func TestSyncProviderHistoryReductionModeFromProjectConfigInvalidEnv(t *testing.T) {
	t.Setenv(config.ProviderHistoryReductionEnvVar, "x")
	runtime := NewAgentRuntime()

	err := syncProviderHistoryReductionModeFromProjectConfig(runtime, nil)
	if err == nil {
		t.Fatal("syncProviderHistoryReductionModeFromProjectConfig() error = nil, want invalid env error")
	}
	want := `invalid provider history reduction mode "x" (expected: off, dry_run, apply, auto)`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want containing %q", err.Error(), want)
	}
}

func TestProviderHistoryReductionPolicyForRuntimeModeResolution(t *testing.T) {
	tests := []struct {
		name string
		opts RuntimeOptions
		want ProviderHistoryReductionMode
	}{
		{
			name: "default off",
			want: ProviderHistoryReductionDisabled,
		},
		{
			name: "legacy bool apply",
			opts: RuntimeOptions{EnableProviderHistoryReduction: true},
			want: ProviderHistoryReductionApply,
		},
		{
			name: "dry run mode overrides legacy bool",
			opts: RuntimeOptions{
				EnableProviderHistoryReduction:  true,
				ProviderHistoryReductionMode:    ProviderHistoryReductionDryRun,
				ProviderHistoryReductionModeSet: true,
			},
			want: ProviderHistoryReductionDryRun,
		},
		{
			name: "explicit off overrides legacy bool",
			opts: RuntimeOptions{
				EnableProviderHistoryReduction:  true,
				ProviderHistoryReductionMode:    ProviderHistoryReductionDisabled,
				ProviderHistoryReductionModeSet: true,
			},
			want: ProviderHistoryReductionDisabled,
		},
		{
			name: "auto is dry run effective mode",
			opts: RuntimeOptions{
				ProviderHistoryReductionMode:    ProviderHistoryReductionAuto,
				ProviderHistoryReductionModeSet: true,
			},
			want: ProviderHistoryReductionDryRun,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := providerHistoryReductionPolicyForRuntime(&AgentRuntime{Options: tt.opts}).Mode
			if got != tt.want {
				t.Fatalf("providerHistoryReductionPolicyForRuntime().Mode = %v, want %v", got, tt.want)
			}
		})
	}
}
