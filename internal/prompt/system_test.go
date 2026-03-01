package prompt

import (
	"strings"
	"testing"
)

func TestStripPlanningReferences(t *testing.T) {
	result := StripPlanningReferences(SystemPrompt)

	// Planning Tools セクションが除去されていること
	if strings.Contains(result, "### Planning Tools") {
		t.Error("StripPlanningReferences should remove ### Planning Tools section")
	}

	// create_plan / update_plan への参照が除去されていること
	if strings.Contains(result, "create_plan") {
		t.Error("StripPlanningReferences should remove all create_plan references")
	}
	if strings.Contains(result, "update_plan") {
		t.Error("StripPlanningReferences should remove all update_plan references")
	}
	// Planning Tools セクション自体は除去され、ask_user_question も含め planning 参照が除去されていること
	if strings.Contains(result, "ask_user_question") {
		t.Error("StripPlanningReferences should remove all planning tool references")
	}

	// Workflow Rules 全体は残っていること
	if !strings.Contains(result, "## Workflow Rules") {
		t.Error("StripPlanningReferences should preserve ## Workflow Rules")
	}

	// ## Available Tools セクション自体は残っていること（非FC プロバイダー向け）
	if !strings.Contains(result, "## Available Tools") {
		t.Error("StripPlanningReferences should preserve ## Available Tools")
	}

	// File Operations 等の他ツールは残っていること
	if !strings.Contains(result, "### File Operations") {
		t.Error("StripPlanningReferences should preserve other tool sections")
	}
	if !strings.Contains(result, "read_file") {
		t.Error("StripPlanningReferences should preserve non-planning tool references")
	}
}

func TestStripPlanningReferences_Idempotent(t *testing.T) {
	first := StripPlanningReferences(SystemPrompt)
	second := StripPlanningReferences(first)
	if first != second {
		t.Error("StripPlanningReferences should be idempotent")
	}
}
