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

func TestStripPlanningReferences_PreservesNormalModeAskPolicy(t *testing.T) {
	result := StripPlanningReferences(SystemPrompt)

	for _, want := range []string{
		"Do not use ambiguity as a reason to stop when a reasonable reversible default exists.",
		"Ask only when a choice is consequential, irreversible, externally visible, costly, permission-sensitive, or impossible to infer responsibly",
		"do not ask for preferences that repo evidence can resolve",
	} {
		if !strings.Contains(result, want) {
			t.Fatalf("stripped SystemPrompt missing normal-mode ask policy %q", want)
		}
	}
	for _, forbidden := range []string{
		"If uncertain: proceed with a stated assumption when the choice is local and reversible",
		"Use tools proactively: verify before modifying",
		"Ask via ask_user_question only when",
	} {
		if strings.Contains(result, forbidden) {
			t.Fatalf("stripped SystemPrompt kept obsolete planning fallback %q", forbidden)
		}
	}
}

func TestStripPlanningReferences_Idempotent(t *testing.T) {
	first := StripPlanningReferences(SystemPrompt)
	second := StripPlanningReferences(first)
	if first != second {
		t.Error("StripPlanningReferences should be idempotent")
	}
}
