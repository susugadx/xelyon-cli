package plan

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/investigation"
	promptfragments "github.com/susugadx/xelyon-cli/internal/prompt/fragments"
)

func TestBuildInvestigationPrompt_ContainsGatherContext(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request", investigation.SurfaceEditExactControl)
	if !strings.Contains(prompt, "gather_context") {
		t.Error("investigation prompt should mention gather_context as the default tool")
	}
	// inspect_symbol は search_code に統合済みなので参照しない
	if strings.Contains(prompt, "inspect_symbol") {
		t.Error("investigation prompt should not mention inspect_symbol (integrated into search_code)")
	}
}

func TestBuildInvestigationPrompt_UsesSharedInvestigationBlock(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request", investigation.SurfaceEditExactControl)
	block := promptfragments.BuildInvestigationToolingBlock(promptfragments.InvestigationToolingOptions{
		Surface:                investigation.SurfaceEditExactControl,
		SearchOverrideLabel:    "a low-level expert override",
		ReadOverrideExtra:      "Use it only when you already know the exact file or range and need exact manual control.",
		IncludeBatchRead:       true,
		BatchReadOverrideLabel: "a low-level override",
	})
	for _, want := range []string{
		block,
		promptfragments.SharedChangeGatherContextLine(""),
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("investigation prompt should include shared fragment %q", want)
		}
	}
	if strings.Contains(prompt, promptfragments.LowLevelOverridesWhenExposedLine()) {
		t.Fatal("edit exact-control investigation prompt should not advertise search_code legacy overrides")
	}
}

func TestBuildInvestigationPrompt_BashLimitedToReadOnlyGit(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request", investigation.SurfaceEditExactControl)
	for _, want := range []string{"bash", "git status", "git diff", "git log"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("investigation prompt should mention %q", want)
		}
	}
	for _, forbidden := range []string{"bash (find/read-only)", "build/test/git only"} {
		if strings.Contains(prompt, forbidden) {
			t.Errorf("investigation prompt should not contain %q", forbidden)
		}
	}
}

func TestBuildInvestigationPrompt_ContainsToolSelectionExamples(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request", investigation.SurfaceEditExactControl)
	if !strings.Contains(prompt, "### EXAMPLES") {
		t.Error("investigation prompt should include tool selection examples")
	}
	if !strings.Contains(prompt, `gather_context(query="chatCore"`) {
		t.Error("investigation prompt should include a gather_context example")
	}
	if !strings.Contains(prompt, `gather_context(query="impl.go,impl_test.go", path="pkg", file_filter="go")`) {
		t.Error("investigation prompt should include a scoped combined gather_context example")
	}
}

func TestBuildInvestigationPrompt_LocalVsSharedGuidance(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request", investigation.SurfaceEditExactControl)
	if !strings.Contains(prompt, "local changes") {
		t.Error("investigation checklist should mention local changes")
	}
	if !strings.Contains(prompt, "shared changes") {
		t.Error("investigation checklist should mention shared changes")
	}
	if !strings.Contains(prompt, "Avoid broad exploration") {
		t.Error("investigation checklist should discourage broad exploration when target is clear")
	}
}

func TestBuildInvestigationPrompt_ContainsPlanSchemaWithFiles(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request", investigation.SurfaceEditExactControl)
	assertContainsPlanSchema(t, "investigation prompt", prompt)
}

func TestBuildInvestigationPrompt_LegacyAllowedTools(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request", investigation.SurfaceLegacyOverrides)
	if !strings.Contains(prompt, promptfragments.InvestigationAllowedToolsLine(investigation.SurfaceLegacyOverrides)) {
		t.Error("legacy investigation prompt should list low-level overrides in the allowed surface")
	}
	if !strings.Contains(prompt, promptfragments.LowLevelOverridesWhenExposedLine()) {
		t.Error("legacy investigation prompt should mention low-level overrides when they are visible")
	}
}

func TestBuildInvestigationPrompt_EditExactControlAllowedToolsStayGatherContextFirst(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request", investigation.SurfaceEditExactControl)
	if strings.Contains(prompt, promptfragments.InvestigationAllowedToolsLine(investigation.SurfaceLegacyOverrides)) {
		t.Error("edit exact-control investigation prompt should not list legacy low-level overrides as normally allowed")
	}
	if !strings.Contains(prompt, promptfragments.InvestigationAllowedToolsLine(investigation.SurfaceEditExactControl)) {
		t.Error("edit exact-control investigation prompt should list the visible gather_context/read_file surface")
	}
	if !strings.Contains(prompt, "read_file: exact-content reader for edit/apply_patch exact-control override") {
		t.Error("edit exact-control investigation prompt should position read_file as exact-control only")
	}
}
