package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerhistory"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/review"
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
	if got := runtime.Options.ProviderHistoryRawOutputArtifacts.Mode; got != config.ProviderHistoryRawOutputArtifactsModeDryRun {
		t.Fatalf("runtime raw_output_artifacts.mode = %q, want dry_run", got)
	}

	if err := syncProviderHistoryRuntimeConfigFromProjectConfig(runtime, nil); err != nil {
		t.Fatalf("syncProviderHistoryRuntimeConfigFromProjectConfig(nil) error = %v", err)
	}
	if runtime.Options.ProviderHistoryReductionMode != ProviderHistoryReductionDryRun || runtime.Options.ProviderHistoryReductionModeSet {
		t.Fatalf("runtime provider history reduction mode after nil project = (%v, %v), want dry-run unset", runtime.Options.ProviderHistoryReductionMode, runtime.Options.ProviderHistoryReductionModeSet)
	}
	if !runtime.Options.EnableProviderHistoryRehydrateContext {
		t.Fatal("runtime EnableProviderHistoryRehydrateContext after nil project = false, want true")
	}
	if got := runtime.Options.ProviderHistoryRawOutputArtifacts.Mode; got != config.ProviderHistoryRawOutputArtifactsModeDryRun {
		t.Fatalf("runtime raw_output_artifacts.mode after nil project = %q, want global/default dry_run", got)
	}
}

func TestSyncProviderHistoryRuntimeConfigUsesStableGlobalConfig(t *testing.T) {
	unsetProviderHistoryRuntimeConfigEnv(t)
	cfg := newProjectMapDisabledConfig()
	cfg.ProviderHistoryReduction.Mode = config.ProviderHistoryReductionModeApply
	cfg.ProviderHistoryReduction.RehydrateContext = false
	runtime := NewAgentRuntimeWithConfig(cfg)

	if err := syncProviderHistoryRuntimeConfigFromProjectConfig(runtime, nil); err != nil {
		t.Fatalf("syncProviderHistoryRuntimeConfigFromProjectConfig() error = %v", err)
	}
	if runtime.Options.ProviderHistoryReductionMode != ProviderHistoryReductionApply || runtime.Options.ProviderHistoryReductionModeSet {
		t.Fatalf("runtime provider history reduction mode = (%v, %v), want apply unset", runtime.Options.ProviderHistoryReductionMode, runtime.Options.ProviderHistoryReductionModeSet)
	}
	if runtime.Options.EnableProviderHistoryRehydrateContext {
		t.Fatal("runtime EnableProviderHistoryRehydrateContext = true, want global false")
	}
	if got := runtime.Options.ProviderHistoryRawOutputArtifacts.Mode; got != config.ProviderHistoryRawOutputArtifactsModeDryRun {
		t.Fatalf("runtime raw_output_artifacts.mode = %q, want global/default dry_run", got)
	}
}

func TestSyncProviderHistoryRuntimeConfigUsesRawOutputArtifactsProjectOverride(t *testing.T) {
	unsetProviderHistoryRuntimeConfigEnv(t)
	runtime := NewAgentRuntime()
	projectCfg := &config.ProjectConfig{
		ProviderHistoryReduction: config.ProjectStableProviderHistoryReductionConfig{
			Mode: config.ProviderHistoryReductionModeApply,
			RawOutputArtifacts: config.ProviderHistoryRawOutputArtifactsConfig{
				Mode:                         config.ProviderHistoryRawOutputArtifactsModeApply,
				MaxArtifactBytes:             8192,
				SessionQuotaBytes:            16384,
				ChunkBytes:                   4096,
				ActiveContextBudgetTokens:    2048,
				ActiveContextBudgetMaxTokens: 4096,
				Retention:                    config.ProviderHistoryRawOutputArtifactsRetentionSession,
			},
		},
	}

	if err := syncProviderHistoryRuntimeConfigFromProjectConfig(runtime, projectCfg); err != nil {
		t.Fatalf("syncProviderHistoryRuntimeConfigFromProjectConfig() error = %v", err)
	}
	raw := runtime.Options.ProviderHistoryRawOutputArtifacts
	if !runtime.Options.ProviderHistoryRawOutputArtifactsSet ||
		raw.Mode != config.ProviderHistoryRawOutputArtifactsModeApply ||
		raw.MaxArtifactBytes != 8192 ||
		raw.SessionQuotaBytes != 16384 ||
		raw.ChunkBytes != 4096 ||
		raw.ActiveContextBudgetTokens != 2048 ||
		raw.ActiveContextBudgetMaxTokens != 4096 ||
		raw.Retention != config.ProviderHistoryRawOutputArtifactsRetentionSession {
		t.Fatalf("runtime raw_output_artifacts = (%#v, set=%v), want project override", raw, runtime.Options.ProviderHistoryRawOutputArtifactsSet)
	}
	policy := providerHistoryReductionPolicyForRuntime(runtime)
	if policy.RawOutputArtifactsMode != providerhistory.RawOutputArtifactsApply ||
		!policy.RawOutputRehydrateContextEnabled {
		t.Fatalf("providerHistoryReductionPolicyForRuntime() = %#v, want raw output apply mode and rehydrate gate", policy)
	}
}

func TestSyncProviderHistoryRuntimeConfigInvalidatesRawOutputArtifactStoreOnConfigChange(t *testing.T) {
	unsetProviderHistoryRuntimeConfigEnv(t)
	runtime := NewAgentRuntime()
	store, err := rawoutputs.OpenStore(rawoutputs.Root(t.TempDir()), rawoutputs.StoreOptions{})
	if err != nil {
		t.Fatalf("rawoutputs.OpenStore() error = %v", err)
	}
	runtime.RawOutputArtifactStore = store
	projectCfg := &config.ProjectConfig{
		ProviderHistoryReduction: config.ProjectStableProviderHistoryReductionConfig{
			RawOutputArtifacts: config.ProviderHistoryRawOutputArtifactsConfig{
				Mode:                         config.ProviderHistoryRawOutputArtifactsModeApply,
				MaxArtifactBytes:             8192,
				SessionQuotaBytes:            16384,
				ChunkBytes:                   4096,
				ActiveContextBudgetTokens:    2048,
				ActiveContextBudgetMaxTokens: 4096,
				Retention:                    config.ProviderHistoryRawOutputArtifactsRetentionSession,
			},
		},
	}

	if err := syncProviderHistoryRuntimeConfigFromProjectConfig(runtime, projectCfg); err != nil {
		t.Fatalf("syncProviderHistoryRuntimeConfigFromProjectConfig() error = %v", err)
	}
	if runtime.RawOutputArtifactStore != nil {
		t.Fatal("runtime RawOutputArtifactStore was not invalidated after raw_output_artifacts config changed")
	}
}

func TestSyncProviderHistoryRuntimeConfigStableProjectOverridesExperimental(t *testing.T) {
	unsetProviderHistoryRuntimeConfigEnv(t)
	runtime := NewAgentRuntime()
	projectCfg := &config.ProjectConfig{
		ProviderHistoryReduction: config.ProjectStableProviderHistoryReductionConfig{
			Mode: config.ProviderHistoryReductionModeApply,
		},
		Experimental: config.ProjectExperimentalConfig{
			ProviderHistoryReduction: config.ProjectProviderHistoryReductionConfig{
				Mode:                config.ProjectProviderHistoryReductionModeDryRun,
				RehydrateContext:    false,
				RehydrateContextSet: true,
			},
		},
	}

	if err := syncProviderHistoryRuntimeConfigFromProjectConfig(runtime, projectCfg); err != nil {
		t.Fatalf("syncProviderHistoryRuntimeConfigFromProjectConfig() error = %v", err)
	}
	if runtime.Options.ProviderHistoryReductionMode != ProviderHistoryReductionApply || !runtime.Options.ProviderHistoryReductionModeSet {
		t.Fatalf("runtime provider history reduction mode = (%v, %v), want stable apply set", runtime.Options.ProviderHistoryReductionMode, runtime.Options.ProviderHistoryReductionModeSet)
	}
	if !runtime.Options.EnableProviderHistoryRehydrateContext {
		t.Fatal("runtime EnableProviderHistoryRehydrateContext = false, want stable default true")
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

func TestReviewPromptReductionModeUsesEffectiveProviderHistoryMode(t *testing.T) {
	tests := []struct {
		name string
		opts RuntimeOptions
		want review.ReviewPromptReductionMode
	}{
		{
			name: "off",
			want: review.ReviewPromptReductionModeOff,
		},
		{
			name: "apply",
			opts: RuntimeOptions{
				ProviderHistoryReductionMode:    ProviderHistoryReductionApply,
				ProviderHistoryReductionModeSet: true,
			},
			want: review.ReviewPromptReductionModeApply,
		},
		{
			name: "dry run does not alter review prompt",
			opts: RuntimeOptions{
				ProviderHistoryReductionMode:    ProviderHistoryReductionDryRun,
				ProviderHistoryReductionModeSet: true,
			},
			want: review.ReviewPromptReductionModeDryRun,
		},
		{
			name: "auto remains dry run for review prompt",
			opts: RuntimeOptions{
				ProviderHistoryReductionMode:    ProviderHistoryReductionAuto,
				ProviderHistoryReductionModeSet: true,
			},
			want: review.ReviewPromptReductionModeDryRun,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{Runtime: &AgentRuntime{Options: tt.opts}}
			if got := agent.reviewPromptReductionMode(); got != tt.want {
				t.Fatalf("reviewPromptReductionMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReviewRawOutputArtifactsModeUsesStableDefault(t *testing.T) {
	agent := &Agent{Runtime: &AgentRuntime{}}
	if got := agent.reviewRawOutputArtifactsMode(); got != review.ReviewRawOutputArtifactsModeDryRun {
		t.Fatalf("reviewRawOutputArtifactsMode() = %q, want default dry_run", got)
	}

	resolved := reviewRawOutputArtifactsConfigForRuntime(&AgentRuntime{})
	defaults := config.DefaultProviderHistoryRawOutputArtifactsConfig()
	if resolved.Mode != defaults.Mode ||
		resolved.ActiveContextBudgetTokens != defaults.ActiveContextBudgetTokens ||
		resolved.ActiveContextBudgetMaxTokens != defaults.ActiveContextBudgetMaxTokens {
		t.Fatalf("reviewRawOutputArtifactsConfigForRuntime() = %#v, want defaults %#v", resolved, defaults)
	}
}

func TestReviewRawOutputArtifactStoreSkipsWhenPromptReductionOff(t *testing.T) {
	agent := &Agent{Runtime: &AgentRuntime{}}
	if got := agent.reviewPromptReductionMode(); got != review.ReviewPromptReductionModeOff {
		t.Fatalf("reviewPromptReductionMode() = %q, want off", got)
	}
	if store := agent.reviewRawOutputArtifactStore(); store != nil {
		t.Fatalf("reviewRawOutputArtifactStore() = %#v, want nil when review prompt reduction is off", store)
	}
	if agent.Runtime.RawOutputArtifactStore != nil {
		t.Fatalf("runtime RawOutputArtifactStore = %#v, want unopened", agent.Runtime.RawOutputArtifactStore)
	}
}
