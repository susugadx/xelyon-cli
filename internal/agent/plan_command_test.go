package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// mockPlanProvider for testing
type mockPlanProvider struct {
	api.Provider
}

func (m *mockPlanProvider) Name() string {
	return "mock"
}

func (m *mockPlanProvider) IsFunctionCallingEnabled() bool {
	return true
}

func TestHandlePlanCommand_On(t *testing.T) {
	agent := NewAgent("test-model", &mockPlanProvider{}, false)
	agent.PlanModeEnabled = false

	result := handlePlanCommand(agent, []string{"on"})

	if !result {
		t.Error("handlePlanCommand should return true")
	}
	if !agent.PlanModeEnabled {
		t.Error("PlanModeEnabled should be true after /plan on")
	}
}

func TestHandlePlanCommand_Off(t *testing.T) {
	agent := NewAgent("test-model", &mockPlanProvider{}, false)
	agent.PlanModeEnabled = true

	result := handlePlanCommand(agent, []string{"off"})

	if !result {
		t.Error("handlePlanCommand should return true")
	}
	if agent.PlanModeEnabled {
		t.Error("PlanModeEnabled should be false after /plan off")
	}
}

func TestHandlePlanCommand_TogglesSurfaceVisibility(t *testing.T) {
	runtime := newIsolatedRuntime()
	agent := NewAgentWithRuntime("gpt-5.4", &mockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	if toolNameInList(agent.registry().GetExcludedTools(), "ask_user_question") == false {
		t.Fatalf("normal mode should start with ask_user_question excluded, got %v", agent.registry().GetExcludedTools())
	}

	if !handlePlanCommand(agent, []string{"on"}) {
		t.Fatal("handlePlanCommand(on) = false, want true")
	}
	if toolNameInList(agent.registry().GetExcludedTools(), "ask_user_question") {
		t.Fatalf("plan mode should expose ask_user_question, got %v", agent.registry().GetExcludedTools())
	}

	agent.syncCurrentDerivedRuntimeState()
	if toolNameInList(agent.registry().GetExcludedTools(), "ask_user_question") {
		t.Fatalf("plan-mode derived sync should keep ask_user_question visible, got %v", agent.registry().GetExcludedTools())
	}

	if !handlePlanCommand(agent, []string{"off"}) {
		t.Fatal("handlePlanCommand(off) = false, want true")
	}
	excluded := agent.registry().GetExcludedTools()
	if !toolNameInList(excluded, "ask_user_question") {
		t.Fatalf("plan off should restore normal surface exclusion for ask_user_question, got %v", excluded)
	}
	if toolNameInList(excluded, "read_file") {
		t.Fatalf("plan off should keep read_file visible on apply_patch surfaces, got %v", excluded)
	}
}

func TestHandlePlanCommand_PreservesRuntimeSpecificExclusions(t *testing.T) {
	runtime := newIsolatedRuntime()
	agent := NewAgentWithRuntime("gpt-5.4", &mockProvider{name: "openai"}, false, runtime)
	t.Cleanup(agent.Cleanup)

	agent.registry().SetExcludedTools(append(agent.registry().GetExcludedTools(), "read_file", "mcp_github_get_issue"))

	if !handlePlanCommand(agent, []string{"on"}) {
		t.Fatal("handlePlanCommand(on) = false, want true")
	}
	if !handlePlanCommand(agent, []string{"off"}) {
		t.Fatal("handlePlanCommand(off) = false, want true")
	}

	excluded := agent.registry().GetExcludedTools()
	for _, name := range []string{"read_file", "mcp_github_get_issue", "ask_user_question"} {
		if !toolNameInList(excluded, name) {
			t.Fatalf("plan toggle should preserve runtime-specific exclusion for %s, got %v", name, excluded)
		}
	}
}

func TestHandlePlanCommand_Status(t *testing.T) {
	agent := NewAgent("test-model", &mockPlanProvider{}, false)

	// Test status when OFF
	agent.PlanModeEnabled = false
	result := handlePlanCommand(agent, []string{"status"})
	if !result {
		t.Error("handlePlanCommand should return true for status")
	}

	// Test status when ON
	agent.PlanModeEnabled = true
	result = handlePlanCommand(agent, []string{"status"})
	if !result {
		t.Error("handlePlanCommand should return true for status")
	}
}

func TestHandlePlanCommand_StatusUsesRuntimeOutput(t *testing.T) {
	var out bytes.Buffer
	agent := NewAgent("test-model", &mockPlanProvider{}, false)
	agent.Runtime = &AgentRuntime{
		UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
	}

	result := handlePlanCommand(agent, []string{"status"})
	if !result {
		t.Fatal("handlePlanCommand() = false, want true")
	}

	output := out.String()
	if !strings.Contains(output, "Plan Mode: OFF") {
		t.Fatalf("expected runtime output to contain plan status, got %q", output)
	}
}

func TestHandlePlanCommand_NoArgs(t *testing.T) {
	agent := NewAgent("test-model", &mockPlanProvider{}, false)
	agent.PlanModeEnabled = false

	// No args should show current status (not toggle)
	result := handlePlanCommand(agent, []string{})

	if !result {
		t.Error("handlePlanCommand should return true")
	}
	// Should remain false (status display, not toggle)
	if agent.PlanModeEnabled {
		t.Error("PlanModeEnabled should remain false when no args (status display)")
	}
}

func TestHandlePlanCommand_InvalidArg(t *testing.T) {
	agent := NewAgent("test-model", &mockPlanProvider{}, false)
	agent.PlanModeEnabled = false

	// Invalid arg should show status (fallthrough)
	result := handlePlanCommand(agent, []string{"invalid"})

	if !result {
		t.Error("handlePlanCommand should return true")
	}
	// Should remain unchanged
	if agent.PlanModeEnabled {
		t.Error("PlanModeEnabled should remain unchanged for invalid arg")
	}
}

func TestAgent_PlanModeEnabled_Default(t *testing.T) {
	agent := NewAgent("test-model", &mockPlanProvider{}, false)

	if agent.PlanModeEnabled {
		t.Error("PlanModeEnabled should default to false")
	}
}

func TestHandleSpecialCommand_Plan(t *testing.T) {
	agent := NewAgent("test-model", &mockPlanProvider{}, false)

	// Test /plan command is recognized
	result := handleSpecialCommand("/plan on", agent)
	if !result {
		t.Error("handleSpecialCommand should return true for /plan")
	}
	if !agent.PlanModeEnabled {
		t.Error("PlanModeEnabled should be true after /plan on")
	}

	result = handleSpecialCommand("/plan off", agent)
	if !result {
		t.Error("handleSpecialCommand should return true for /plan off")
	}
	if agent.PlanModeEnabled {
		t.Error("PlanModeEnabled should be false after /plan off")
	}
}

func TestHandleSpecialCommand_Status(t *testing.T) {
	agent := NewAgent("test-model", &mockPlanProvider{}, false)

	result := handleSpecialCommand("/status", agent)
	if !result {
		t.Error("handleSpecialCommand should return true for /status")
	}

	result = handleSpecialCommand("/stats", agent)
	if !result {
		t.Error("handleSpecialCommand should return true for /stats alias")
	}
}
