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
	if !strings.Contains(claudePrompt, "Copy exact old_str and existing context from actual gather_context, read_file, or search_code output") {
		t.Fatal("claude edit prompt should require evidence-backed str_replace old_str")
	}
	if !strings.Contains(claudePrompt, "Write new_str as the intended replacement based on that verified context") {
		t.Fatal("claude edit prompt should allow generated replacement text based on verified context")
	}
	if !strings.Contains(claudePrompt, "edits=[{old_str,new_str},...]") {
		t.Fatal("claude edit prompt should recommend same-file str_replace batch edits")
	}
	if !strings.Contains(claudePrompt, "advanced fallback only") {
		t.Fatal("claude edit prompt should demote line-range str_replace to fallback guidance")
	}

	openAIPrompt := PromptForTaskType(TaskTypeEdit, "openai", "gpt-5.4")
	if !strings.Contains(openAIPrompt, "apply_patch") {
		t.Fatal("openai edit prompt should mention apply_patch")
	}
	if strings.Contains(openAIPrompt, "str_replace") {
		t.Fatal("openai edit prompt should not mention str_replace")
	}
	if strings.Contains(openAIPrompt, "search_code: code discovery tool") || strings.Contains(openAIPrompt, "read_file: low-level exact-content reader") {
		t.Fatal("openai edit prompt should not advertise legacy low-level investigation tools")
	}
	if !strings.Contains(openAIPrompt, "read_file: exact-content reader for edit/apply_patch exact-control override") {
		t.Fatal("openai edit prompt should keep read_file exact-control guidance when it stays visible")
	}

	kimiPrompt := PromptForTaskType(TaskTypeEdit, "kimi", "kimi-k2.6")
	if strings.Contains(kimiPrompt, "apply_patch") {
		t.Fatal("kimi edit prompt should not mention apply_patch")
	}
	if !strings.Contains(kimiPrompt, "str_replace") {
		t.Fatal("kimi edit prompt should mention str_replace")
	}
}
