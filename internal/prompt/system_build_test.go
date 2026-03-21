package prompt

import (
	"strings"
	"testing"
)

func TestBuildSystemPrompt(t *testing.T) {
	base := "You are a helpful agent.\n## Workflow Rules\n- Do good things."

	// planModeEnabled=false の場合、basePrompt がそのまま返ること
	result := BuildSystemPrompt(base, false)
	if result != base {
		t.Errorf("BuildSystemPrompt(base, false) should return base unchanged, got %q", result)
	}

	// planModeEnabled=true の場合、Planning 関連のコンテンツが追加されること
	result = BuildSystemPrompt(base, true)
	if result == base {
		t.Error("BuildSystemPrompt(base, true) should differ from base")
	}
	if !strings.HasPrefix(result, base) {
		t.Error("result should start with the base prompt")
	}
	if !strings.Contains(result, "Plan Mode") {
		t.Error("result should contain Plan Mode content when planModeEnabled=true")
	}
	if !strings.Contains(result, "ask_user_question") {
		t.Error("result should contain ask_user_question reference when planModeEnabled=true")
	}
}

func TestBuildSystemPrompt_EmptyBase(t *testing.T) {
	result := BuildSystemPrompt("", false)
	if result != "" {
		t.Errorf("BuildSystemPrompt(\"\", false) should return empty string, got %q", result)
	}

	result = BuildSystemPrompt("", true)
	if result == "" {
		t.Error("BuildSystemPrompt(\"\", true) should return non-empty string with planning prompt")
	}
}

func TestBuildSystemPrompt_SystemPromptConstant(t *testing.T) {
	// SystemPrompt 定数を base として使えること
	result := BuildSystemPrompt(SystemPrompt, false)
	if result != SystemPrompt {
		t.Error("BuildSystemPrompt(SystemPrompt, false) should return SystemPrompt unchanged")
	}

	result = BuildSystemPrompt(SystemPrompt, true)
	if !strings.Contains(result, "## Workflow Rules") {
		t.Error("result should preserve Workflow Rules from SystemPrompt")
	}
	if !strings.Contains(result, "## Core Identity") {
		t.Error("result should preserve Core Identity from SystemPrompt")
	}
	if !strings.Contains(result, "Plan Mode") {
		t.Error("result should append Plan Mode content")
	}
}
