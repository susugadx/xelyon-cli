package agent

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestInitializeProjectInstructions_LoadsScopedGuidanceForExistingCwdRelativeSlashPath(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "work")
	writeTestFile(t, filepath.Join(root, "xelyon.yaml"), "context: \"repo\"\n")
	writeTestFile(t, filepath.Join(root, "AGENTS.md"), "# root guidance\n")
	writeTestFile(t, filepath.Join(root, "work", "pkg", "AGENTS.md"), "# work pkg guidance\n")
	writeTestFile(t, filepath.Join(root, "work", "pkg", "x.go"), "package pkg\n")

	agent := initializeProjectInstructionPromptForInput(t, root, cwd, "pkg/x.go を見て")

	assertProjectInstructionPromptHasScope(t, agent.SystemPrompt, "work/pkg")
	if strings.Contains(agent.SystemPrompt, `<repository_instructions scope="pkg" source="pkg/AGENTS.md">`) {
		t.Fatalf("prompt used root-relative missing fallback before existing cwd-relative path:\n%s", agent.SystemPrompt)
	}
}

func TestInitializeProjectInstructions_LoadsScopedGuidanceForMissingCwdRelativeSlashPath(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "work")
	writeTestFile(t, filepath.Join(root, "xelyon.yaml"), "context: \"repo\"\n")
	writeTestFile(t, filepath.Join(root, "AGENTS.md"), "# root guidance\n")
	writeTestFile(t, filepath.Join(root, "pkg", "AGENTS.md"), "# root pkg guidance\n")
	writeTestFile(t, filepath.Join(root, "work", "pkg", "AGENTS.md"), "# work pkg guidance\n")

	agent := initializeProjectInstructionPromptForInput(t, root, cwd, "pkg/new_feature.go を作る")

	assertProjectInstructionPromptHasScope(t, agent.SystemPrompt, "work/pkg")
	if strings.Contains(agent.SystemPrompt, `<repository_instructions scope="pkg" source="pkg/AGENTS.md">`) {
		t.Fatalf("prompt used root-relative missing fallback before cwd-relative missing path:\n%s", agent.SystemPrompt)
	}
}

func TestInitializeProjectInstructions_LoadsScopedGuidanceForExistingAbsoluteInputPath(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "internal", "agent", "foo.go")
	writeTestFile(t, filepath.Join(root, "xelyon.yaml"), "context: \"repo\"\n")
	writeTestFile(t, filepath.Join(root, "AGENTS.md"), "# root guidance\n")
	writeTestFile(t, filepath.Join(root, "internal", "agent", "AGENTS.md"), "# agent guidance\n")
	writeTestFile(t, targetPath, "package agent\n")

	agent := initializeProjectInstructionPromptForInput(t, root, root, targetPath+" を見て")

	assertProjectInstructionPromptHasScope(t, agent.SystemPrompt, "internal/agent")
}

func TestInitializeProjectInstructions_LoadsScopedGuidanceForExistingRootRelativeInputPath(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "xelyon.yaml"), "context: \"repo\"\n")
	writeTestFile(t, filepath.Join(root, "AGENTS.md"), "# root guidance\n")
	writeTestFile(t, filepath.Join(root, "internal", "agent", "AGENTS.md"), "# agent guidance\n")
	writeTestFile(t, filepath.Join(root, "internal", "agent", "foo.go"), "package agent\n")

	agent := initializeProjectInstructionPromptForInput(t, root, root, "internal/agent/foo.go を見て")

	assertProjectInstructionPromptHasScope(t, agent.SystemPrompt, "internal/agent")
}

func TestInitializeProjectInstructions_LoadsScopedGuidanceForMissingInputFile(t *testing.T) {
	root, _ := setupProjectPromptRefreshWorkspace(t)
	writeTestFile(t, filepath.Join(root, "AGENTS.md"), "# root guidance\n")
	writeTestFile(t, filepath.Join(root, "internal", "agent", "AGENTS.md"), "# nested guidance\n")

	stateKey := currentProjectMapStateKey(&Agent{}, root)
	agent := newProjectPromptRefreshTestAgent(stateKey, "", root)
	agent.SystemPrompt = "base\n<!-- PROJECT_CONFIG_ANCHOR -->"
	agent.cfg().ProjectMap.Enabled = false

	if err := agent.promptManager().InitializeProjectInstructions(projectInstructionApplyOptions{
		projectMapInput: "internal/agent/new_feature.go を作る",
	}); err != nil {
		t.Fatalf("InitializeProjectInstructions() error = %v", err)
	}

	if !strings.Contains(agent.SystemPrompt, `<repository_instructions scope="internal/agent" source="internal/agent/AGENTS.md">`) {
		t.Fatalf("initial prompt missing missing-file scoped guidance wrapper:\n%s", agent.SystemPrompt)
	}
}

func TestInitializeProjectInstructions_RejectsUnsafeInputPathsForScopedGuidance(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	outsidePath := filepath.Join(outside, "outside.go")
	writeTestFile(t, filepath.Join(root, "xelyon.yaml"), "context: \"repo\"\n")
	writeTestFile(t, filepath.Join(root, "AGENTS.md"), "# root guidance\n")
	writeTestFile(t, filepath.Join(root, "internal", "agent", "AGENTS.md"), "# agent guidance\n")
	writeTestFile(t, filepath.Join(root, "pkg", "AGENTS.md"), "# pkg guidance\n")
	writeTestFile(t, filepath.Join(root, "pkg", "x.go"), "package pkg\n")
	writeTestFile(t, outsidePath, "package outside\n")

	cases := []struct {
		name           string
		input          string
		forbiddenScope string
	}{
		{
			name:           "outside absolute",
			input:          outsidePath + " を見て",
			forbiddenScope: "pkg",
		},
		{
			name:           "missing in-repo absolute",
			input:          filepath.Join(root, "internal", "agent", "new_feature.go") + " を見て",
			forbiddenScope: "internal/agent",
		},
		{
			name:           "parent traversal segment",
			input:          "internal/../pkg/x.go を見て",
			forbiddenScope: "pkg",
		},
	}
	if runtime.GOOS != "windows" {
		windowsMissing := "C:" + filepath.ToSlash(filepath.Join(root, "internal", "agent", "new_feature.go"))
		cases = append(cases, struct {
			name           string
			input          string
			forbiddenScope string
		}{
			name:           "missing windows absolute",
			input:          windowsMissing + " を見て",
			forbiddenScope: "internal/agent",
		})
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			agent := initializeProjectInstructionPromptForInput(t, root, root, tt.input)

			assertProjectInstructionPromptHasScope(t, agent.SystemPrompt, ".")
			assertProjectInstructionPromptMissingScope(t, agent.SystemPrompt, tt.forbiddenScope)
		})
	}
}

func TestInitializeProjectInstructions_RejectsOutsideSymlinkInputPathForScopedGuidance(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "repo")
	outside := filepath.Join(base, "outside")
	writeTestFile(t, filepath.Join(root, "xelyon.yaml"), "context: \"repo\"\n")
	writeTestFile(t, filepath.Join(root, "AGENTS.md"), "# root guidance\n")
	writeTestFile(t, filepath.Join(outside, "AGENTS.md"), "# outside guidance\n")
	writeTestFile(t, filepath.Join(outside, "outside.go"), "package outside\n")
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	agent := initializeProjectInstructionPromptForInput(t, root, root, "link/outside.go を見て")

	assertProjectInstructionPromptHasScope(t, agent.SystemPrompt, ".")
	assertProjectInstructionPromptMissingScope(t, agent.SystemPrompt, "link")
	if strings.Contains(agent.SystemPrompt, "outside guidance") {
		t.Fatalf("prompt loaded guidance through outside-root symlink:\n%s", agent.SystemPrompt)
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

func initializeProjectInstructionPromptForInput(t *testing.T, root, cwd, input string) *Agent {
	t.Helper()
	stateKey := currentProjectMapStateKey(&Agent{}, root)
	agent := newProjectPromptRefreshTestAgent(stateKey, "", root)
	agent.SystemPrompt = "base\n<!-- PROJECT_CONFIG_ANCHOR -->"
	agent.Runtime.InvocationCWD = cwd
	agent.cfg().ProjectMap.Enabled = false

	if err := agent.promptManager().InitializeProjectInstructions(projectInstructionApplyOptions{
		projectMapInput: input,
	}); err != nil {
		t.Fatalf("InitializeProjectInstructions() error = %v", err)
	}
	return agent
}

func assertProjectInstructionPromptHasScope(t *testing.T, systemPrompt, scope string) {
	t.Helper()
	if !strings.Contains(systemPrompt, `<repository_instructions scope="`+scope+`"`) {
		t.Fatalf("prompt missing scoped guidance %q:\n%s", scope, systemPrompt)
	}
}

func assertProjectInstructionPromptMissingScope(t *testing.T, systemPrompt, scope string) {
	t.Helper()
	if strings.Contains(systemPrompt, `<repository_instructions scope="`+scope+`"`) {
		t.Fatalf("prompt unexpectedly loaded scoped guidance %q:\n%s", scope, systemPrompt)
	}
}
