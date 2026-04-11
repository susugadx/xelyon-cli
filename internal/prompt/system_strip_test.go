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
