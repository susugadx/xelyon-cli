package agent

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type scriptedStreamingProvider struct {
	name     string
	response string
}

func (p *scriptedStreamingProvider) Name() string {
	if p.name != "" {
		return p.name
	}
	return "test"
}

func (p *scriptedStreamingProvider) SupportsImages() bool { return false }

func (p *scriptedStreamingProvider) IsFunctionCallingEnabled() bool { return true }

func (p *scriptedStreamingProvider) ChatWithTools(ctx context.Context, _ string, _ []api.Message, _ string) (string, error) {
	if api.ShouldStreamAssistantText(ctx) {
		api.PrintAIHeaderWithContext(ctx)
		_, _ = api.OutputWriterFromContext(ctx).Write([]byte(p.response + "\n"))
	}
	return p.response, nil
}

func (p *scriptedStreamingProvider) ChatWithImage(_ context.Context, _ string, _ []api.Message, _ string, _ *api.ImageData, _ string) (string, error) {
	return "", nil
}

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
				provider = &scriptedStreamingProvider{name: "test", response: "investigation result"}
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

func TestPlanInvestigationRunner_HandleNoToolResponse_InvalidPlanJSONRequestsRetry(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, &mockProvider{name: "test"}, &out)
	runner := newPlanInvestigationRunner(agent, context.Background())

	response := "```json\n{\"plan\":{\"summary\":\"Research only\",\"steps\":[]}}\n```"
	p, action, err := runner.handleNoToolResponse(response)
	if err != nil {
		t.Fatalf("handleNoToolResponse() error = %v", err)
	}
	if p != nil {
		t.Fatalf("handleNoToolResponse() plan = %v, want nil", p)
	}
	if action != investigationLoopContinue {
		t.Fatalf("handleNoToolResponse() action = %v, want continue", action)
	}
	if len(agent.History) == 0 {
		t.Fatal("expected retry instruction to be appended to history")
	}
	if !strings.Contains(agent.History[len(agent.History)-1].Content, "Plan JSON を**必ず**") {
		t.Fatalf("expected retry instruction in history, got %q", agent.History[len(agent.History)-1].Content)
	}
}

func TestPlanInvestigationRunner_Run_PropagatesContextCanceled(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &blockingCancelProvider{started: make(chan struct{})}
	agent := newChatRequestTestAgent(t, provider, &out)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		_, err := newPlanInvestigationRunner(agent, ctx).Run()
		errCh <- err
	}()

	select {
	case <-provider.started:
	case <-time.After(2 * time.Second):
		t.Fatal("ChatWithTools was not called")
	}
	cancel()

	err := <-errCh
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", err)
	}
}
