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

type scriptedPlanProvider struct {
	name      string
	responses []string
	index     int
}

type scriptedStepProvider struct {
	name     string
	response string
}

func (p *scriptedPlanProvider) Name() string {
	if p.name != "" {
		return p.name
	}
	return "test"
}

func (p *scriptedPlanProvider) SupportsImages() bool {
	return false
}

func (p *scriptedPlanProvider) IsFunctionCallingEnabled() bool {
	return true
}

func (p *scriptedPlanProvider) ChatWithTools(_ context.Context, _ string, _ []api.Message, _ string) (string, error) {
	if p.index >= len(p.responses) {
		return "No more scripted responses.", nil
	}
	resp := p.responses[p.index]
	p.index++
	return resp, nil
}

func (p *scriptedPlanProvider) ChatWithImage(_ context.Context, _ string, _ []api.Message, _ string, _ *api.ImageData, _ string) (string, error) {
	return "", nil
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

			if err := agent.executeStepV2(context.Background(), p, &p.Steps[0], 0, 0); err != nil {
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

func TestRunPlanMode_ClearContextOnApproval_ConfigIgnored(t *testing.T) {
	var out bytes.Buffer

	cfg := config.DefaultConfig()
	cfg.PlanMode.ClearContextOnApproval = true
	provider := &scriptedPlanProvider{
		name: "test",
		responses: []string{
			`{"plan":{"summary":"Implement change","steps":[{"id":1,"description":"Apply the requested fix","tools":[]}]}}`,
			"Applied the requested fix.",
		},
	}
	agent := NewAgentWithRuntime("test-model", provider, false, &AgentRuntime{
		Config: cfg,
		UI:     ui.NewRuntime(strings.NewReader("1\n"), &out, &out),
	})
	agent.storage = nil
	agent.session = nil
	t.Cleanup(agent.Cleanup)

	if err := agent.RunPlanMode(context.Background(), "auth.goを修正して"); err != nil {
		t.Fatalf("RunPlanMode() error = %v", err)
	}

	if len(agent.History) != 4 {
		t.Fatalf("History length = %d, want 4", len(agent.History))
	}
	if strings.Contains(agent.History[0].Content, "[Approved Implementation Plan]") {
		t.Fatalf("History[0].Content = %q, want original investigation prompt", agent.History[0].Content)
	}
	if strings.Contains(out.String(), "Context cleared for implementation") {
		t.Fatalf("did not expect clear-context message, got %q", out.String())
	}
}
