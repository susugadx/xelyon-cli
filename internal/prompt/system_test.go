package prompt

import (
	"strings"
	"testing"
)

func TestStripPlanningReferences(t *testing.T) {
	result := StripPlanningReferences(SystemPrompt)

	// create_plan / update_plan への参照が除去されていること
	if strings.Contains(result, "create_plan") {
		t.Error("StripPlanningReferences should remove all create_plan references")
	}
	if strings.Contains(result, "update_plan") {
		t.Error("StripPlanningReferences should remove all update_plan references")
	}

	// Workflow Rules 全体は残っていること
	if !strings.Contains(result, "## Workflow Rules") {
		t.Error("StripPlanningReferences should preserve ## Workflow Rules")
	}
}

func TestStripPlanningReferences_Idempotent(t *testing.T) {
	first := StripPlanningReferences(SystemPrompt)
	second := StripPlanningReferences(first)
	if first != second {
		t.Error("StripPlanningReferences should be idempotent")
	}
}

func TestSystemPrompt_BashForbiddenForCodeInvestigation(t *testing.T) {
	// bash がコード調査に使用禁止であることが明記されている
	if !strings.Contains(SystemPrompt, "NEVER use bash for code investigation") {
		t.Error("SystemPrompt should explicitly forbid bash for code investigation")
	}
	if !strings.Contains(SystemPrompt, "FORBIDDEN") {
		t.Error("SystemPrompt should use FORBIDDEN for bash code investigation tools")
	}
}

func TestSystemPrompt_BashAllowedForBuildTestGit(t *testing.T) {
	// bash がビルド・テスト・git 用途では許可されている
	if !strings.Contains(SystemPrompt, "bash is ONLY for: build, test, format, lint, git") {
		t.Error("SystemPrompt should specify bash is only for build/test/format/lint/git")
	}
}

func TestSystemPrompt_DedicatedToolsExplicit(t *testing.T) {
	// 各専用ツールが明示されている
	checks := []struct {
		tool string
		desc string
	}{
		{"inspect_symbol", "known symbol investigation"},
		{"search_code", "code search / regex discovery"},
		{"read_file", "file contents"},
		{"list_dir", "directory listing"},
	}
	for _, c := range checks {
		if !strings.Contains(SystemPrompt, c.tool) {
			t.Errorf("SystemPrompt should mention %s for %s", c.tool, c.desc)
		}
	}
}

func TestSystemPrompt_InspectSymbolPreferredForExactGoSymbols(t *testing.T) {
	if !strings.Contains(SystemPrompt, "Known exact Go symbol name -> inspect_symbol FIRST") {
		t.Error("SystemPrompt should prefer inspect_symbol when the exact Go symbol is known")
	}
	if !strings.Contains(SystemPrompt, "instead of search_code+read_file") {
		t.Error("SystemPrompt should state inspect_symbol replaces search_code+read_file for exact Go symbols")
	}
	if !strings.Contains(SystemPrompt, "do not read_file the same symbol unless the body is truncated") {
		t.Error("SystemPrompt should discourage redundant read_file after inspect_symbol")
	}
}

func TestSystemPrompt_ListDirPreferredForFilesystemExploration(t *testing.T) {
	if !strings.Contains(SystemPrompt, "Use list_dir for current filesystem exploration") {
		t.Error("SystemPrompt should prefer list_dir for filesystem exploration")
	}
	if !strings.Contains(SystemPrompt, "Project Map is a hint, not a replacement") {
		t.Error("SystemPrompt should clarify Project Map does not replace list_dir for current filesystem state")
	}
	if !strings.Contains(SystemPrompt, "Directory listing -> list_dir; use it first for current filesystem contents") {
		t.Error("SystemPrompt should prefer list_dir first for filesystem contents and next file choice")
	}
}

func TestSystemPrompt_PrefersParallelInvestigationForIndependentSteps(t *testing.T) {
	if !strings.Contains(SystemPrompt, "default to parallel reads/searches when the steps do not depend on each other") {
		t.Error("SystemPrompt should prefer parallel reads/searches for independent steps")
	}
	if !strings.Contains(SystemPrompt, "prefer one read_file call with paths, or parallel read_file calls") {
		t.Error("SystemPrompt should prefer batched or parallel read_file usage")
	}
	if !strings.Contains(SystemPrompt, "prefer one search_code call with comma-separated patterns instead of serial searches") {
		t.Error("SystemPrompt should prefer one multi-pattern search_code call over serial searches")
	}
	if !strings.Contains(SystemPrompt, "read target code and its callers/tests in parallel when independent") {
		t.Error("SystemPrompt should encourage parallel investigation for shared changes")
	}
}

func TestSystemPrompt_ToolSelectionExamplesExist(t *testing.T) {
	if !strings.Contains(SystemPrompt, "### 4.5. Tool Selection Examples") {
		t.Error("SystemPrompt should include tool selection examples")
	}
	if !strings.Contains(SystemPrompt, "inspect_symbol(symbol=\"chatCore\", path=\"internal/agent/agent_chat.go\")") {
		t.Error("SystemPrompt should include an inspect_symbol example for an exact Go symbol")
	}
	if !strings.Contains(SystemPrompt, "use list_dir first") {
		t.Error("SystemPrompt should include a list_dir-first exploration example")
	}
	if !strings.Contains(SystemPrompt, "one read_file call with paths or parallel reads") {
		t.Error("SystemPrompt should include a batched/parallel read_file example")
	}
}

func TestSystemPrompt_NoPhantomReadFiles(t *testing.T) {
	// "read_files" が独立ツール名のように出てこないこと
	// 実在ツールは "read_file" であり、複数読みは paths パラメータで行う
	// "read_file" の直後に "s" が続く箇所を探す（"read_files" 単独出現）
	// ただし "read_file symbol" のような正当な表現は除外する
	if strings.Contains(SystemPrompt, "read_files") {
		t.Error("SystemPrompt should not mention 'read_files' as an independent tool — use 'read_file with paths' instead")
	}
}

func TestSystemPrompt_LocalVsSharedChangeGuidance(t *testing.T) {
	// Local vs shared changes の区別がプロンプトに含まれている
	if !strings.Contains(SystemPrompt, "Local vs shared changes") {
		t.Error("SystemPrompt should contain 'Local vs shared changes' guidance")
	}
	// shared changes では callers, references, tests の確認を要求
	if !strings.Contains(SystemPrompt, "shared changes") {
		t.Error("SystemPrompt should mention shared changes requiring broader investigation")
	}
}

func TestSystemPrompt_ImpactAnalysisLocalSharedConsistency(t *testing.T) {
	// Impact Analysis が local/shared の区分を含む
	if !strings.Contains(SystemPrompt, "**Shared changes**") {
		t.Error("Impact Analysis should have **Shared changes** section")
	}
	if !strings.Contains(SystemPrompt, "**Local changes**") {
		t.Error("Impact Analysis should have **Local changes** section")
	}
	// Local changes では broad search 不要
	if !strings.Contains(SystemPrompt, "broad reference search is not required") {
		t.Error("Impact Analysis should explicitly state broad search is not required for local changes")
	}
}

func TestSystemPrompt_NoBroadFirstDefault(t *testing.T) {
	// "read broadly first" は削除されている
	if strings.Contains(SystemPrompt, "read broadly first") {
		t.Error("SystemPrompt should not contain 'read broadly first' — replaced by local/shared distinction")
	}
	// "Read enough surrounding context" は削除されている
	if strings.Contains(SystemPrompt, "Read enough surrounding context") {
		t.Error("SystemPrompt should not contain 'Read enough surrounding context' — replaced by local/shared distinction")
	}
}

func TestSystemPrompt_LocalChangeExamplesSafe(t *testing.T) {
	// "adding a field" は local change の例に含まれない（波及しやすいため）
	// local change の例文を検出して "adding a field" が含まれていないことを確認
	if strings.Contains(SystemPrompt, "local changes (bug fix in one function, adding a field") {
		t.Error("local change examples should not include 'adding a field' — it can ripple to constructors/tests")
	}
}

func TestSystemPrompt_NarrowFirstStrategy(t *testing.T) {
	if !strings.Contains(SystemPrompt, "narrow-first") {
		t.Error("SystemPrompt should contain narrow-first investigation strategy")
	}
	if !strings.Contains(SystemPrompt, "Read priority:") {
		t.Error("SystemPrompt should define read priority order")
	}
}

func TestSystemPrompt_HypothesisDrivenInvestigation(t *testing.T) {
	if !strings.Contains(SystemPrompt, "After 2-4 targeted reads/searches, form a working hypothesis") {
		t.Error("SystemPrompt should include hypothesis-driven investigation rule with bounded read count")
	}
	if !strings.Contains(SystemPrompt, "switch to implementation") {
		t.Error("SystemPrompt should instruct switching to implementation once hypothesis is clear")
	}
}

func TestSystemPrompt_StopExploringRule(t *testing.T) {
	if !strings.Contains(SystemPrompt, "do not search \"just in case\"") {
		t.Error("SystemPrompt should explicitly discourage speculative searching")
	}
	if !strings.Contains(SystemPrompt, "Do not read neighboring files speculatively") {
		t.Error("SystemPrompt should discourage speculative neighboring file reads")
	}
}

func TestSystemPrompt_TargetedVerification(t *testing.T) {
	if !strings.Contains(SystemPrompt, "targeted verification first") {
		t.Error("SystemPrompt should prefer targeted verification before full CI")
	}
	if !strings.Contains(SystemPrompt, "Do not rerun the same failing command without a code change") {
		t.Error("SystemPrompt should forbid rerunning the same failing command without changes")
	}
}

func TestSystemPrompt_PhaseBasedNarration(t *testing.T) {
	if !strings.Contains(SystemPrompt, "phase boundaries") {
		t.Error("SystemPrompt should describe narration at phase boundaries")
	}
	if strings.Contains(SystemPrompt, "briefly state what you are about to do") {
		t.Error("SystemPrompt should not contain per-tool narration rule")
	}
	if !strings.Contains(SystemPrompt, "At most one short progress update per phase") {
		t.Error("SystemPrompt should limit progress updates to at most one per phase")
	}
}

func TestSystemPrompt_FullFileReadUpgradeGuard(t *testing.T) {
	if !strings.Contains(SystemPrompt, "Do not upgrade from targeted read to full file read unless") {
		t.Error("SystemPrompt should guard against unnecessary full file read upgrades")
	}
}

func TestSystemPrompt_NoBashRecommendations(t *testing.T) {
	// "bash (grep)" や "Use bash" といった調査用 bash 推奨パターンが含まれていない
	// NOTE: "bash cat/head/tail/grep/find/sed/awk are FORBIDDEN" のような禁止文は OK
	forbidden := []string{
		"bash (grep)",           // str_replace recovery で使われていた旧パターン
		"NOT bash cat",          // 旧 "read_file, NOT bash cat/head/tail/sed" パターン
		"NOT bash ls",           // 旧 "list_dir, NOT bash ls/find" パターン
		"Use bash (grep)",       // 旧 recovery guidance
		"Use bash (find",        // 旧 investigation guidance
		"bash (find/read-only)", // 旧 investigation allowed list
	}
	for _, f := range forbidden {
		if strings.Contains(SystemPrompt, f) {
			t.Errorf("SystemPrompt should not contain %q — use dedicated tool names instead", f)
		}
	}
}
