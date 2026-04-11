package subagent

import (
	"strings"
	"testing"
)

func TestPromptForTaskType_EditUsesProviderResolvedMode(t *testing.T) {
	claudePrompt := PromptForTaskType(TaskTypeEdit, "claude", "claude-sonnet-4-6")
	if strings.Contains(claudePrompt, "apply_patch") {
		t.Fatal("claude edit prompt should not mention apply_patch")
	}
	if !strings.Contains(claudePrompt, "str_replace") {
		t.Fatal("claude edit prompt should mention str_replace")
	}
	if !strings.Contains(claudePrompt, "search_code: code discovery tool") || !strings.Contains(claudePrompt, "read_file: low-level exact-content reader") {
		t.Fatal("claude edit prompt should expose low-level investigation overrides")
	}

	openAIPrompt := PromptForTaskType(TaskTypeEdit, "openai", "gpt-5.4")
	if !strings.Contains(openAIPrompt, "apply_patch") {
		t.Fatal("openai edit prompt should mention apply_patch")
	}
	if strings.Contains(openAIPrompt, "str_replace") {
		t.Fatal("openai edit prompt should not mention str_replace")
	}
	if strings.Contains(openAIPrompt, "search_code: code discovery tool") || strings.Contains(openAIPrompt, "read_file: low-level exact-content reader") {
		t.Fatal("openai edit prompt should not advertise hidden low-level investigation tools")
	}
}
