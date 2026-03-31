package agent

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type scriptedStepProvider struct {
	name     string
	response string
}

func (p *scriptedStepProvider) Name() string {
	if p.name != "" {
		return p.name
	}
	return "test"
}

func (p *scriptedStepProvider) SupportsImages() bool {
	return false
}

func (p *scriptedStepProvider) IsFunctionCallingEnabled() bool {
	return true
}

func (p *scriptedStepProvider) ChatWithTools(ctx context.Context, _ string, _ []api.Message, _ string) (string, error) {
	if api.ShouldStreamAssistantText(ctx) {
		api.PrintAIHeaderWithContext(ctx)
		_, _ = fmt.Fprintln(api.OutputWriterFromContext(ctx), p.response)
	}
	return p.response, nil
}

func (p *scriptedStepProvider) ChatWithImage(_ context.Context, _ string, _ []api.Message, _ string, _ *api.ImageData, _ string) (string, error) {
	return "", nil
}

func TestExtractPlanJSON_Mode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "basic plan",
			input: `Based on my investigation, here is my plan:
{"plan": {"summary": "Add new feature", "steps": [{"id": 1, "description": "Step 1", "tools": ["write_file"]}]}}`,
			expected: `{"plan": {"summary": "Add new feature", "steps": [{"id": 1, "description": "Step 1", "tools": ["write_file"]}]}}`,
		},
		{
			name: "plan with space",
			input: `I'll implement this:
{ "plan": {"summary": "Fix bug", "steps": [{"id": 1, "description": "Fix", "tools": []}]}}`,
			expected: `{ "plan": {"summary": "Fix bug", "steps": [{"id": 1, "description": "Fix", "tools": []}]}}`,
		},
		{
			name:     "no plan",
			input:    "This is just a regular response without any plan.",
			expected: "",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := plan.ExtractPlanJSON(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractPlanJSON() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParsePlan_V2Format(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantSummary string
		wantSteps   int
		wantErr     bool
	}{
		{
			name:        "valid plan with wrapper",
			input:       `{"plan": {"summary": "Add feature X", "steps": [{"id": 1, "description": "Step 1", "tools": ["write_file"]}, {"id": 2, "description": "Step 2", "tools": ["str_replace"]}]}}`,
			wantSummary: "Add feature X",
			wantSteps:   2,
			wantErr:     false,
		},
		{
			name:        "empty steps",
			input:       `{"plan": {"summary": "Research only", "steps": []}}`,
			wantSummary: "Research only",
			wantSteps:   0,
			wantErr:     false,
		},
		{
			name:        "no summary",
			input:       `{"plan": {"steps": [{"id": 1, "description": "Step 1", "tools": []}]}}`,
			wantSummary: "",
			wantSteps:   1,
			wantErr:     false,
		},
		{
			name:    "invalid json",
			input:   `{"plan": invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := plan.ParsePlan(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePlan() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ParsePlan() unexpected error: %v", err)
				return
			}
			if p.Summary != tt.wantSummary {
				t.Errorf("ParsePlan() summary = %q, want %q", p.Summary, tt.wantSummary)
			}
			if len(p.Steps) != tt.wantSteps {
				t.Errorf("ParsePlan() steps count = %d, want %d", len(p.Steps), tt.wantSteps)
			}
		})
	}
}

func TestPlanStep_Tools(t *testing.T) {
	input := `{"plan": {"summary": "Test", "steps": [{"id": 1, "description": "Write file", "tools": ["write_file", "str_replace"]}]}}`
	p, err := plan.ParsePlan(input)
	if err != nil {
		t.Fatalf("ParsePlan() error: %v", err)
	}

	if len(p.Steps) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(p.Steps))
	}

	step := p.Steps[0]
	if step.ID != 1 {
		t.Errorf("Step ID = %d, want 1", step.ID)
	}
	if step.Description != "Write file" {
		t.Errorf("Step Description = %q, want %q", step.Description, "Write file")
	}
	if len(step.Tools) != 2 {
		t.Errorf("Step Tools count = %d, want 2", len(step.Tools))
	}
	if step.Tools[0] != "write_file" || step.Tools[1] != "str_replace" {
		t.Errorf("Step Tools = %v, want [write_file, str_replace]", step.Tools)
	}
}

func TestRunPlanMode_UsesRuntimeOutput(t *testing.T) {
	var out bytes.Buffer

	agent := NewAgentWithRuntime("test-model", &mockProvider{name: "test"}, false, &AgentRuntime{
		UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
	})
	t.Cleanup(agent.Cleanup)

	if err := agent.RunPlanMode(context.Background(), "investigate only"); err != nil {
		t.Fatalf("RunPlanMode() error = %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Investigation phase - researching the codebase") {
		t.Fatalf("expected runtime output to contain investigation header, got %q", output)
	}
	if !strings.Contains(output, "Investigation complete. No implementation needed.") {
		t.Fatalf("expected runtime output to contain completion message, got %q", output)
	}
}

func TestExecuteStepV2_PrintsFinalPlainTextResponseOnceWhenAssistantUpdatesSuppressed(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{name: "verbose", mode: api.AssistantUpdatesVerbose},
		{name: "phase", mode: api.AssistantUpdatesPhase},
		{name: "off", mode: api.AssistantUpdatesOff},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			cfg := config.DefaultConfig()
			cfg.Output.AssistantUpdates = tt.mode

			agent := &Agent{
				CurrentModel:    "test-model",
				CurrentProvider: &scriptedStepProvider{response: "Plan step final response"},
				Runtime: &AgentRuntime{
					Config: cfg,
					UI:     ui.NewRuntime(strings.NewReader(""), &out, &out),
				},
				History: []api.Message{},
			}

			p := &plan.Plan{
				Steps: []plan.PlanStep{{
					ID:          1,
					Description: "Finish the step",
				}},
			}

			if err := agent.executeStepV2(context.Background(), p, &p.Steps[0], 0, &retryState{}); err != nil {
				t.Fatalf("executeStepV2() error = %v", err)
			}

			output := out.String()
			if got := strings.Count(output, "Plan step final response"); got != 1 {
				t.Fatalf("expected final response exactly once in %s mode, got %d in output %q", tt.mode, got, output)
			}
			if !strings.Contains(output, "✓ Step 1 completed") {
				t.Fatalf("expected step completion message, got %q", output)
			}
		})
	}
}

func TestConfirmPlan_UsesRuntimePromptIO(t *testing.T) {
	var out bytes.Buffer

	agent := &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader("2\n"), &out, &out),
		},
	}

	approved, feedback := agent.confirmPlan()
	if approved {
		t.Fatal("confirmPlan() approved = true, want false")
	}
	if feedback != "" {
		t.Fatalf("confirmPlan() feedback = %q, want empty", feedback)
	}
	if !strings.Contains(out.String(), "Approve this plan?") {
		t.Fatalf("expected runtime output to contain confirmation prompt, got %q", out.String())
	}
}

func TestRetryState_DistinctErrors_NeverStalled(t *testing.T) {
	rs := &retryState{}
	for i := 0; i < 10; i++ {
		level := rs.recordFailure(fmt.Sprintf("error %d: something went wrong", i))
		if level != stalledNone {
			t.Fatalf("distinct error at iteration %d returned %d, want stalledNone", i, level)
		}
	}
	if rs.count != 10 {
		t.Errorf("count = %d, want 10", rs.count)
	}
}

func TestRetryState_SameError_SoftThenHard(t *testing.T) {
	rs := &retryState{}
	sameErr := "Error: compilation failed"

	// stalledRetryThreshold-1 回は stalledNone
	for i := 0; i < stalledRetryThreshold-1; i++ {
		if rs.recordFailure(sameErr) != stalledNone {
			t.Fatalf("iteration %d: expected stalledNone", i)
		}
	}

	// 閾値到達 → stalledSoft（まだ hard stop しない）
	for i := 0; i < stalledHardThreshold; i++ {
		level := rs.recordFailure(sameErr)
		if level != stalledSoft {
			t.Fatalf("soft phase iteration %d: got %d, want stalledSoft", i, level)
		}
	}

	// soft 後もさらに続くと stalledHard
	if rs.recordFailure(sameErr) != stalledHard {
		t.Fatal("expected stalledHard after soft threshold exhausted")
	}
}

func TestRetryState_DifferentErrorResetsStalledRuns(t *testing.T) {
	rs := &retryState{}

	// 同一エラー2回
	rs.recordFailure("error A")
	rs.recordFailure("error A")
	if rs.sameCount != 2 {
		t.Fatalf("sameCount = %d, want 2", rs.sameCount)
	}

	// 異なるエラーで sameCount + stalledRuns リセット
	rs.recordFailure("error B")
	if rs.sameCount != 1 {
		t.Fatalf("sameCount after different error = %d, want 1", rs.sameCount)
	}
	if rs.stalledRuns != 0 {
		t.Fatalf("stalledRuns after different error = %d, want 0", rs.stalledRuns)
	}
	if rs.count != 3 {
		t.Fatalf("count = %d, want 3", rs.count)
	}
}

func TestRetryState_Reset(t *testing.T) {
	rs := &retryState{}
	rs.recordFailure("error")
	rs.recordFailure("error")
	rs.recordFailure("error") // stalledSoft
	rs.reset()
	if rs.count != 0 || rs.sameCount != 0 || rs.stalledRuns != 0 {
		t.Errorf("reset() did not clear state: count=%d sameCount=%d stalledRuns=%d",
			rs.count, rs.sameCount, rs.stalledRuns)
	}
}

func TestErrorFingerprint_Normalize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(string) bool
	}{
		{"short preserved", "short error", func(fp string) bool { return fp == "short error" }},
		{"truncated to 200", strings.Repeat("x", 300), func(fp string) bool { return len(fp) == 200 }},
		{"trim whitespace", "  error msg  \n", func(fp string) bool { return fp == "error msg" }},
		{"compact spaces", "a  b\t\tc\n\nd", func(fp string) bool { return fp == "a b c d" }},
		{"strip ANSI", "\x1b[31mError\x1b[0m: fail", func(fp string) bool { return fp == "Error: fail" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := errorFingerprint(tt.input)
			if !tt.check(fp) {
				t.Errorf("errorFingerprint(%q) = %q", tt.input, fp)
			}
		})
	}
}
