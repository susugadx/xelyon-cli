package plan

import (
	"strings"
	"testing"
)

func TestBuildInvestigationPrompt_ContainsInspectSymbol(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request")
	if !strings.Contains(prompt, "inspect_symbol") {
		t.Error("investigation prompt should mention inspect_symbol as an allowed tool")
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
	checks := []string{"inspect_symbol", "search_code", "read_file", "list_dir"}
	for _, tool := range checks {
		if !strings.Contains(prompt, tool) {
			t.Errorf("investigation checklist should mention %s", tool)
		}
	}
}

func TestBuildInvestigationPrompt_UsesInspectSymbolForGoSymbols(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request")
	if !strings.Contains(prompt, "Use inspect_symbol for Go symbol lookup") {
		t.Error("investigation prompt should describe inspect_symbol as the Go symbol lookup tool")
	}
	if strings.Contains(prompt, "search_code+read_file") {
		t.Error("investigation prompt should not compare inspect_symbol against search_code+read_file")
	}
}

func TestBuildInvestigationPrompt_PrefersParallelInvestigation(t *testing.T) {
	prompt := BuildInvestigationPrompt("test request")
	if !strings.Contains(prompt, "Prefer parallel investigation") {
		t.Error("investigation prompt should encourage parallel investigation")
	}
	if !strings.Contains(prompt, "call read_file multiple times in the same turn") {
		t.Error("investigation prompt should prefer multiple read_file calls in the same turn")
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
	if !strings.Contains(prompt, "inspect_symbol(symbol=\"chatCore\", path=\"internal/agent/agent_chat.go\")") {
		t.Error("investigation prompt should include an inspect_symbol example")
	}
	if !strings.Contains(prompt, "use list_dir first") {
		t.Error("investigation prompt should include a list_dir-first example")
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
