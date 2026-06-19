package plan

import (
	"strings"
	"testing"
)

// --- BuildPlanningPrompt ---

func TestPlanJSONSchemaInstructions_ContainsFilesContract(t *testing.T) {
	assertContainsAll(t, "plan schema", PlanJSONSchemaInstructions(), []string{
		`"plan"`,
		`"summary"`,
		`"findings"`,
		`"evidence"`,
		`"constraints"`,
		`"steps"`,
		`"description"`,
		`"purpose"`,
		`"tools"`,
		`"files"`,
		`"verification"`,
		"reviewable sentence",
		"stable facts discovered from the codebase",
		"files, functions, tests, commands",
		"constraints, compatibility requirements",
		"normally 2-6 steps",
		"understandable without the investigation transcript",
		"review-facing reason",
		"not test commands",
		"implementation-relevant repo-relative files",
		"focused commands or checks",
	})
}

func TestPlanJSONSchemaInstructions_UsesCurrentHandoffOwnerExample(t *testing.T) {
	schema := PlanJSONSchemaInstructions()
	assertContainsAll(t, "plan schema", schema, []string{
		"internal/agent/plan/handoff.go: ImplementationHandoff.NormalModeInput builds the implementation handoff",
		"internal/agent/plan/handoff.go",
		"internal/agent/plan/handoff_test.go",
	})
	for _, stale := range []string{
		"internal/agent/plan_handoff.go",
		"internal/agent/plan_handoff_test.go",
		"normalModeInput",
	} {
		if strings.Contains(schema, stale) {
			t.Fatalf("plan schema should not contain stale handoff owner reference %q:\n%s", stale, schema)
		}
	}
}

func TestBuildPlanningPrompt_ContainsPlanMode(t *testing.T) {
	prompt := BuildPlanningPrompt()
	if !strings.Contains(prompt, "Plan Mode") {
		t.Error("expected prompt to mention Plan Mode")
	}
}

func TestBuildPlanningPrompt_ContainsAskUserQuestion(t *testing.T) {
	prompt := BuildPlanningPrompt()
	if !strings.Contains(prompt, "ask_user_question") {
		t.Error("expected prompt to mention ask_user_question tool")
	}
}

func TestBuildPlanningPrompt_ContainsOutputFormat(t *testing.T) {
	prompt := BuildPlanningPrompt()
	if !strings.Contains(prompt, "JSON") {
		t.Error("expected prompt to mention JSON output format")
	}
	if !strings.Contains(prompt, "ExtractPlanJSON") {
		t.Error("expected prompt to reference ExtractPlanJSON")
	}
}

func TestBuildPlanningPrompt_ContainsPlanSchemaWithFiles(t *testing.T) {
	prompt := BuildPlanningPrompt()
	assertContainsPlanSchema(t, "planning prompt", prompt)
}

// --- BuildPlanRequestMessage ---

func TestBuildPlanRequestMessage_ContainsToolName(t *testing.T) {
	msg := BuildPlanRequestMessage("write_file")
	if !strings.Contains(msg, "write_file") {
		t.Error("expected message to contain the tool name")
	}
}

func TestBuildPlanRequestMessage_MentionsModificationTool(t *testing.T) {
	msg := BuildPlanRequestMessage("str_replace")
	if !strings.Contains(msg, "modification tool") {
		t.Error("expected message to mention modification tool")
	}
}

func TestBuildPlanRequestMessage_RequestsPlan(t *testing.T) {
	msg := BuildPlanRequestMessage("delete_file")
	if !strings.Contains(msg, "implementation plan") {
		t.Error("expected message to request an implementation plan")
	}
	if !strings.Contains(msg, "ExtractPlanJSON") {
		t.Error("expected message to reference ExtractPlanJSON")
	}
	if !strings.Contains(msg, "Do not call tools in this response.") {
		t.Error("expected message to forbid tool calls in the retry response")
	}
}

func TestBuildPlanRequestMessage_ContainsPlanSchemaWithFiles(t *testing.T) {
	msg := BuildPlanRequestMessage("str_replace")
	assertContainsPlanSchema(t, "plan request message", msg)
}

func TestBuildPlanJSONRetryMessage_ContainsPlanSchemaWithFiles(t *testing.T) {
	msg := BuildPlanJSONRetryMessage()
	if !strings.Contains(msg, "Plan JSON retry") {
		t.Fatalf("expected retry message to contain retry instruction, got %q", msg)
	}
	if strings.Contains(msg, "[SYSTEM") {
		t.Fatalf("retry message should not contain fake system marker, got %q", msg)
	}
	assertContainsPlanSchema(t, "retry message", msg)
}

func assertContainsPlanSchema(t *testing.T, label string, content string) {
	t.Helper()
	if !strings.Contains(content, PlanJSONSchemaInstructions()) {
		t.Fatalf("%s should contain PlanJSONSchemaInstructions()", label)
	}
}

func assertContainsAll(t *testing.T, label string, content string, fragments []string) {
	t.Helper()
	for _, want := range fragments {
		if !strings.Contains(content, want) {
			t.Fatalf("%s should contain schema fragment %q", label, want)
		}
	}
}
