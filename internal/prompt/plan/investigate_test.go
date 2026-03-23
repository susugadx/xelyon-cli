package plan

import (
	"strings"
	"testing"
)

func TestBuildInvestigationPrompt_ContainsSearchCode(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request")
	if !strings.Contains(prompt, "search_code") {
		t.Error("investigation prompt should mention search_code as an allowed tool")
	}
	// inspect_symbol は search_code に統合済みなので参照しない
	if strings.Contains(prompt, "inspect_symbol") {
		t.Error("investigation prompt should not mention inspect_symbol (integrated into search_code)")
	}
	// list_dir は除外済みなので参照しない
	if strings.Contains(prompt, "list_dir") {
		t.Error("investigation prompt should not mention list_dir (excluded from registry)")
	}
}

func TestBuildInvestigationPrompt_BashLimitedToReadOnlyGit(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request")
	// bash should be listed but restricted to read-only git commands
	if !strings.Contains(prompt, "bash") {
		t.Error("investigation prompt should still mention bash (for read-only git)")
	}
	// bash should NOT be suggested for find/read-only code investigation
	if strings.Contains(prompt, "bash (find/read-only)") {
		t.Error("investigation prompt should not suggest bash for find/read-only code investigation")
	}
	// bash should NOT allow build/test (not strictly read-only)
	if strings.Contains(prompt, "build/test/git only") {
		t.Error("investigation prompt bash should not allow build/test under READ-ONLY ONLY")
	}
	// should mention specific read-only git commands
	if !strings.Contains(prompt, "git status") || !strings.Contains(prompt, "git diff") || !strings.Contains(prompt, "git log") {
		t.Error("investigation prompt should list specific read-only git commands")
	}
}

func TestBuildInvestigationPrompt_DedicatedToolsInChecklist(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request")
	checks := []string{"search_code", "read_file"}
	for _, tool := range checks {
		if !strings.Contains(prompt, tool) {
			t.Errorf("investigation checklist should mention %s", tool)
		}
	}
}

func TestBuildInvestigationPrompt_SearchCodeForGoSymbols(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request")
	if !strings.Contains(prompt, "For Go symbols, it automatically returns callers, references, and tests") {
		t.Error("investigation prompt should describe search_code Go symbol auto-resolution")
	}
	if strings.Contains(prompt, "search_code+read_file") {
		t.Error("investigation prompt should not compare search_code against search_code+read_file")
	}
}

func TestBuildInvestigationPrompt_PrefersParallelInvestigation(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request")
	if !strings.Contains(prompt, "Prefer parallel investigation") {
		t.Error("investigation prompt should encourage parallel investigation")
	}
	if !strings.Contains(prompt, "one read_file call with all paths") {
		t.Error("investigation prompt should prefer read_file paths for multi-file reads")
	}
	if !strings.Contains(prompt, "prefer one search_code call with comma-separated patterns") {
		t.Error("investigation prompt should prefer multi-pattern search_code")
	}
}

func TestBuildInvestigationPrompt_ContainsToolSelectionExamples(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request")
	if !strings.Contains(prompt, "### EXAMPLES") {
		t.Error("investigation prompt should include tool selection examples")
	}
	if !strings.Contains(prompt, `search_code(pattern="chatCore"`) {
		t.Error("investigation prompt should include a search_code example")
	}
	if !strings.Contains(prompt, `read_file(paths=["impl.go", "impl_test.go"])`) {
		t.Error("investigation prompt should include a read_file paths example")
	}
}

func TestBuildInvestigationPrompt_LocalVsSharedGuidance(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request")
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
