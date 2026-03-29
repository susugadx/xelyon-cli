package subagent

import (
	"strings"
	"testing"
)

func TestExplorePrompt_ProjectMapGuidance(t *testing.T) {
	if !strings.Contains(ExplorePrompt, "symbol definitions with line ranges") {
		t.Error("ExplorePrompt should describe Project Map as structure index")
	}
	if !strings.Contains(ExplorePrompt, "search_code: code discovery tool") {
		t.Error("ExplorePrompt should describe search_code as code discovery tool")
	}
	if strings.Contains(ExplorePrompt, "imports") || strings.Contains(ExplorePrompt, "← refs") {
		t.Error("ExplorePrompt should not mention imports or refs")
	}
}

func TestExplorePrompt_SearchCodeNotOverlyRestricted(t *testing.T) {
	if strings.Contains(ExplorePrompt, "ONLY for patterns NOT in Project Map") {
		t.Error("search_code guidance should not be overly restrictive")
	}
	if !strings.Contains(ExplorePrompt, "Uses language-aware routing across symbol-aware resolution, literal search, and regex search") {
		t.Error("search_code should describe language-aware routing")
	}
	if !strings.Contains(ExplorePrompt, "Prefer mode=auto") {
		t.Error("search_code should prefer mode=auto")
	}
}

func TestEditPromptForEditTool_Default(t *testing.T) {
	editPrompt := EditPromptForEditTool("")
	if !strings.Contains(editPrompt, "apply_patch") {
		t.Error("default mode should mention apply_patch")
	}
	if strings.Contains(editPrompt, "str_replace") {
		t.Error("default mode should not mention str_replace")
	}
	if !strings.Contains(editPrompt, "search_code: code discovery tool") {
		t.Error("EditPrompt should describe search_code as code discovery tool")
	}
	if strings.Contains(editPrompt, "imports") || strings.Contains(editPrompt, "← refs") {
		t.Error("EditPrompt should not mention imports or refs")
	}
}

func TestEditPromptForEditTool_Legacy(t *testing.T) {
	prompt := EditPromptForEditTool("str_replace")
	if strings.Contains(prompt, "apply_patch") {
		t.Error("legacy mode should not mention apply_patch")
	}
	if !strings.Contains(prompt, "str_replace") {
		t.Error("legacy mode should mention str_replace")
	}
}

func TestEditPrompt_SearchCodeNotOverlyRestricted(t *testing.T) {
	editPrompt := EditPromptForEditTool("")
	if strings.Contains(editPrompt, "ONLY for patterns NOT in Project Map") {
		t.Error("search_code guidance should not be overly restrictive")
	}
}

func TestPromptForTaskType_EditUsesProviderResolvedMode(t *testing.T) {
	claudePrompt := PromptForTaskType(TaskTypeEdit, "claude", "claude-sonnet-4-6")
	if strings.Contains(claudePrompt, "apply_patch") {
		t.Fatal("claude edit prompt should not mention apply_patch")
	}
	if !strings.Contains(claudePrompt, "str_replace") {
		t.Fatal("claude edit prompt should mention str_replace")
	}

	openAIPrompt := PromptForTaskType(TaskTypeEdit, "openai", "gpt-5.4")
	if !strings.Contains(openAIPrompt, "apply_patch") {
		t.Fatal("openai edit prompt should mention apply_patch")
	}
	if strings.Contains(openAIPrompt, "str_replace") {
		t.Fatal("openai edit prompt should not mention str_replace")
	}
}
