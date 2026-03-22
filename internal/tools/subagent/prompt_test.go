package subagent

import (
	"strings"
	"testing"
)

func TestExplorePrompt_ProjectMapGuidance(t *testing.T) {
	if !strings.Contains(ExplorePrompt, "symbol definitions with line ranges") {
		t.Error("ExplorePrompt should describe Project Map as structure index")
	}
	if !strings.Contains(ExplorePrompt, "For callers and references, use inspect_symbol") {
		t.Error("ExplorePrompt should direct to inspect_symbol for callers/refs")
	}
	if strings.Contains(ExplorePrompt, "imports") || strings.Contains(ExplorePrompt, "← refs") {
		t.Error("ExplorePrompt should not mention imports or refs")
	}
}

func TestExplorePrompt_SearchCodeNotOverlyRestricted(t *testing.T) {
	if strings.Contains(ExplorePrompt, "ONLY for patterns NOT in Project Map") {
		t.Error("search_code guidance should not be overly restrictive")
	}
	if !strings.Contains(ExplorePrompt, "non-Go references") {
		t.Error("search_code should mention non-Go references as a use case")
	}
}

func TestEditPromptForEditTool_Default(t *testing.T) {
	prompt := EditPromptForEditTool("")
	if !strings.Contains(prompt, "apply_patch") {
		t.Error("default mode should mention apply_patch")
	}
	if strings.Contains(prompt, "str_replace") {
		t.Error("default mode should not mention str_replace")
	}
	if !strings.Contains(prompt, "For callers and references, use inspect_symbol") {
		t.Error("EditPrompt should direct to inspect_symbol for callers/refs")
	}
	if strings.Contains(prompt, "imports") || strings.Contains(prompt, "← refs") {
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
	prompt := EditPromptForEditTool("")
	if strings.Contains(prompt, "ONLY for patterns NOT in Project Map") {
		t.Error("search_code guidance should not be overly restrictive")
	}
}
