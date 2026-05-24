package tui

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review"
)

func TestReviewReportTimelineMessage_SanitizesGroupAndFindingTitles(t *testing.T) {
	report := newTUITestReviewReport()
	report.RootCauseGroups[0].Title = "request\nstate"
	report.RootCauseGroups[0].Findings[0].Title = "stale\nresult"

	message := reviewReportTimelineMessage(report)
	if strings.Contains(message, "request\nstate") || strings.Contains(message, "stale\nresult") {
		t.Fatalf("timeline report contains unsanitized multiline title:\n%s", message)
	}

	if !strings.Contains(message, "Group: request state") {
		t.Fatalf("rendered report missing sanitized group title:\n%s", message)
	}
	if !strings.Contains(message, "Finding: stale result") {
		t.Fatalf("rendered report missing sanitized finding title:\n%s", message)
	}
}

func TestReviewReportTimelineMessage_RendersFindingSummaryAndEvidenceRefs(t *testing.T) {
	report := newTUITestReviewReport()
	report.RootCauseGroups[0].Summary = "request state lifecycle is split"
	report.RootCauseGroups[0].FixStrategy = "centralize request lifecycle"
	report.RootCauseGroups[0].Findings[0].Summary = "completed review result is discarded after close"
	report.RootCauseGroups[0].Findings[0].EvidenceRefs = []review.ReviewEvidenceRef{{
		Kind:         review.ReviewEvidenceKindFile,
		Summary:      "Esc closes the screen while the request keeps running",
		Path:         "internal/tui/review_screen_input.go",
		Line:         87,
		Snippet:      "return reviewCommandClose",
		ProbeID:      "probe-1",
		CommandIndex: review.ReviewCommandIndex(0),
	}}

	message := reviewReportTimelineMessage(report)
	for _, want := range []string{
		"Group summary: request state lifecycle is split",
		"Fix strategy: centralize request lifecycle",
		"Finding summary: completed review result is discarded after close",
		"Evidence: file - internal/tui/review_screen_input.go:87; probe probe-1 cmd 0; Esc closes the screen while the request keeps running; snippet: return reviewCommandClose",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("review report message missing %q:\n%s", want, message)
		}
	}
}

func TestReviewReportTimelineMessage_RendersComputedSummaryCounts(t *testing.T) {
	report := newTUITestReviewReport()
	report.ComputedSummary = &review.ReviewReportComputedSummary{
		RootCauseGroupCount:       2,
		FindingCount:              3,
		CheckedSurfaceCount:       4,
		FindingSurfaceCount:       5,
		UnverifiedSurfaceCount:    6,
		ResidualSurfaceCount:      7,
		CandidateRiskCount:        8,
		DismissedRiskCount:        9,
		FindingRiskCount:          10,
		UnverifiedRiskCount:       11,
		ResidualRiskCount:         12,
		NewReportPassFindingCount: 13,
		ProbeCount:                14,
		PassedProbeCount:          15,
		FailedProbeCount:          16,
		TimedOutProbeCount:        17,
		BlockedProbeCount:         18,
		MutatedWorktreeProbeCount: 19,
	}

	message := reviewReportTimelineMessage(report)
	for _, want := range []string{
		"Root cause groups: 2  Findings: 3",
		"Surfaces: checked 4  finding 5  unverified 6  residual 7",
		"Candidate risks: total 8  dismissed 9  finding 10  unverified 11  residual 12",
		"New report-pass findings: 13",
		"Probes: total 14  passed 15  failed 16  timed_out 17  blocked 18  mutated 19",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("review report message missing %q:\n%s", want, message)
		}
	}
}
