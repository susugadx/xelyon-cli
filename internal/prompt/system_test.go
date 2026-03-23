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
	if !strings.Contains(SystemPrompt, "Project Map lists file paths, symbol definitions with line ranges for the project.") {
		t.Error("SystemPrompt should describe Project Map as structure index")
	}
	if !strings.Contains(SystemPrompt, "Do NOT call search_code to find symbols already listed in Project Map") {
		t.Error("SystemPrompt should skip search_code when symbol is in Project Map")
	}
	if !strings.Contains(SystemPrompt, "search_code: code discovery tool") {
		t.Error("SystemPrompt should describe search_code as code discovery tool")
	}
	if strings.Contains(SystemPrompt, "imports") || strings.Contains(SystemPrompt, "← refs") {
		t.Error("SystemPrompt should not mention imports or refs in Project Map")
	}
}

func TestSystemPrompt_ToolGuidanceMatchesCurrentSchema(t *testing.T) {
	if !strings.Contains(SystemPrompt, "search_code: code discovery tool") {
		t.Error("SystemPrompt should describe search_code as code discovery tool")
	}
	if !strings.Contains(SystemPrompt, "read_file: to read actual file contents. Use line ranges from Project Map.") {
		t.Error("SystemPrompt should describe current read_file behavior")
	}
	if !strings.Contains(SystemPrompt, "### apply_patch (edit tool)") {
		t.Error("default SystemPrompt should include the apply_patch guide")
	}
	if strings.Contains(SystemPrompt, "start_line") || strings.Contains(SystemPrompt, "end_line") {
		t.Error("SystemPrompt should not describe read_file with start_line/end_line parameters")
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
	if !strings.Contains(SystemPrompt, "sub-agents via spawn_agent") {
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
	if !strings.Contains(SystemPrompt, "NEVER set timeout_ms in wait_agent") {
		t.Error("SystemPrompt should forbid short-circuit wait_agent timeouts")
	}
	if !strings.Contains(SystemPrompt, "do NOT use read_file/search_code yourself for the same delegated task") {
		t.Error("SystemPrompt should forbid duplicate local exploration after delegation")
	}
	if !strings.Contains(SystemPrompt, "Fall back to direct tool use ONLY when ALL sub-agents fail") {
		t.Error("SystemPrompt should restrict direct-tool fallback after delegation")
	}
	if !strings.Contains(SystemPrompt, "SINGLE response as parallel tool calls") {
		t.Error("SystemPrompt should require all spawn_agent calls in a single response")
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
	if !strings.Contains(SystemPrompt, "Use exact context from actual read_file or search_code output") {
		t.Error("SystemPrompt should require exact edit-context provenance")
	}
}

func TestCurrentSystemPrompt_LegacyEditToolMode(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	prompt := CurrentSystemPrompt()
	if !strings.Contains(prompt, "### Legacy edit tools") {
		t.Error("legacy mode prompt should include legacy edit tool guidance")
	}
	if strings.Contains(prompt, "### apply_patch (edit tool)") {
		t.Error("legacy mode prompt should not include apply_patch guide")
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
