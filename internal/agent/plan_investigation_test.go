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

const planJSONRetryPromptFragment = "Plan JSON を**必ず**"

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
	agent, runner := newPlanInvestigationNoToolTest(t)

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
	assertPlanJSONRetryPromptAppended(t, agent, `"files"`)
}

func TestPlanInvestigationRunner_HandleNoToolResponse_MalformedPlanWrapperRequestsRetry(t *testing.T) {
	agent, runner := newPlanInvestigationNoToolTest(t)

	p, action, err := runner.handleNoToolResponse(`{"plan": invalid}`)
	if err != nil {
		t.Fatalf("handleNoToolResponse() error = %v", err)
	}
	if p != nil {
		t.Fatalf("handleNoToolResponse() plan = %v, want nil", p)
	}
	if action != investigationLoopContinue {
		t.Fatalf("handleNoToolResponse() action = %v, want continue", action)
	}
	assertPlanJSONRetryPromptAppended(t, agent)
}

func TestPlanInvestigationRunner_HandleNoToolResponse_MalformedLegacyStepsRequestsRetry(t *testing.T) {
	agent, runner := newPlanInvestigationNoToolTest(t)

	p, action, err := runner.handleNoToolResponse(`{"steps": invalid}`)
	if err != nil {
		t.Fatalf("handleNoToolResponse() error = %v", err)
	}
	if p != nil {
		t.Fatalf("handleNoToolResponse() plan = %v, want nil", p)
	}
	if action != investigationLoopContinue {
		t.Fatalf("handleNoToolResponse() action = %v, want continue", action)
	}
	assertPlanJSONRetryPromptAppended(t, agent)
}

func TestPlanInvestigationRunner_HandleNoToolResponse_SchemaInvalidPlanJSONRequestsRetry(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{
			name:     "wrapper steps object",
			response: `{"plan":{"summary":"Fix","steps":{"id":1,"description":"Do it"}}}`,
		},
		{
			name:     "legacy steps object",
			response: `{"summary":"Fix","steps":{"id":1,"description":"Do it"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, runner := newPlanInvestigationNoToolTest(t)

			p, action, err := runner.handleNoToolResponse(tt.response)
			if err != nil {
				t.Fatalf("handleNoToolResponse() error = %v", err)
			}
			if p != nil {
				t.Fatalf("handleNoToolResponse() plan = %v, want nil", p)
			}
			if action != investigationLoopContinue {
				t.Fatalf("handleNoToolResponse() action = %v, want continue", action)
			}
			assertPlanJSONRetryPromptAppended(t, agent)
		})
	}
}

func TestPlanInvestigationRunner_HandleNoToolResponse_FencedLegacyRetrySchemaReturnsPlan(t *testing.T) {
	agent, runner := newPlanInvestigationNoToolTest(t)

	response := "```json\n" + `{
  "title": "Fix parser",
  "goal": "Preserve legacy plan compatibility",
  "assumptions": ["Parser still accepts legacy steps"],
  "steps": [
    {
      "id": 1,
      "description": "Update legacy evidence",
      "expected_output": "Legacy fenced plan is extracted"
    }
  ]
}` + "\n```"
	p, action, err := runner.handleNoToolResponse(response)
	if err != nil {
		t.Fatalf("handleNoToolResponse() error = %v", err)
	}
	if p == nil {
		t.Fatal("handleNoToolResponse() plan = nil, want legacy plan")
	}
	if action != investigationLoopDone {
		t.Fatalf("handleNoToolResponse() action = %v, want done", action)
	}
	if p.Title != "Fix parser" || len(p.Steps) != 1 || p.Steps[0].Description != "Update legacy evidence" {
		t.Fatalf("handleNoToolResponse() plan = %#v, want parsed legacy plan", p)
	}
	assertPlanJSONRetryPromptNotAppended(t, agent)
}

func TestPlanInvestigationRunner_HandleNoToolResponse_ToolCallJSONWithPlanShapedStepsDoesNotRetry(t *testing.T) {
	agent, runner := newPlanInvestigationNoToolTest(t)

	response := "```json\n" +
		`{"tool":"read_file","steps":[{"id":1,"description":"Read parser","tools":["read_file"]}],"args":{"paths":["internal/agent/plan/parser.go"]}}` +
		"\n```"
	p, action, err := runner.handleNoToolResponse(response)
	if err != nil {
		t.Fatalf("handleNoToolResponse() error = %v", err)
	}
	if p != nil {
		t.Fatalf("handleNoToolResponse() plan = %v, want nil", p)
	}
	if action != investigationLoopDone {
		t.Fatalf("handleNoToolResponse() action = %v, want done", action)
	}
	assertPlanJSONRetryPromptNotAppended(t, agent)
}

func newPlanInvestigationNoToolTest(t *testing.T) (*Agent, *planInvestigationRunner) {
	t.Helper()
	disableColors(t)

	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, &mockProvider{name: "test"}, &out)
	return agent, newPlanInvestigationRunner(agent, context.Background())
}

func assertPlanJSONRetryPromptAppended(t *testing.T, agent *Agent, requiredFragments ...string) {
	t.Helper()

	if len(agent.History) == 0 {
		t.Fatal("expected retry instruction to be appended to history")
	}
	content := agent.History[len(agent.History)-1].Content
	if !strings.Contains(content, planJSONRetryPromptFragment) {
		t.Fatalf("expected retry instruction in history, got %q", content)
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected retry instruction to include %q, got %q", fragment, content)
		}
	}
}

func assertPlanJSONRetryPromptNotAppended(t *testing.T, agent *Agent) {
	t.Helper()

	for _, msg := range agent.History {
		if strings.Contains(msg.Content, planJSONRetryPromptFragment) {
			t.Fatalf("plan mode should not append retry instruction for tool-call JSON, got %#v", agent.History)
		}
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
