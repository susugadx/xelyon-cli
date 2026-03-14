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
	checks := []string{"inspect_symbol", "search_code", "read_file"}
	for _, tool := range checks {
		if !strings.Contains(prompt, tool) {
			t.Errorf("investigation checklist should mention %s", tool)
		}
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
