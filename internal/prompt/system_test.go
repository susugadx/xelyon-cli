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

func TestSystemPrompt_NoPhantomReadFiles(t *testing.T) {
	// "read_files" が独立ツール名のように出てこないこと
	// 実在ツールは "read_file" であり、複数読みは paths パラメータで行う
	// "read_file" の直後に "s" が続く箇所を探す（"read_files" 単独出現）
	// ただし "read_file symbol" のような正当な表現は除外する
	if strings.Contains(SystemPrompt, "read_files") {
		t.Error("SystemPrompt should not mention 'read_files' as an independent tool — use 'read_file with paths' instead")
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
