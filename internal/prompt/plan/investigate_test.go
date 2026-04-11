package plan

import (
	"strings"
	"testing"

	promptfragments "github.com/susugadx/xelyon-cli/internal/prompt/fragments"
)

func TestBuildInvestigationPrompt_ContainsGatherContext(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request", false)
	if !strings.Contains(prompt, "gather_context") {
		t.Error("investigation prompt should mention gather_context as the default tool")
	}
	// inspect_symbol は search_code に統合済みなので参照しない
	if strings.Contains(prompt, "inspect_symbol") {
		t.Error("investigation prompt should not mention inspect_symbol (integrated into search_code)")
	}
}

func TestBuildInvestigationPrompt_UsesSharedInvestigationBlock(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request", false)
	block := promptfragments.BuildInvestigationToolingBlock(promptfragments.InvestigationToolingOptions{
		AllowLowLevelOverrides: false,
		SearchOverrideLabel:    "a low-level expert override",
		ReadOverrideExtra:      "Use it only when you already know the exact file or range.",
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
		t.Fatal("default investigation prompt should not advertise hidden low-level overrides")
	}
}

func TestBuildInvestigationPrompt_BashLimitedToReadOnlyGit(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request", false)
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
	prompt := BuildInvestigationPrompt("test request", false)
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
	prompt := BuildInvestigationPrompt("test request", false)
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

func TestBuildInvestigationPrompt_LegacyAllowedTools(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request", true)
	if !strings.Contains(prompt, promptfragments.InvestigationAllowedToolsLine(true)) {
		t.Error("legacy investigation prompt should list low-level overrides in the allowed surface")
	}
	if !strings.Contains(prompt, promptfragments.LowLevelOverridesWhenExposedLine()) {
		t.Error("legacy investigation prompt should mention low-level overrides when they are visible")
	}
}

func TestBuildInvestigationPrompt_DefaultAllowedToolsStayGatherContextFirst(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request", false)
	if strings.Contains(prompt, promptfragments.InvestigationAllowedToolsLine(true)) {
		t.Error("default investigation prompt should not list low-level overrides as normally allowed")
	}
	if !strings.Contains(prompt, promptfragments.InvestigationAllowedToolsLine(false)) {
		t.Error("default investigation prompt should list gather_context-first allowed tools")
	}
}
