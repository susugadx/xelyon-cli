package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeProviderHistoryReductionProjectConfig(t *testing.T, dir, mode string) {
	t.Helper()
	data := []byte("experimental:\n  provider_history_reduction:\n    mode: " + mode + "\n")
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
