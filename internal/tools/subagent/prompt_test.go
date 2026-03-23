package subagent

import (
	"strings"
	"testing"
)

func TestExplorePrompt_ProjectMapGuidance(t *testing.T) {
	if !strings.Contains(ExplorePrompt, "symbol definitions with line ranges") {
		t.Error("ExplorePrompt should describe Project Map as structure index")
	}
	if !strings.Contains(ExplorePrompt, "search_code: for all code discovery") {
		t.Error("ExplorePrompt should describe search_code for all code discovery")
	}
	if strings.Contains(ExplorePrompt, "imports") || strings.Contains(ExplorePrompt, "← refs") {
		t.Error("ExplorePrompt should not mention imports or refs")
	}
}

func TestExplorePrompt_SearchCodeNotOverlyRestricted(t *testing.T) {
	if strings.Contains(ExplorePrompt, "ONLY for patterns NOT in Project Map") {
		t.Error("search_code guidance should not be overly restrictive")
	}
	if !strings.Contains(ExplorePrompt, "For Go symbols, automatically returns callers and references") {
		t.Error("search_code should describe Go symbol auto-resolution")
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
	if !strings.Contains(prompt, "search_code: for all code discovery") {
		t.Error("EditPrompt should describe search_code for all code discovery")
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
