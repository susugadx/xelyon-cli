package prompt

import (
	"strings"
	"testing"
)

func TestStripPlanningReferences(t *testing.T) {
	result := StripPlanningReferences(SystemPrompt)

	if strings.Contains(result, "create_plan") {
		t.Error("StripPlanningReferences should remove all create_plan references")
	}
	if strings.Contains(result, "update_plan") {
		t.Error("StripPlanningReferences should remove all update_plan references")
	}
	if !strings.Contains(result, "## Workflow Rules") {
		t.Error("StripPlanningReferences should preserve workflow rules")
	}
}

func TestStripPlanningReferences_Idempotent(t *testing.T) {
	first := StripPlanningReferences(SystemPrompt)
	second := StripPlanningReferences(first)
	if first != second {
		t.Error("StripPlanningReferences should be idempotent")
	}
}

func TestSystemPrompt_BashRules(t *testing.T) {
	if !strings.Contains(SystemPrompt, "NEVER use bash for code investigation") {
		t.Error("SystemPrompt should forbid bash for code investigation")
	}
	if !strings.Contains(SystemPrompt, "bash is ONLY for: build, test, format, lint, git") {
		t.Error("SystemPrompt should restrict bash to build/test/format/lint/git")
	}
	if !strings.Contains(SystemPrompt, "FORBIDDEN") {
		t.Error("SystemPrompt should use FORBIDDEN for bash investigation rules")
	}
}

func TestSystemPrompt_ProjectMapGuidance(t *testing.T) {
	if !strings.Contains(SystemPrompt, "If Project Map is available, check it first") {
		t.Error("SystemPrompt should prefer Project Map first")
	}
	if !strings.Contains(SystemPrompt, "function locations, line counts, and symbol signatures") {
		t.Error("SystemPrompt should mention Project Map symbol signatures")
	}
	if !strings.Contains(SystemPrompt, "go directly to inspect_symbol or read_file; do not search_code first") {
		t.Error("SystemPrompt should skip search_code when Project Map already gives the target location")
	}
	if !strings.Contains(SystemPrompt, "Use list_dir only when you need current filesystem state") {
		t.Error("SystemPrompt should weaken list_dir to current-state checks")
	}
}

func TestSystemPrompt_ToolGuidanceMatchesCurrentSchema(t *testing.T) {
	if !strings.Contains(SystemPrompt, "Go symbol lookup -> inspect_symbol.") {
		t.Error("SystemPrompt should describe inspect_symbol as Go symbol lookup")
	}
	if !strings.Contains(SystemPrompt, "read_file returns full content for all requested files") {
		t.Error("SystemPrompt should describe current read_file behavior")
	}
	forbidden := []string{
		"search_code+read_file",
		"read_file with symbol parameter",
		"do not read_file the same symbol unless the body is truncated",
		"output_mode",
		"manifest",
		"token_budget",
		"summary mode",
		"full mode",
		"start_line",
		"end_line",
		"line range",
		"line-range",
	}
	for _, s := range forbidden {
		if strings.Contains(SystemPrompt, s) {
			t.Errorf("SystemPrompt should not mention removed or stale behavior: %q", s)
		}
	}
}

func TestSystemPrompt_ParallelGuidanceIsConsolidated(t *testing.T) {
	if !strings.Contains(SystemPrompt, "Independent operations -> call multiple tools in one response") {
		t.Error("SystemPrompt should keep one general parallel guidance rule")
	}
	if !strings.Contains(SystemPrompt, "For shared changes, read target code and its callers/tests in parallel when independent") {
		t.Error("SystemPrompt should keep shared-change parallel guidance")
	}
	if !strings.Contains(SystemPrompt, "Reading 2+ independent files -> pass them all in one read_file call") {
		t.Error("SystemPrompt should prefer read_file paths for multi-file reads")
	}
	if !strings.Contains(SystemPrompt, "prefer one search_code call with comma-separated patterns instead of serial searches") {
		t.Error("SystemPrompt should keep multi-pattern search_code guidance")
	}
	if !strings.Contains(SystemPrompt, "delegate to sub-agents via spawn_agent") {
		t.Error("SystemPrompt should guide spawn_agent delegation")
	}
	if !strings.Contains(SystemPrompt, "Use wait_agent to collect results before synthesizing your response") {
		t.Error("SystemPrompt should mention wait_agent collection")
	}
	if !strings.Contains(SystemPrompt, "NEVER specify model or reasoning_effort in spawn_agent") {
		t.Error("SystemPrompt should forbid manual sub-agent model overrides")
	}
	if !strings.Contains(SystemPrompt, "message must be concise") {
		t.Error("SystemPrompt should require concise sub-agent messages")
	}
	if !strings.Contains(SystemPrompt, "do NOT use read_file/search_code/inspect_symbol yourself for the same delegated task") {
		t.Error("SystemPrompt should forbid duplicate local exploration after delegation")
	}
	if !strings.Contains(SystemPrompt, "Fall back to direct tool use ONLY when ALL sub-agents fail") {
		t.Error("SystemPrompt should restrict direct-tool fallback after delegation")
	}
}

func TestSystemPrompt_NoToolSelectionExamples(t *testing.T) {
	if strings.Contains(SystemPrompt, "Tool Selection Examples") {
		t.Error("SystemPrompt should not include tool selection examples")
	}
}

func TestSystemPrompt_LocalVsSharedGuidance(t *testing.T) {
	if !strings.Contains(SystemPrompt, "Local vs shared changes") {
		t.Error("SystemPrompt should distinguish local and shared changes")
	}
	if !strings.Contains(SystemPrompt, "**Shared changes**") {
		t.Error("SystemPrompt should include shared changes guidance")
	}
	if !strings.Contains(SystemPrompt, "**Local changes**") {
		t.Error("SystemPrompt should include local changes guidance")
	}
	if !strings.Contains(SystemPrompt, "Broad reference search is not required") {
		t.Error("SystemPrompt should keep the local-change broad-search guard")
	}
}

func TestSystemPrompt_EfficientExecutionGuidance(t *testing.T) {
	if !strings.Contains(SystemPrompt, "Do not upgrade from targeted read to full-file read unless") {
		t.Error("SystemPrompt should guard against unnecessary full-file reads")
	}
	if !strings.Contains(SystemPrompt, "NEVER re-read a file already returned in full") {
		t.Error("SystemPrompt should forbid rereading files already returned in full")
	}
	if !strings.Contains(SystemPrompt, "Do not search \"just in case\"") {
		t.Error("SystemPrompt should discourage speculative searching")
	}
	if !strings.Contains(SystemPrompt, "read neighboring files speculatively") {
		t.Error("SystemPrompt should discourage speculative neighboring reads")
	}
	if !strings.Contains(SystemPrompt, "str_replace old_str must come from actual inspect_symbol, read_file, or search_code output") {
		t.Error("SystemPrompt should require exact old_str provenance")
	}
}

func TestSystemPrompt_VerificationAndOutputGuidance(t *testing.T) {
	if !strings.Contains(SystemPrompt, "make ci-check") {
		t.Error("SystemPrompt should mention project-defined verification commands")
	}
	if !strings.Contains(SystemPrompt, "Prefer targeted verification first") {
		t.Error("SystemPrompt should prefer targeted verification first")
	}
	if !strings.Contains(SystemPrompt, "Do not rerun the same failing command without a code change") {
		t.Error("SystemPrompt should forbid rerunning failing commands unchanged")
	}
	if !strings.Contains(SystemPrompt, "Give one short progress update only at phase boundaries") {
		t.Error("SystemPrompt should limit progress updates to phase boundaries")
	}
	if !strings.Contains(SystemPrompt, "At most one short progress update per phase") {
		t.Error("SystemPrompt should cap progress updates per phase")
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
