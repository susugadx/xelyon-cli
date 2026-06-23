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
	if !strings.Contains(claudePrompt, "search_code: low-level exact-search tool") || !strings.Contains(claudePrompt, "read_file: low-level exact-content override") {
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
	if strings.Contains(openAIPrompt, "search_code: low-level exact-search tool") || strings.Contains(openAIPrompt, "read_file: low-level exact-content override") {
		t.Fatal("openai edit prompt should not advertise legacy low-level investigation tools")
	}
	if !strings.Contains(openAIPrompt, "read_file: exact-content override for known files or ranges when edit/apply_patch needs precise context") {
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

func TestVerifyPrompt_RemainsReadOnlyVerificationOnly(t *testing.T) {
	for _, want := range []string{
		"Execute the verification command(s) described in the task message",
		"Do NOT modify any files",
		"Do not attempt to fix failures",
	} {
		if !strings.Contains(VerifyPrompt, want) {
			t.Fatalf("VerifyPrompt should keep verification-only restriction %q", want)
		}
	}
}
