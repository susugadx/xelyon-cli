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
