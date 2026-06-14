package prompt

import (
	"strings"
	"testing"
)

func TestInjectProjectRules_Normal(t *testing.T) {
	systemPrompt := `## Workflow Rules

### 10. Verification Protocol (MANDATORY)
A task is NOT complete until verification passes

### 11. Impact Analysis
Check references before changes`

	rulesBlock := "\n\n=== PROJECT-SPECIFIC RULES (MANDATORY) ===\nTest rule\nViolating ANY of these rules is a critical failure."

	result := InjectProjectRules(systemPrompt, rulesBlock)

	// Rule #10 の直後に挿入されること
	idx10 := strings.Index(result, "verification passes")
	idxRules := strings.Index(result, "PROJECT-SPECIFIC RULES")
	idx11 := strings.Index(result, "### 11. Impact Analysis")

	if idxRules < 0 {
		t.Fatal("rules block not found in result")
	}
	if idxRules < idx10 {
		t.Error("rules block should come AFTER Rule #10")
	}
	if idx11 < idxRules {
		t.Error("rules block should come BEFORE Rule #11")
	}
}

func TestBuildProjectConfigBlock(t *testing.T) {
	block := BuildProjectConfigBlock(
		[]string{"Always run tests"},
		[]string{"base context", "### Agent\nagent context"},
	)

	if !strings.Contains(block, "<!-- PROJECT_CONFIG_START -->") {
		t.Fatal("project config start marker missing")
	}
	if !strings.Contains(block, "PROJECT-SPECIFIC RULES") {
		t.Fatal("rules block missing")
	}
	if !strings.Contains(block, "## Project Context:") {
		t.Fatal("project context header missing")
	}
	if !strings.Contains(block, "### Agent\nagent context") {
		t.Fatal("conditional context missing")
	}
}

func TestInjectProjectConfigBlockAndStrip(t *testing.T) {
	systemPrompt := `## Workflow Rules

### 10. Verification Protocol (MANDATORY)
A task is NOT complete until verification passes

### 11. Impact Analysis
Check references before changes`
	block := BuildProjectConfigBlock([]string{"Always run tests"}, []string{"base context"})

	result := InjectProjectConfigBlock(systemPrompt, block)
	if !strings.Contains(result, "<!-- PROJECT_CONFIG_START -->") {
		t.Fatal("project config block missing")
	}

	stripped := StripProjectConfigSections(result)
	if stripped != systemPrompt {
		t.Fatalf("StripProjectConfigSections() = %q, want %q", stripped, systemPrompt)
	}
}

func TestInjectProjectRules_EmptyBlock(t *testing.T) {
	systemPrompt := "original prompt"
	result := InjectProjectRules(systemPrompt, "")
	if result != systemPrompt {
		t.Error("should return original prompt when block is empty")
	}
}

func TestInjectProjectRules_MarkerNotFound(t *testing.T) {
	systemPrompt := "some prompt without the marker"
	rulesBlock := "\n\nRULES HERE"

	result := InjectProjectRules(systemPrompt, rulesBlock)
	// マーカーがなければ末尾に追加
	if !strings.Contains(result, rulesBlock) {
		t.Error("should append to end when marker not found")
	}
}

func TestInjectProjectConfigBlock_UsesVerificationSectionAnchor(t *testing.T) {
	systemPrompt := `## Workflow Rules

### 10. Verification Protocol (MANDATORY)
Verification section body changed and no legacy marker exists.

### 11. Impact Analysis
Check references before changes`
	projectBlock := "\n\n<!-- PROJECT_CONFIG_START -->\nproject rules\n<!-- PROJECT_CONFIG_END -->"

	result := InjectProjectConfigBlock(systemPrompt, projectBlock)
	idxRules := strings.Index(result, "PROJECT_CONFIG_START")
	idxSection10 := strings.Index(result, "### 10. Verification Protocol")
	idxSection11 := strings.Index(result, "### 11. Impact Analysis")

	if idxRules < 0 {
		t.Fatal("project block was not injected")
	}
	if idxRules < idxSection10 || idxSection11 < idxRules {
		t.Fatalf("project block should be injected between section #10 and #11: %s", result)
	}
}

func TestBuildRulesBlockFromList_Empty(t *testing.T) {
	result := BuildRulesBlockFromList(nil)
	if result != "" {
		t.Errorf("expected empty string for nil rules, got %q", result)
	}
	result = BuildRulesBlockFromList([]string{})
	if result != "" {
		t.Errorf("expected empty string for empty rules, got %q", result)
	}
}

func TestBuildRulesBlockFromList_Single(t *testing.T) {
	result := BuildRulesBlockFromList([]string{"Always run tests"})
	if !strings.Contains(result, "PROJECT-SPECIFIC RULES (MANDATORY)") {
		t.Error("should contain MANDATORY header")
	}
	if !strings.Contains(result, "1. Always run tests") {
		t.Error("should contain numbered rule")
	}
	if !strings.Contains(result, "Violating ANY") {
		t.Error("should contain violation warning")
	}
}

func TestBuildRulesBlockFromList_Multiple(t *testing.T) {
	rules := []string{
		"Run go fmt before commit",
		"All tests must pass",
		"No hardcoded secrets",
	}
	result := BuildRulesBlockFromList(rules)
	if !strings.Contains(result, "1. Run go fmt before commit") {
		t.Error("should contain rule 1")
	}
	if !strings.Contains(result, "2. All tests must pass") {
		t.Error("should contain rule 2")
	}
	if !strings.Contains(result, "3. No hardcoded secrets") {
		t.Error("should contain rule 3")
	}
}

func TestBuildProjectInstructionBlock_NoXelyonUsesProjectGuidanceLanguage(t *testing.T) {
	block := BuildProjectInstructionBlock(ProjectInstructionBlockInput{
		ProjectGuidance: []ProjectInstructionEntry{
			{Label: "AGENTS.md", Content: "Follow repo guidance strictly.", Strength: "project_guidance"},
		},
	})

	if !strings.Contains(block, "<!-- PROJECT_CONFIG_START -->") || !strings.Contains(block, "<!-- PROJECT_CONFIG_END -->") {
		t.Fatal("project config markers should exist")
	}
	if strings.Contains(block, "PROJECT-SPECIFIC RULES (MANDATORY)") {
		t.Fatal("mandatory block should not be generated when xelyon.yaml rules are absent")
	}
	if !strings.Contains(block, "No legacy xelyon.yaml rules/context are active for this turn.") {
		t.Fatal("missing no-legacy guidance explanation")
	}
	if !strings.Contains(block, "### AGENTS.md") {
		t.Fatal("missing guidance heading")
	}
}

func TestBuildProjectInstructionBlock_NoLegacyInstructionsKeepProjectGuidanceLanguage(t *testing.T) {
	block := BuildProjectInstructionBlock(ProjectInstructionBlockInput{
		ProjectGuidance: []ProjectInstructionEntry{
			{Label: "AGENTS.md", Content: "Follow repo guidance strictly.", Strength: "project_guidance"},
		},
	})

	if !strings.Contains(block, "No legacy xelyon.yaml rules/context are active for this turn.") {
		t.Fatal("missing no-legacy guidance explanation")
	}
	if strings.Contains(block, "The following project guidance files are treated as advisory guidance.") {
		t.Fatalf("no legacy instructions should not render advisory guidance explanation:\n%s", block)
	}
	if !strings.Contains(block, "### AGENTS.md (project guidance)") {
		t.Fatalf("project guidance heading missing:\n%s", block)
	}
}

func TestBuildProjectInstructionBlock_XelyonRulesRemainMandatory(t *testing.T) {
	block := BuildProjectInstructionBlock(ProjectInstructionBlockInput{
		MandatoryRules: []string{"Run go test ./..."},
		ProjectGuidance: []ProjectInstructionEntry{
			{Label: "CLAUDE.md", Content: "Advisory style note.", Strength: "project_guidance"},
		},
	})

	if !strings.Contains(block, "PROJECT-SPECIFIC RULES (MANDATORY)") {
		t.Fatal("mandatory rules header missing")
	}
	if !strings.Contains(block, "1. Run go test ./...") {
		t.Fatal("mandatory rule missing")
	}
	if !strings.Contains(block, "advisory guidance") {
		t.Fatal("advisory explanation missing for project guidance with xelyon")
	}
	if strings.Contains(block, "1. Advisory style note.") {
		t.Fatal("guidance content should not be converted into mandatory numbered rules")
	}
	if !strings.Contains(block, "### CLAUDE.md (advisory)") {
		t.Fatal("missing CLAUDE.md heading")
	}
}

func TestBuildProjectInstructionBlock_GlobalGuidanceIsAdvisory(t *testing.T) {
	block := BuildProjectInstructionBlock(ProjectInstructionBlockInput{
		GlobalGuidance: []ProjectInstructionEntry{
			{Label: "~/.xelyon/AGENTS.md", Content: "Personal preference.", Strength: "advisory"},
		},
	})

	if !strings.Contains(block, "## Enabled Global Guidance") {
		t.Fatal("global guidance section missing")
	}
	if !strings.Contains(block, "Global guidance is advisory personal preference.") {
		t.Fatal("global advisory explanation missing")
	}
	if !strings.Contains(block, "### ~/.xelyon/AGENTS.md (advisory)") {
		t.Fatal("global guidance file heading missing")
	}
}

func TestBuildProjectInstructionBlock_RendersStrengthAndWarnings(t *testing.T) {
	block := BuildProjectInstructionBlock(ProjectInstructionBlockInput{
		ProjectGuidance: []ProjectInstructionEntry{
			{Label: "AGENTS.md", Content: "Follow strict policy.", Strength: "project_guidance"},
		},
		Warnings: []string{
			"Skipped invalid project guidance path: ../outside.md",
		},
	})

	if !strings.Contains(block, "### AGENTS.md (project guidance)") {
		t.Fatal("project guidance strength label missing")
	}
	if !strings.Contains(block, "## Guidance Load Notes") {
		t.Fatal("guidance load notes section missing")
	}
	if !strings.Contains(block, "Skipped invalid project guidance path: ../outside.md") {
		t.Fatal("guidance warning missing")
	}
}

func TestBuildProjectInstructionBlock_NoSignalsReturnsEmpty(t *testing.T) {
	block := BuildProjectInstructionBlock(ProjectInstructionBlockInput{
		Warnings: []string{"   "},
	})
	if block != "" {
		t.Fatalf("expected empty block, got: %q", block)
	}
}
