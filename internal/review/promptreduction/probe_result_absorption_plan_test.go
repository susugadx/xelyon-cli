package promptreduction

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestProbeResultAbsorptionPlanKeepsFindingEvidenceProbe(t *testing.T) {
	report := newProbeAbsorptionCleanReportForTest([]reviewreport.ReviewEvidenceRef{
		{Kind: reviewreport.ReviewEvidenceKindProbe, ProbeID: "probe-1"},
	})
	report.RootCauseGroups = []reviewreport.ReviewRootCauseGroup{
		{
			ID:                 "group-1",
			Title:              "Finding group",
			Severity:           reviewreport.ReviewGroupSeverityHigh,
			VerificationStatus: reviewreport.ReviewVerificationVerified,
			Findings: []reviewreport.ReviewFinding{
				{
					ID:           "finding-1",
					Title:        "Finding uses probe evidence",
					EvidenceRefs: []reviewreport.ReviewEvidenceRef{{Kind: reviewreport.ReviewEvidenceKindProbe, ProbeID: "probe-1"}},
				},
			},
		},
	}
	result := reviewprobe.ReviewProbeResult{
		ID:     "probe-1",
		Mode:   domain.ReviewProbeHostReadOnly,
		Status: domain.ReviewProbePassed,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{Command: "customtool", Status: domain.ReviewProbePassed, Output: strings.Repeat("finding probe output ", 300)},
		},
	}

	plan := BuildProbeResultAbsorptionPlan(report, []reviewprobe.ReviewProbeResult{result})
	if !plan.Empty() {
		t.Fatalf("BuildProbeResultAbsorptionPlan() = %#v, want finding evidence probe kept", plan)
	}
}

func TestProbeResultAbsorptionPlanKeepsFindingEvidenceCommandButAbsorbsSafeSibling(t *testing.T) {
	ref0 := reviewreport.ReviewEvidenceRef{
		Kind:         reviewreport.ReviewEvidenceKindProbeCommand,
		ProbeID:      "probe-1",
		CommandIndex: reviewreport.ReviewCommandIndex(0),
	}
	ref1 := reviewreport.ReviewEvidenceRef{
		Kind:         reviewreport.ReviewEvidenceKindProbeCommand,
		ProbeID:      "probe-1",
		CommandIndex: reviewreport.ReviewCommandIndex(1),
	}
	report := newProbeAbsorptionCleanReportForTest([]reviewreport.ReviewEvidenceRef{ref0, ref1})
	report.RootCauseGroups = []reviewreport.ReviewRootCauseGroup{
		{
			ID:                 "group-1",
			Title:              "Finding group",
			Severity:           reviewreport.ReviewGroupSeverityHigh,
			VerificationStatus: reviewreport.ReviewVerificationVerified,
			Findings: []reviewreport.ReviewFinding{
				{
					ID:           "finding-1",
					Title:        "Finding uses command[0] evidence",
					EvidenceRefs: []reviewreport.ReviewEvidenceRef{ref0},
				},
			},
		},
	}
	result := reviewprobe.ReviewProbeResult{
		ID:     "probe-1",
		Mode:   domain.ReviewProbeHostReadOnly,
		Status: domain.ReviewProbePassed,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{Command: "customtool", Args: []string{"--first"}, Status: domain.ReviewProbePassed, Output: strings.Repeat("finding command output ", 120)},
			{Command: "customtool", Args: []string{"--second"}, Status: domain.ReviewProbePassed, Output: strings.Repeat("safe sibling command output ", 120)},
		},
	}

	plan := BuildProbeResultAbsorptionPlan(report, []reviewprobe.ReviewProbeResult{result})
	if got := plan.ProbeCount(); got != 0 {
		t.Fatalf("ProbeCount() = %d, want no full-probe absorption when one command is finding evidence", got)
	}
	if _, ok := plan.CommandCandidate(reviewmodelinput.ProbeCommandResultKey{ProbeID: "probe-1", CommandIndex: 0}); ok {
		t.Fatalf("command[0] candidate exists, want finding evidence command kept")
	}
	if _, ok := plan.CommandCandidate(reviewmodelinput.ProbeCommandResultKey{ProbeID: "probe-1", CommandIndex: 1}); !ok {
		t.Fatalf("command[1] candidate missing, want safe sibling absorbed")
	}
}

func TestProbeResultAbsorptionPlanBuildsReductionRecordPolicy(t *testing.T) {
	ref := reviewreport.ReviewEvidenceRef{
		Kind:         reviewreport.ReviewEvidenceKindProbeCommand,
		ProbeID:      "probe-1",
		CommandIndex: reviewreport.ReviewCommandIndex(0),
	}
	report := newProbeAbsorptionCleanReportForTest([]reviewreport.ReviewEvidenceRef{ref})
	result := reviewprobe.ReviewProbeResult{
		ID:     "probe-1",
		Mode:   domain.ReviewProbeHostReadOnly,
		Status: domain.ReviewProbePassed,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{Command: "customtool", Args: []string{"--inspect"}, Status: domain.ReviewProbePassed, Output: strings.Repeat("absorbed command output ", 150)},
		},
	}
	key := reviewmodelinput.ProbeCommandResultKey{ProbeID: "probe-1", CommandIndex: 0}

	plan := BuildProbeResultAbsorptionPlan(report, []reviewprobe.ReviewProbeResult{result})
	records := plan.ReductionRecords(ReviewModelPhaseSaturationCheck, true, ProbeResultAbsorptionArtifactRefs{
		ProbeCommands: map[reviewmodelinput.ProbeCommandResultKey]string{key: "rawout_command"},
	})
	if len(records) != 1 {
		t.Fatalf("ReductionRecords() len = %d, want 1", len(records))
	}
	record := records[0]
	if record.Classifier != "probe_command_result_absorption_candidate" || record.SavedBytes <= 0 || record.SavedTokens <= 0 {
		t.Fatalf("ReductionRecords()[0] = %#v, want command classifier with savings", record)
	}
	item := record.Item
	if item.ID != "probe_result:probe-1:command:0" ||
		item.Phase != ReviewModelPhaseSaturationCheck ||
		item.Status != ReviewPromptReductionItemAbsorbed ||
		item.RawArtifactRef != "rawout_command" ||
		item.Family != ReviewPromptReductionFamilyProbeResult ||
		len(item.AbsorbedBy) != 2 ||
		len(item.EvidenceRefs) != 1 ||
		item.EvidenceRefs[0].Kind != reviewreport.ReviewEvidenceKindProbeCommand ||
		item.EvidenceRefs[0].CommandIndex == nil ||
		*item.EvidenceRefs[0].CommandIndex != 0 ||
		item.OriginalBytes <= item.ReplacementBytes {
		t.Fatalf("ReductionRecords()[0].Item = %#v, want applied command reduction item", item)
	}
}

func newProbeAbsorptionCleanReportForTest(evidenceRefs []reviewreport.ReviewEvidenceRef) reviewreport.ReviewReport {
	return reviewreport.ReviewReport{
		SchemaVersion: reviewreport.ReviewReportSchemaVersionV2,
		ScopeCoverage: &reviewreport.ReviewReportScopeCoverage{
			ReviewedImpactSurfaces: []reviewreport.ReviewReportImpactSurfaceCoverage{
				{
					SurfaceID:    "surface-1",
					Status:       reviewreport.ReviewReportImpactSurfaceChecked,
					Summary:      "surface-1 was checked.",
					EvidenceRefs: append([]reviewreport.ReviewEvidenceRef(nil), evidenceRefs...),
				},
			},
			ReviewedCandidateRisks: []reviewreport.ReviewReportCandidateRiskCoverage{
				{
					RiskID:       "risk-1",
					Status:       reviewreport.ReviewReportCandidateRiskDismissed,
					Summary:      "risk-1 was dismissed.",
					EvidenceRefs: append([]reviewreport.ReviewEvidenceRef(nil), evidenceRefs...),
				},
			},
		},
	}
}
