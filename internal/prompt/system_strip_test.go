package prompt

import (
	"strings"
	"testing"
)

func TestStripPlanningReferences(t *testing.T) {
	result := StripPlanningReferences(SystemPrompt)

	if strings.Contains(result, "ask_user_question") {
		t.Error("StripPlanningReferences should remove plan-mode tool references")
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
