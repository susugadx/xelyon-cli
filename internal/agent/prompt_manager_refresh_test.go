package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectPromptRefreshDecision_InstructionBundleChangeTriggersRefresh(t *testing.T) {
	root, _ := setupProjectPromptRefreshWorkspace(t)
	guidancePath := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(guidancePath, []byte("# guidance v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stateKey := currentProjectMapStateKey(&Agent{}, root)
	agent := newProjectPromptRefreshTestAgent(stateKey, "", root)
	agent.loadProjectInstructionBundleCached(true)

	if err := os.WriteFile(guidancePath, []byte("# guidance v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	decision := agent.promptManager().ProjectPromptRefreshDecision("実装方針を整理して")
	if !decision.NeedRefresh {
		t.Fatal("expected guidance change to trigger refresh")
	}
	if decision.Reason != refreshReasonInstructionChanged {
		t.Fatalf("decision.Reason = %q, want %q", decision.Reason, refreshReasonInstructionChanged)
	}
}

func TestProjectPromptRefreshDecision_InstructionBundleChangeTriggersRefreshWhenProjectMapDisabled(t *testing.T) {
	root, _ := setupProjectPromptRefreshWorkspace(t)
	guidancePath := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(guidancePath, []byte("# guidance v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stateKey := currentProjectMapStateKey(&Agent{}, root)
	agent := newProjectPromptRefreshTestAgent(stateKey, "", root)
	agent.cfg().ProjectMap.Enabled = false
	agent.loadProjectInstructionBundleCached(true)

	if err := os.WriteFile(guidancePath, []byte("# guidance v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	decision := agent.promptManager().ProjectPromptRefreshDecision("実装方針を整理して")
	if !decision.NeedRefresh {
		t.Fatal("expected guidance change to trigger refresh even when project map is disabled")
	}
	if decision.Reason != refreshReasonInstructionChanged {
		t.Fatalf("decision.Reason = %q, want %q", decision.Reason, refreshReasonInstructionChanged)
	}
}

func TestProjectPromptRefreshDecision_InputSpecificGuidanceSelectionTriggersRefresh(t *testing.T) {
	root, _ := setupProjectPromptRefreshWorkspace(t)
	writeTestFile(t, filepath.Join(root, "AGENTS.md"), "# root guidance\n")
	writeTestFile(t, filepath.Join(root, "internal", "agent", "AGENTS.md"), "# nested guidance\n")

	stateKey := currentProjectMapStateKey(&Agent{}, root)
	agent := newProjectPromptRefreshTestAgent(stateKey, "", root)
	agent.cfg().ProjectMap.Enabled = false
	agent.loadProjectInstructionBundleCachedWithInput(true, "")

	decision := agent.promptManager().ProjectPromptRefreshDecision("internal/agent/compress.go を見て")
	if !decision.NeedRefresh {
		t.Fatal("expected input-selected nested guidance to trigger refresh")
	}
	if decision.Reason != refreshReasonInstructionChanged {
		t.Fatalf("decision.Reason = %q, want %q", decision.Reason, refreshReasonInstructionChanged)
	}
}

func TestRefreshProjectPrompt_LoadsInputSpecificScopedGuidance(t *testing.T) {
	root, _ := setupProjectPromptRefreshWorkspace(t)
	writeTestFile(t, filepath.Join(root, "AGENTS.md"), "# root guidance\n")
	writeTestFile(t, filepath.Join(root, "internal", "agent", "AGENTS.md"), "# nested guidance\n")

	stateKey := currentProjectMapStateKey(&Agent{}, root)
	agent := newProjectPromptRefreshTestAgent(stateKey, "", root)
	agent.SystemPrompt = "base\n<!-- PROJECT_CONFIG_ANCHOR -->"
	agent.cfg().ProjectMap.Enabled = false
	agent.loadProjectInstructionBundleCachedWithInput(true, "")

	agent.refreshProjectPrompt("internal/agent/compress.go を見て")

	if !strings.Contains(agent.SystemPrompt, `<repository_instructions scope="internal/agent" source="internal/agent/AGENTS.md">`) {
		t.Fatalf("refreshed prompt missing nested guidance wrapper:\n%s", agent.SystemPrompt)
	}
}

func TestRefreshProjectPrompt_LoadsGuidanceForMoreThanFiveReferencedPaths(t *testing.T) {
	root, _ := setupProjectPromptRefreshWorkspace(t)
	writeTestFile(t, filepath.Join(root, "AGENTS.md"), "# root guidance\n")

	refs := make([]string, 0, 6)
	areas := []string{"area1", "area2", "area3", "area4", "area5", "area6"}
	for _, area := range areas {
		dir := filepath.Join(root, "pkg", area)
		writeTestFile(t, filepath.Join(dir, "AGENTS.md"), "# scoped guidance\n")
		writeTestFile(t, filepath.Join(dir, "target.go"), "package area\n")
		refs = append(refs, "pkg/"+area+"/target.go")
	}

	stateKey := currentProjectMapStateKey(&Agent{}, root)
	agent := newProjectPromptRefreshTestAgent(stateKey, "", root)
	agent.SystemPrompt = "base\n<!-- PROJECT_CONFIG_ANCHOR -->"
	agent.cfg().ProjectMap.Enabled = false
	agent.loadProjectInstructionBundleCachedWithInput(true, "")

	agent.refreshProjectPrompt(strings.Join(refs, " "))

	if !strings.Contains(agent.SystemPrompt, `<repository_instructions scope="pkg/area6" source="pkg/area6/AGENTS.md">`) {
		t.Fatalf("refreshed prompt missing guidance for sixth referenced path:\n%s", agent.SystemPrompt)
	}
}
