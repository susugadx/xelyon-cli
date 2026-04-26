package goimpact

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestClassifyRisk_ExportedSymbolIsHigh(t *testing.T) {
	got := ClassifyRisk(navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{Name: "Run", Exported: true},
	})
	if got != RiskHigh {
		t.Fatalf("ClassifyRisk() = %q, want %q", got, RiskHigh)
	}
}

func TestPlanForRisk(t *testing.T) {
	plan := PlanForRisk(RiskMedium)
	if plan.RiskLevel != RiskMedium || plan.ImplementationLimit != 4 {
		t.Fatalf("PlanForRisk() = %#v", plan)
	}
}
