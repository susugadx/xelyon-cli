package prompt

import (
	"strings"
	"testing"

	promptfragments "github.com/susugadx/xelyon-cli/internal/prompt/fragments"
)

func TestSystemPrompt_ProjectMapGuidance(t *testing.T) {
	if !strings.Contains(SystemPrompt, "Project Map lists file paths, symbol definitions with line ranges for the project.") {
		t.Error("SystemPrompt should describe Project Map as structure index")
	}
	if !strings.Contains(SystemPrompt, `gather_context(query="agent.go:161-328")`) {
		t.Error("SystemPrompt should prefer gather_context for exact Project Map reads")
	}
	if !strings.Contains(SystemPrompt, "Do NOT re-search symbols already listed in Project Map") {
		t.Error("SystemPrompt should skip redundant symbol lookup when Project Map already provides the location")
	}
	if strings.Contains(SystemPrompt, "imports") || strings.Contains(SystemPrompt, "← refs") {
		t.Error("SystemPrompt should not mention imports or refs in Project Map")
	}
}

func TestSystemPrompt_UsesSharedInvestigationFragments(t *testing.T) {
	toolingBlock := promptfragments.BuildInvestigationToolingBlock(promptfragments.InvestigationToolingOptions{
		AllowLowLevelOverrides: false,
		SearchOverrideLabel:    "an expert override",
		SearchOverrideExtra:    `Short symbol queries when possible, and regex only when needed. For related code discovery, multi-pattern search is the default. For shared-change impact analysis starting from one symbol, prefer search_code(intent="impact", pattern="SymbolName") only when gather_context is clearly insufficient.`,
		ReadOverrideExtra:      "Use line ranges from Project Map when exact manual control matters.",
	})
	for _, want := range []string{
		toolingBlock,
		promptfragments.GatherContextFirstLine(""),
		promptfragments.InvestigationMultiPatternLine(false, "For independent patterns and related code discovery, one combined query is preferred over serial narrow searches whenever possible."),
		promptfragments.InvestigationFollowUpLine(false, ""),
		strings.TrimPrefix(promptfragments.SharedChangeGatherContextLine("Then edit once the affected files are clear."), "- "),
	} {
		if !strings.Contains(SystemPrompt, want) {
			t.Fatalf("SystemPrompt should embed shared investigation fragment %q", want)
		}
	}
}

func TestSystemPrompt_WorkflowRules(t *testing.T) {
	checks := []string{
		"NEVER use bash for code investigation",
		"bash is ONLY for: build, test, format, lint, git",
		"Local vs shared changes",
		"Broad reference search is not required",
		"Do not upgrade from targeted read to full-file read unless",
		"NEVER re-read a file already returned in full",
		"Do not search \"just in case\"",
		"Use exact context from actual visible investigation tool output in this session",
		"make ci-check",
		"Prefer targeted verification first",
		"Do not rerun the same failing command without a code change",
		"Give one short progress update only at phase boundaries",
		"At most one short progress update per phase",
	}
	for _, check := range checks {
		if !strings.Contains(SystemPrompt, check) {
			t.Errorf("SystemPrompt missing workflow rule %q", check)
		}
	}
}

func TestSystemPrompt_ParallelGuidanceIsConsolidated(t *testing.T) {
	for _, want := range []string{
		"Independent operations -> call multiple tools in one response",
		"For shared changes, gather the target code and its callers/tests in parallel when independent",
		"Sub-agents are fetch tools, not decision-makers",
		"Use wait_agent to collect results before synthesizing your response",
		"do NOT repeat the same investigation yourself with gather_context or other visible read-only tools",
		"Fall back to direct tool use ONLY when ALL sub-agents fail",
		"SINGLE response as parallel tool calls",
		"Skip sub-agents for simple tasks",
		"Fetch",
	} {
		if !strings.Contains(SystemPrompt, want) {
			t.Errorf("SystemPrompt missing consolidated parallel guidance %q", want)
		}
	}
}

func TestSystemPrompt_DefaultSurfaceAvoidsHiddenLowLevelOverrides(t *testing.T) {
	for _, forbidden := range []string{
		"search_code: code discovery tool",
		"read_file: low-level exact-content reader",
		"search_code(intent=\"impact\"",
		"Reading 2+ independent exact files -> read_file can batch",
	} {
		if strings.Contains(SystemPrompt, forbidden) {
			t.Fatalf("default SystemPrompt should not advertise hidden low-level investigation tool %q", forbidden)
		}
	}
}

func TestSystemPrompt_NoToolSelectionExamples(t *testing.T) {
	if strings.Contains(SystemPrompt, "Tool Selection Examples") {
		t.Error("SystemPrompt should not include tool selection examples")
	}
}

func TestSystemPrompt_NoBashRecommendations(t *testing.T) {
	forbidden := []string{
		"bash (grep)",
		"NOT bash cat",
		"NOT bash ls",
		"Use bash (grep)",
		"Use bash (find",
		"bash (find/read-only)",
	}
	for _, f := range forbidden {
		if strings.Contains(SystemPrompt, f) {
			t.Errorf("SystemPrompt should not recommend bash investigation patterns: %q", f)
		}
	}
}

func TestSystemPrompt_ReviewInvestigationGuidanceStaysReadOnly(t *testing.T) {
	for _, want := range []string{
		"Review or investigation request: do not modify files unless asked.",
		"Prefer read-only reproduction: use existing tests, focused verification commands, and actual visible tool output.",
		"say so and wait for explicit permission to modify files",
	} {
		if !strings.Contains(SystemPrompt, want) {
			t.Fatalf("SystemPrompt missing read-only review guidance %q", want)
		}
	}
	if strings.Contains(SystemPrompt, "write a temporary test inside the target package") {
		t.Fatal("SystemPrompt should not require temporary test creation during read-only review/investigation")
	}
}
