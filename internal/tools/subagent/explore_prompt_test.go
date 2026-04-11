package subagent

import (
	"strings"
	"testing"

	promptfragments "github.com/susugadx/xelyon-cli/internal/prompt/fragments"
)

func TestExplorePrompt_UsesSharedInvestigationBlock(t *testing.T) {
	block := promptfragments.BuildInvestigationToolingBlock(promptfragments.InvestigationToolingOptions{
		AllowLowLevelOverrides: false,
		SearchOverrideLabel:    "a low-level expert override",
		ReadOverrideExtra:      "Use it when you already know the exact file or range.",
	})
	for _, want := range []string{
		block,
		promptfragments.SharedChangeGatherContextLine("For shared-symbol or impact investigation, do this before narrow follow-up searches whenever possible."),
		promptfragments.GatherContextFirstLine("The orchestrator must explicitly justify lower-level control."),
		promptfragments.InvestigationMultiPatternLine(false, ""),
	} {
		if !strings.Contains(ExplorePrompt, want) {
			t.Fatalf("ExplorePrompt should embed shared investigation fragment %q", want)
		}
	}
	if strings.Contains(ExplorePrompt, "search_code: code discovery tool") || strings.Contains(ExplorePrompt, "read_file: low-level exact-content reader") {
		t.Fatal("default ExplorePrompt should not advertise hidden low-level investigation tools")
	}
}

func TestExplorePrompt_ProjectMapAssembly(t *testing.T) {
	if !strings.Contains(ExplorePrompt, "symbol definitions with line ranges") {
		t.Error("ExplorePrompt should describe Project Map as structure index")
	}
	if strings.Contains(ExplorePrompt, "imports") || strings.Contains(ExplorePrompt, "← refs") {
		t.Error("ExplorePrompt should not mention imports or refs")
	}
	if strings.Contains(ExplorePrompt, "ONLY for patterns NOT in Project Map") {
		t.Error("ExplorePrompt should not over-restrict search_code")
	}
}

func TestExplorePromptForEditTool_LegacyKeepsLowLevelOverrides(t *testing.T) {
	prompt := ExplorePromptForEditTool("str_replace")
	if !strings.Contains(prompt, "search_code: code discovery tool") {
		t.Fatal("legacy explore prompt should expose search_code guidance")
	}
	if !strings.Contains(prompt, "read_file: low-level exact-content reader") {
		t.Fatal("legacy explore prompt should expose read_file guidance")
	}
}
