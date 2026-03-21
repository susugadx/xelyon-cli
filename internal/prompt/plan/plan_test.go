package plan

import (
	"strings"
	"testing"
)

// --- BuildPlanningPrompt ---

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

// --- BuildStepPrompt ---

func TestBuildStepPrompt_ContainsStepID(t *testing.T) {
	prompt := BuildStepPrompt(3, "Refactor the function", nil)
	if !strings.Contains(prompt, "step 3") {
		t.Error("expected prompt to contain step number")
	}
}

func TestBuildStepPrompt_ContainsDescription(t *testing.T) {
	prompt := BuildStepPrompt(1, "Add unit tests for parser", nil)
	if !strings.Contains(prompt, "Add unit tests for parser") {
		t.Error("expected prompt to contain step description")
	}
}

func TestBuildStepPrompt_WithTools(t *testing.T) {
	prompt := BuildStepPrompt(2, "Edit file", []string{"str_replace", "read_file"})
	if !strings.Contains(prompt, "str_replace") {
		t.Error("expected prompt to list str_replace tool")
	}
	if !strings.Contains(prompt, "read_file") {
		t.Error("expected prompt to list read_file tool")
	}
	if !strings.Contains(prompt, "MUST use") {
		t.Error("expected prompt to require tool usage")
	}
}

func TestBuildStepPrompt_WithoutTools(t *testing.T) {
	prompt := BuildStepPrompt(1, "Task", nil)
	if strings.Contains(prompt, "MUST use the following tools") {
		t.Error("expected no tools hint when tools list is empty")
	}
}

func TestBuildStepPrompt_ContainsRules(t *testing.T) {
	prompt := BuildStepPrompt(1, "Task", nil)
	if !strings.Contains(prompt, "Execute ONLY this step") {
		t.Error("expected step isolation rule")
	}
	if !strings.Contains(prompt, "WHAT:") {
		t.Error("expected report format")
	}
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
}
