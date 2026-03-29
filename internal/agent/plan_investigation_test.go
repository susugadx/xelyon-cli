package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestRunInvestigationPhase_DebugOutputUsesRuntimeErrorOutput(t *testing.T) {
	t.Setenv("XELYON_DEBUG_PARSE", "1")

	var out bytes.Buffer
	var errOut bytes.Buffer
	agent := &Agent{
		CurrentModel:    "test-model",
		CurrentProvider: &mockProvider{name: "test"},
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &errOut),
		},
	}

	plan, err := agent.runInvestigationPhase(context.Background())
	if err != nil {
		t.Fatalf("runInvestigationPhase() error = %v", err)
	}
	if plan != nil {
		t.Fatalf("runInvestigationPhase() plan = %v, want nil", plan)
	}

	if !strings.Contains(errOut.String(), "ParseToolCalls returned 0 tools") {
		t.Fatalf("expected runtime error output to contain debug parse message, got %q", errOut.String())
	}
	if !strings.Contains(out.String(), "mock response") {
		t.Fatalf("expected runtime output to contain assistant response, got %q", out.String())
	}
}

func TestRunInvestigationPhase_PlanModeShowsFinalProse(t *testing.T) {
	tests := []struct {
		name      string
		mode      string // assistant_updates config value
		streaming bool   // provider streams text when verbose
	}{
		{name: "verbose_prints_once", mode: "verbose", streaming: true},
		{name: "phase_prints_once", mode: "phase", streaming: false},
		{name: "off_prints_once", mode: "off", streaming: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			cfg := config.DefaultConfig()
			cfg.Output.AssistantUpdates = tt.mode

			var provider api.Provider
			if tt.streaming {
				provider = &scriptedStepProvider{name: "test", response: "investigation result"}
			} else {
				provider = &mockProvider{name: "test"}
			}

			agent := &Agent{
				CurrentModel:    "test-model",
				CurrentProvider: provider,
				PlanModeEnabled: true,
				Runtime: &AgentRuntime{
					Config: cfg,
					UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
				},
			}

			plan, err := agent.runInvestigationPhase(context.Background())
			if err != nil {
				t.Fatalf("runInvestigationPhase() error = %v", err)
			}
			if plan != nil {
				t.Fatalf("runInvestigationPhase() plan = %v, want nil", plan)
			}

			output := out.String()
			needle := "investigation result"
			if !tt.streaming {
				needle = "mock response"
			}
			count := strings.Count(output, needle)
			if count != 1 {
				t.Fatalf("expected %q exactly once in %s mode, got %d in output: %q", needle, tt.mode, count, output)
			}
		})
	}
}
