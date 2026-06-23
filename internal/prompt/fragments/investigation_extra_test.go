package fragments

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/investigation"
)

func TestInvestigationSharedLinesAndFallbackLabels(t *testing.T) {
	if !strings.Contains(GatherContextFirstLine("Add exact follow-up."), "Add exact follow-up.") {
		t.Fatalf("GatherContextFirstLine() should append extra guidance")
	}
	if !strings.Contains(SharedChangeGatherContextLine("Combine patterns."), "Combine patterns.") {
		t.Fatalf("SharedChangeGatherContextLine() should append extra guidance")
	}
	if !strings.Contains(ProjectMapExactReadLine(investigation.SurfaceDefault), `gather_context(query="agent.go:161-328")`) {
		t.Fatalf("ProjectMapExactReadLine(default) should keep gather_context exact-read guidance")
	}
	if !strings.Contains(ProjectMapExactReadLine(investigation.SurfaceLegacyOverrides), "read_file with range syntax") {
		t.Fatalf("ProjectMapExactReadLine(legacy) should expose read_file override guidance")
	}
	if !strings.Contains(ProjectMapKnownSymbolLine(investigation.SurfaceDefault), "Do NOT re-search") {
		t.Fatalf("ProjectMapKnownSymbolLine(default) should avoid low-level tool names")
	}
	if !strings.Contains(ProjectMapKnownSymbolLine(investigation.SurfaceLegacyOverrides), "Do NOT call search_code") {
		t.Fatalf("ProjectMapKnownSymbolLine(legacy) should mention search_code override")
	}
	if !strings.Contains(ReviewInvestigationSentence(investigation.SurfaceLegacyOverrides), "search_code/read_file") {
		t.Fatalf("ReviewInvestigationSentence(legacy) should mention low-level overrides")
	}

	if !strings.Contains(LowLevelOverridesWhenExposedLine(), "low-level expert overrides") {
		t.Fatalf("LowLevelOverridesWhenExposedLine() missing override guidance")
	}
	if !strings.Contains(DedicatedToolUsageSentence(), "dedicated investigation tools first") {
		t.Fatalf("DedicatedToolUsageSentence() missing dedicated-tool guidance")
	}
	if !strings.Contains(NoBashSubstituteSentence(), "do not use bash as a substitute") {
		t.Fatalf("NoBashSubstituteSentence() missing bash restriction")
	}
	if !strings.Contains(DelegatedInvestigationWaitLine(investigation.SurfaceLegacyOverrides), "search_code/read_file") {
		t.Fatalf("DelegatedInvestigationWaitLine(legacy) missing low-level tool names")
	}
	if !strings.Contains(DelegatedInvestigationWaitLine(investigation.SurfaceDefault), "gather_context") {
		t.Fatalf("DelegatedInvestigationWaitLine(default) missing gather_context guidance")
	}
	if !strings.Contains(InvestigationCoverageLine(investigation.SurfaceLegacyOverrides), "search_code") {
		t.Fatalf("InvestigationCoverageLine(legacy) missing low-level guidance")
	}
	if !strings.Contains(InvestigationCoverageLine(investigation.SurfaceDefault), "gather_context") {
		t.Fatalf("InvestigationCoverageLine(default) missing gather_context guidance")
	}
	if !strings.Contains(CombinedInvestigationQueryLine(investigation.SurfaceLegacyOverrides), "search_code") {
		t.Fatalf("CombinedInvestigationQueryLine(legacy) missing search_code guidance")
	}
	if !strings.Contains(CombinedInvestigationQueryLine(investigation.SurfaceDefault), "gather_context") {
		t.Fatalf("CombinedInvestigationQueryLine(default) missing gather_context guidance")
	}
}

func TestInvestigationOverrideLabelFallbacks(t *testing.T) {
	searchLine := SearchCodeOverrideLine("  ", "")
	if !strings.Contains(searchLine, "an expert override") {
		t.Fatalf("SearchCodeOverrideLine() should fall back to expert override label, got %q", searchLine)
	}

	batchLine := ReadFileBatchOverrideLine(investigation.SurfaceLegacyOverrides, " ")
	if !strings.Contains(batchLine, "an expert override") {
		t.Fatalf("ReadFileBatchOverrideLine() should fall back to expert override label, got %q", batchLine)
	}
}
