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
		`"steps"`,
		`"description"`,
		`"tools"`,
		`"files"`,
		"reviewable sentence",
		"normally 2-6 steps",
		"understandable without the investigation transcript",
		"implementation-relevant repo-relative files",
	})
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
	if !strings.Contains(msg, "Plan JSON を**必ず**") {
		t.Fatalf("expected retry message to contain retry instruction, got %q", msg)
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
