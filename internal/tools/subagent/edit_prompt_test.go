package subagent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/investigation"
	promptfragments "github.com/susugadx/xelyon-cli/internal/prompt/fragments"
)

func TestEditPromptBase_UsesSharedInvestigationBlock(t *testing.T) {
	block := promptfragments.BuildInvestigationToolingBlock(promptfragments.InvestigationToolingOptions{
		Surface:             investigation.SurfaceEditExactControl,
		SearchOverrideLabel: "a low-level expert override",
		ReadOverrideExtra:   "Use it only when you already know the exact file or range and need exact manual control.",
	})
	if !strings.Contains(editPromptBase, block) {
		t.Fatal("editPromptBase should embed the shared investigation block")
	}
	if !strings.Contains(editPromptBase, strings.TrimPrefix(promptfragments.SharedChangeGatherContextLine("If the affected surface is not already clear from the Project Map, known files, or orchestrator-provided scope, do this before narrower follow-up investigation."), "- ")) {
		t.Fatal("editPromptBase should embed shared-change guidance")
	}
	if strings.Contains(editPromptBase, "search_code: code discovery tool") || strings.Contains(editPromptBase, "read_file: low-level exact-content reader") {
		t.Fatal("default editPromptBase should not advertise legacy low-level investigation tools")
	}
	if !strings.Contains(editPromptBase, "read_file: exact-content reader for edit/apply_patch exact-control override") {
		t.Fatal("default editPromptBase should keep read_file exact-control guidance aligned with visible tools")
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
}

func TestEditPromptForEditTool_Legacy(t *testing.T) {
	prompt := EditPromptForEditTool("str_replace")
	if strings.Contains(prompt, "apply_patch") {
		t.Error("legacy mode should not mention apply_patch")
	}
	if !strings.Contains(prompt, "str_replace") {
		t.Error("legacy mode should mention str_replace")
	}
	if !strings.Contains(prompt, promptfragments.LegacyStrReplaceContextSourceLine()) {
		t.Error("legacy mode should allow gather_context as exact edit provenance")
	}
	if !strings.Contains(prompt, "search_code: code discovery tool") || !strings.Contains(prompt, "read_file: low-level exact-content reader") {
		t.Error("legacy mode should keep low-level investigation override guidance")
	}
}

func TestEditPrompt_StillRestrictsEditsToParentScope(t *testing.T) {
	for _, want := range []string{
		"Make ONLY the changes explicitly requested",
		"Do not touch files not mentioned in the task",
		"If the orchestrator already specified the impact surface or target files, do not re-investigate broadly",
	} {
		if !strings.Contains(editPromptBase, want) {
			t.Fatalf("editPromptBase should keep parent-scope restriction %q", want)
		}
	}
}
