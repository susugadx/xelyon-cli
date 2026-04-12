package fragments

import (
	"strings"
	"testing"
)

func TestInvestigationSharedLinesAndFallbackLabels(t *testing.T) {
	if !strings.Contains(GatherContextFirstLine("Add exact follow-up."), "Add exact follow-up.") {
		t.Fatalf("GatherContextFirstLine() should append extra guidance")
	}
	if !strings.Contains(SharedChangeGatherContextLine("Combine patterns."), "Combine patterns.") {
		t.Fatalf("SharedChangeGatherContextLine() should append extra guidance")
	}
	if !strings.Contains(ProjectMapExactReadLine(false), `gather_context(query="agent.go:161-328")`) {
		t.Fatalf("ProjectMapExactReadLine(false) should keep gather_context exact-read guidance")
	}
	if !strings.Contains(ProjectMapExactReadLine(true), "read_file with range syntax") {
		t.Fatalf("ProjectMapExactReadLine(true) should expose read_file override guidance")
	}
	if !strings.Contains(ProjectMapKnownSymbolLine(false), "Do NOT re-search") {
		t.Fatalf("ProjectMapKnownSymbolLine(false) should avoid low-level tool names")
	}
	if !strings.Contains(ProjectMapKnownSymbolLine(true), "Do NOT call search_code") {
		t.Fatalf("ProjectMapKnownSymbolLine(true) should mention search_code override")
	}
	if !strings.Contains(ReviewInvestigationSentence(true), "search_code/read_file") {
		t.Fatalf("ReviewInvestigationSentence(true) should mention low-level overrides")
	}

	if !strings.Contains(LowLevelOverridesWhenExposedLine(), "low-level expert overrides") {
		t.Fatalf("LowLevelOverridesWhenExposedLine() missing override guidance")
	}
	if !strings.Contains(DedicatedToolUsageSentence(), "gather_context first") {
		t.Fatalf("DedicatedToolUsageSentence() missing gather_context-first guidance")
	}
	if !strings.Contains(NoBashSubstituteSentence(), "do not use bash as a substitute") {
		t.Fatalf("NoBashSubstituteSentence() missing bash restriction")
	}
	if !strings.Contains(DelegatedInvestigationWaitLine(true), "search_code/read_file") {
		t.Fatalf("DelegatedInvestigationWaitLine(true) missing low-level tool names")
	}
	if !strings.Contains(DelegatedInvestigationWaitLine(false), "gather_context") {
		t.Fatalf("DelegatedInvestigationWaitLine(false) missing gather_context guidance")
	}
	if !strings.Contains(InvestigationCoverageLine(true), "search_code") {
		t.Fatalf("InvestigationCoverageLine(true) missing low-level guidance")
	}
	if !strings.Contains(InvestigationCoverageLine(false), "gather_context") {
		t.Fatalf("InvestigationCoverageLine(false) missing gather_context guidance")
	}
	if !strings.Contains(CombinedInvestigationQueryLine(true), "search_code") {
		t.Fatalf("CombinedInvestigationQueryLine(true) missing search_code guidance")
	}
	if !strings.Contains(CombinedInvestigationQueryLine(false), "gather_context") {
		t.Fatalf("CombinedInvestigationQueryLine(false) missing gather_context guidance")
	}
}

func TestInvestigationOverrideLabelFallbacks(t *testing.T) {
	searchLine := SearchCodeOverrideLine("  ", "")
	if !strings.Contains(searchLine, "an expert override") {
		t.Fatalf("SearchCodeOverrideLine() should fall back to expert override label, got %q", searchLine)
	}

	batchLine := ReadFileBatchOverrideLine(" ")
	if !strings.Contains(batchLine, "an expert override") {
		t.Fatalf("ReadFileBatchOverrideLine() should fall back to expert override label, got %q", batchLine)
	}
}
