package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func writeProviderHistoryReductionProjectConfig(t *testing.T, dir, mode string) {
	t.Helper()
	data := []byte("experimental:\n  provider_history_reduction:\n    mode: " + mode + "\n")
	if err := os.WriteFile(filepath.Join(dir, "xelyon.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeProviderHistoryReductionProjectConfigWithRehydrateContext(t *testing.T, dir, mode string, rehydrateContext bool) {
	t.Helper()
	data := []byte("experimental:\n  provider_history_reduction:\n    mode: " + mode + "\n    rehydrate_context: " + strings.ToLower(boolString(rehydrateContext)) + "\n")
	if err := os.WriteFile(filepath.Join(dir, "xelyon.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeProviderHistoryReductionProjectConfigWithFinalChecks(t *testing.T, dir, mode, command string) {
	t.Helper()
	data := []byte("experimental:\n  provider_history_reduction:\n    mode: " + mode + "\nfinal_checks:\n  commands:\n    - " + command + "\n  timeout: 30\n")
	if err := os.WriteFile(filepath.Join(dir, "xelyon.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func newProviderHistoryRuntimeProjectConfig(mode config.ProjectProviderHistoryReductionMode, rehydrateContext bool) *config.ProjectConfig {
	return &config.ProjectConfig{
		Experimental: config.ProjectExperimentalConfig{
			ProviderHistoryReduction: config.ProjectProviderHistoryReductionConfig{
				Mode:                mode,
				RehydrateContext:    config.ProjectProviderHistoryRehydrateContext(rehydrateContext),
				RehydrateContextSet: true,
			},
		},
	}
}

func assertProviderHistoryReductionRuntimeMode(t *testing.T, runtime *AgentRuntime, wantMode ProviderHistoryReductionMode, wantSet bool) {
	t.Helper()
	if runtime.Options.ProviderHistoryReductionMode != wantMode || runtime.Options.ProviderHistoryReductionModeSet != wantSet {
		t.Fatalf("runtime provider history reduction mode = (%v, %v), want (%v, %v)", runtime.Options.ProviderHistoryReductionMode, runtime.Options.ProviderHistoryReductionModeSet, wantMode, wantSet)
	}
}

func assertRuntimeFinalChecks(t *testing.T, agent *Agent, wantCommands []string, wantTimeout int) {
	t.Helper()
	got := agent.cfg().FinalChecks
	if strings.Join(got.Commands, "\n") != strings.Join(wantCommands, "\n") || got.Timeout != wantTimeout {
		t.Fatalf("runtime FinalChecks = %#v, want commands %#v timeout %d", got, wantCommands, wantTimeout)
	}
}

func assertInvalidProviderHistoryReductionModeError(t *testing.T, got string) {
	t.Helper()
	want := `invalid provider history reduction mode "x" (expected: off, dry_run, apply, auto)`
	if !strings.Contains(got, want) {
		t.Fatalf("error = %q, want containing %q", got, want)
	}
}

func assertInvalidProviderHistoryRehydrateContextError(t *testing.T, got string) {
	t.Helper()
	want := `invalid provider history rehydrate_context "maybe" (expected: 1, true, 0, false)`
	if !strings.Contains(got, want) {
		t.Fatalf("error = %q, want containing %q", got, want)
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
