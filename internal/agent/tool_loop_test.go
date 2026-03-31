package agent

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type loopTestTool struct{}

func (t *loopTestTool) Name() string { return "loop_tool" }

func (t *loopTestTool) Description() string { return "Loop test tool" }

func (t *loopTestTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"iteration": map[string]interface{}{
				"type": "string",
			},
		},
	}
}

func (t *loopTestTool) Run(_ tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	return fmt.Sprintf("iteration=%s", args["iteration"]), nil, nil
}

func disableColors(t *testing.T) {
	t.Helper()
	prev := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = prev
	})
}

func newLoopTestAgent(provider *sequenceMockProvider, cfg *config.Config, out *bytes.Buffer) *Agent {
	registry := tools.DefaultRegistry.Clone()
	registry.Register(&loopTestTool{})

	return &Agent{
		CurrentModel:    "test-model",
		CurrentProvider: provider,
		Runtime: &AgentRuntime{
			Config:   cfg,
			Registry: registry,
			UI:       ui.NewRuntime(strings.NewReader(""), out, out),
		},
	}
}

func buildLoopToolResponses(loopCount int, finalResponse string) []string {
	responses := make([]string, 0, loopCount+1)
	for i := 0; i < loopCount; i++ {
		responses = append(responses, fmt.Sprintf(`{"tool": "loop_tool", "args": {"iteration": "%d"}}`, i))
	}
	responses = append(responses, finalResponse)
	return responses
}

func TestToolLoop_UnlimitedDefault(t *testing.T) {
	disableColors(t)

	cfg := config.DefaultConfig()
	if cfg.General.ToolLoopLimit != 0 {
		t.Fatalf("ToolLoopLimit = %d, want 0", cfg.General.ToolLoopLimit)
	}

	var out bytes.Buffer
	provider := &sequenceMockProvider{
		name:      "test",
		responses: buildLoopToolResponses(101, "Result is available."),
	}
	agent := newLoopTestAgent(provider, cfg, &out)

	if err := agent.runNormalMode(context.Background(), "loop", nil); err != nil {
		t.Fatalf("runNormalMode() error = %v", err)
	}

	if provider.callCount != 102 {
		t.Fatalf("provider.callCount = %d, want 102", provider.callCount)
	}
	if strings.Contains(out.String(), "Tool loop limit reached") {
		t.Fatalf("unexpected hard limit warning: %q", out.String())
	}
	if !strings.Contains(out.String(), "100 iterations reached. Still working...") {
		t.Fatalf("expected 100-iteration warning, got %q", out.String())
	}
}

func TestToolLoop_HardLimitRespected(t *testing.T) {
	disableColors(t)

	cfg := config.DefaultConfig()
	cfg.General.ToolLoopLimit = 50

	var out bytes.Buffer
	provider := &sequenceMockProvider{
		name:      "test",
		responses: buildLoopToolResponses(60, "Result is available."),
	}
	agent := newLoopTestAgent(provider, cfg, &out)

	if err := agent.runNormalMode(context.Background(), "loop", nil); err != nil {
		t.Fatalf("runNormalMode() error = %v", err)
	}

	if provider.callCount != 50 {
		t.Fatalf("provider.callCount = %d, want 50", provider.callCount)
	}
	if !strings.Contains(out.String(), "Tool loop limit reached (50 iterations)") {
		t.Fatalf("expected hard limit warning, got %q", out.String())
	}
	if strings.Contains(out.String(), "100 iterations reached") {
		t.Fatalf("unexpected unlimited warning in hard-limit mode: %q", out.String())
	}
}

func TestToolLoop_NegativeLimitNormalizedToUnlimited(t *testing.T) {
	disableColors(t)

	cfg := config.DefaultConfig()
	cfg.General.ToolLoopLimit = -1

	var out bytes.Buffer
	provider := &sequenceMockProvider{
		name:      "test",
		responses: buildLoopToolResponses(101, "Result is available."),
	}
	agent := newLoopTestAgent(provider, cfg, &out)

	if err := agent.runNormalMode(context.Background(), "loop", nil); err != nil {
		t.Fatalf("runNormalMode() error = %v", err)
	}

	if provider.callCount != 102 {
		t.Fatalf("provider.callCount = %d, want 102", provider.callCount)
	}
	if !strings.Contains(out.String(), "100 iterations reached. Still working...") {
		t.Fatalf("expected unlimited warning after normalization, got %q", out.String())
	}
}

func TestToolLoop_WarningAt100(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	emitLoopWarning(agent, 100)

	if !strings.Contains(out.String(), "100 iterations reached. Still working...") {
		t.Fatalf("expected 100-iteration warning, got %q", out.String())
	}
}

func TestToolLoop_WarningAt200(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	emitLoopWarning(agent, 200)

	if !strings.Contains(out.String(), "200 iterations reached. Use Ctrl+C or /stop to abort if stuck.") {
		t.Fatalf("expected 200-iteration warning, got %q", out.String())
	}
}

func TestToolLoop_NonInteractiveKeepsLimit(t *testing.T) {
	disableColors(t)
	t.Setenv("HOME", t.TempDir())

	cfg := newProjectMapDisabledConfig()
	cfg.General.ToolLoopLimit = 10
	responses := make([]string, 20)
	for i := range responses {
		responses[i] = `{"tool": "list_dir", "args": {"path": "."}}`
	}

	provider := &sequenceMockProvider{
		name:      "test",
		responses: responses,
	}

	result := RunHeadlessWithConfig(context.Background(), "loop", "test-model", provider, cfg)

	if result.Status != "success" {
		t.Fatalf("result.Status = %q, want success", result.Status)
	}
	if provider.callCount != 10 {
		t.Fatalf("provider.callCount = %d, want 10", provider.callCount)
	}
	if len(result.ToolCalls) != 10 {
		t.Fatalf("len(result.ToolCalls) = %d, want 10", len(result.ToolCalls))
	}
}

func TestExecuteStepV2_HardLimitRespected(t *testing.T) {
	disableColors(t)

	cfg := config.DefaultConfig()
	cfg.General.ToolLoopLimit = 5

	var out bytes.Buffer
	provider := &sequenceMockProvider{
		name:      "test",
		responses: buildLoopToolResponses(10, "Result is available."),
	}
	agent := newLoopTestAgent(provider, cfg, &out)

	p := &plan.Plan{
		Summary: "Test plan",
		Steps: []plan.PlanStep{
			{ID: 1, Description: "Looping step", Status: "pending", Tools: []string{"loop_tool"}},
		},
	}

	err := agent.executeStepV2(context.Background(), p, &p.Steps[0], 0, &retryState{})
	if err == nil {
		t.Fatal("executeStepV2() error = nil, want hard limit error")
	}
	if !strings.Contains(err.Error(), "step 1 exceeded max iterations (5)") {
		t.Fatalf("executeStepV2() error = %q", err)
	}
	if provider.callCount != 5 {
		t.Fatalf("provider.callCount = %d, want 5", provider.callCount)
	}
}
