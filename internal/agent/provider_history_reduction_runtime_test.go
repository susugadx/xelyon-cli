package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestSyncProviderHistoryRuntimeConfigFromProjectConfig(t *testing.T) {
	unsetProviderHistoryRuntimeConfigEnv(t)

	runtime := NewAgentRuntime()
	projectCfg := newProviderHistoryRuntimeProjectConfig(config.ProjectProviderHistoryReductionModeDryRun, true)

	if err := syncProviderHistoryRuntimeConfigFromProjectConfig(runtime, projectCfg); err != nil {
		t.Fatalf("syncProviderHistoryRuntimeConfigFromProjectConfig() error = %v", err)
	}
	if runtime.Options.ProviderHistoryReductionMode != ProviderHistoryReductionDryRun || !runtime.Options.ProviderHistoryReductionModeSet {
		t.Fatalf("runtime provider history reduction mode = (%v, %v), want dry-run set", runtime.Options.ProviderHistoryReductionMode, runtime.Options.ProviderHistoryReductionModeSet)
	}
	if !runtime.Options.EnableProviderHistoryRehydrateContext {
		t.Fatal("runtime EnableProviderHistoryRehydrateContext = false, want true")
	}

	if err := syncProviderHistoryRuntimeConfigFromProjectConfig(runtime, nil); err != nil {
		t.Fatalf("syncProviderHistoryRuntimeConfigFromProjectConfig(nil) error = %v", err)
	}
	if runtime.Options.ProviderHistoryReductionMode != ProviderHistoryReductionDisabled || runtime.Options.ProviderHistoryReductionModeSet {
		t.Fatalf("runtime provider history reduction mode after nil project = (%v, %v), want off unset", runtime.Options.ProviderHistoryReductionMode, runtime.Options.ProviderHistoryReductionModeSet)
	}
	if runtime.Options.EnableProviderHistoryRehydrateContext {
		t.Fatal("runtime EnableProviderHistoryRehydrateContext after nil project = true, want false")
	}
}

func TestSyncProviderHistoryRuntimeConfigModeEnvOverride(t *testing.T) {
	t.Setenv(config.ProviderHistoryReductionEnvVar, "auto")
	runtime := NewAgentRuntime()
	projectCfg := newProviderHistoryRuntimeProjectConfig(config.ProjectProviderHistoryReductionModeDryRun, false)

	if err := syncProviderHistoryRuntimeConfigFromProjectConfig(runtime, projectCfg); err != nil {
		t.Fatalf("syncProviderHistoryRuntimeConfigFromProjectConfig() error = %v", err)
	}
	if runtime.Options.ProviderHistoryReductionMode != ProviderHistoryReductionAuto || !runtime.Options.ProviderHistoryReductionModeSet {
		t.Fatalf("runtime provider history reduction mode = (%v, %v), want auto set", runtime.Options.ProviderHistoryReductionMode, runtime.Options.ProviderHistoryReductionModeSet)
	}
}

func TestSyncProviderHistoryRehydrateContextFromProjectConfigEnvOverride(t *testing.T) {
	unsetProviderHistoryRuntimeConfigEnv(t)

	t.Run("env true overrides project false", func(t *testing.T) {
		t.Setenv(config.ProviderHistoryRehydrateContextEnvVar, "true")
		runtime := NewAgentRuntime()
		projectCfg := newProviderHistoryRuntimeProjectConfig(config.ProjectProviderHistoryReductionModeApply, false)

		if err := syncProviderHistoryRuntimeConfigFromProjectConfig(runtime, projectCfg); err != nil {
			t.Fatalf("syncProviderHistoryRuntimeConfigFromProjectConfig() error = %v", err)
		}
		if !runtime.Options.EnableProviderHistoryRehydrateContext {
			t.Fatal("runtime EnableProviderHistoryRehydrateContext = false, want true")
		}
	})

	t.Run("env false overrides project true", func(t *testing.T) {
		t.Setenv(config.ProviderHistoryRehydrateContextEnvVar, "0")
		runtime := NewAgentRuntime()
		projectCfg := newProviderHistoryRuntimeProjectConfig(config.ProjectProviderHistoryReductionModeApply, true)

		if err := syncProviderHistoryRuntimeConfigFromProjectConfig(runtime, projectCfg); err != nil {
			t.Fatalf("syncProviderHistoryRuntimeConfigFromProjectConfig() error = %v", err)
		}
		if runtime.Options.EnableProviderHistoryRehydrateContext {
			t.Fatal("runtime EnableProviderHistoryRehydrateContext = true, want false")
		}
	})
}

func TestSyncProviderHistoryRuntimeConfigModeInvalidEnv(t *testing.T) {
	t.Setenv(config.ProviderHistoryReductionEnvVar, "x")
	runtime := NewAgentRuntime()

	err := syncProviderHistoryRuntimeConfigFromProjectConfig(runtime, nil)
	if err == nil {
		t.Fatal("syncProviderHistoryRuntimeConfigFromProjectConfig() error = nil, want invalid env error")
	}
	want := `invalid provider history reduction mode "x" (expected: off, dry_run, apply, auto)`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want containing %q", err.Error(), want)
	}
}

func TestSyncProviderHistoryRehydrateContextFromProjectConfigInvalidEnvDoesNotMutateRuntime(t *testing.T) {
	unsetProviderHistoryRuntimeConfigEnv(t)
	t.Setenv(config.ProviderHistoryRehydrateContextEnvVar, "maybe")
	runtime := NewAgentRuntime()
	runtime.Options.ProviderHistoryReductionMode = ProviderHistoryReductionApply
	runtime.Options.ProviderHistoryReductionModeSet = true
	runtime.Options.EnableProviderHistoryRehydrateContext = true
	projectCfg := newProviderHistoryRuntimeProjectConfig(config.ProjectProviderHistoryReductionModeDryRun, false)

	err := syncProviderHistoryRuntimeConfigFromProjectConfig(runtime, projectCfg)
	if err == nil {
		t.Fatal("syncProviderHistoryRuntimeConfigFromProjectConfig() error = nil, want invalid env error")
	}
	want := `invalid provider history rehydrate_context "maybe" (expected: 1, true, 0, false)`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want containing %q", err.Error(), want)
	}
	assertProviderHistoryReductionRuntimeMode(t, runtime, ProviderHistoryReductionApply, true)
	if !runtime.Options.EnableProviderHistoryRehydrateContext {
		t.Fatal("runtime EnableProviderHistoryRehydrateContext changed to false after invalid env")
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
