package tui

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

type reviewReportLineBuilder struct {
	lines []string
}

func reviewReportLines(report review.ReviewReport) []string {
	builder := newReviewReportLineBuilder()
	builder.appendReport(report)
	return builder.lines
}

func newReviewReportLineBuilder() reviewReportLineBuilder {
	return reviewReportLineBuilder{}
}

func (b *reviewReportLineBuilder) appendReport(report review.ReviewReport) {
	summary := reviewReportComputedSummary(report)
	b.appendDim("")
	b.appendDim("Verdict: " + string(report.Verdict))
	b.appendDim("Verification: " + string(report.OverallVerificationStatus))
	b.appendDim(fmt.Sprintf("Root cause groups: %d  Findings: %d", summary.RootCauseGroupCount, summary.FindingCount))
	b.appendDim(fmt.Sprintf("Surfaces: checked %d  finding %d  unverified %d  residual %d", summary.CheckedSurfaceCount, summary.FindingSurfaceCount, summary.UnverifiedSurfaceCount, summary.ResidualSurfaceCount))
	b.appendDim(fmt.Sprintf("Candidate risks: total %d  dismissed %d  finding %d  unverified %d  residual %d", summary.CandidateRiskCount, summary.DismissedRiskCount, summary.FindingRiskCount, summary.UnverifiedRiskCount, summary.ResidualRiskCount))
	b.appendDim(fmt.Sprintf("New report-pass findings: %d", summary.NewReportPassFindingCount))
	b.appendDim(fmt.Sprintf("Probes: total %d  passed %d  failed %d  timed_out %d  blocked %d  mutated %d", summary.ProbeCount, summary.PassedProbeCount, summary.FailedProbeCount, summary.TimedOutProbeCount, summary.BlockedProbeCount, summary.MutatedWorktreeProbeCount))
	if strings.TrimSpace(report.Summary) != "" {
		b.appendDim("Summary: " + reviewReportSingleLine(report.Summary))
	}
	for _, group := range report.RootCauseGroups {
		b.appendGroup(group)
	}
	for _, risk := range report.ResidualRisks {
		b.appendDim("Residual risk: " + reviewReportSingleLine(risk.Summary))
	}
	for _, probe := range report.ProbeSummaries {
		b.appendDim(fmt.Sprintf("Probe: %s %s %s", probe.ProbeID, probe.Mode, probe.Status))
	}
	if !report.GeneratedAt.IsZero() {
		b.appendDim("Generated at: " + report.GeneratedAt.Format("2006-01-02T15:04:05Z07:00"))
	}
}

func (b *reviewReportLineBuilder) appendGroup(group review.ReviewRootCauseGroup) {
	b.appendDim(fmt.Sprintf("Group: %s [%s/%s]", reviewReportSingleLine(group.Title), group.Severity, group.VerificationStatus))
	if strings.TrimSpace(group.Summary) != "" {
		b.appendDim("Group summary: " + reviewReportSingleLine(group.Summary))
	}
	if strings.TrimSpace(group.FixStrategy) != "" {
		b.appendDim("Fix strategy: " + reviewReportSingleLine(group.FixStrategy))
	}
	for _, finding := range group.Findings {
		b.appendFinding(group, finding)
	}
}

func (b *reviewReportLineBuilder) appendFinding(group review.ReviewRootCauseGroup, finding review.ReviewFinding) {
	b.appendDim(fmt.Sprintf("Finding: %s [%s/%s]", reviewReportSingleLine(finding.Title), group.Severity, group.VerificationStatus))
	if strings.TrimSpace(finding.Summary) != "" {
		b.appendDim("Finding summary: " + reviewReportSingleLine(finding.Summary))
	}
	for _, ref := range finding.EvidenceRefs {
		b.appendDim(reviewEvidenceRefLine(ref))
	}
}

func (b *reviewReportLineBuilder) appendDim(text string) {
	b.lines = append(b.lines, theme.Config.BgNormal+theme.Config.FgDim+" "+text+theme.Config.Reset)
}

func reviewEvidenceRefLine(ref review.ReviewEvidenceRef) string {
	segments := []string{"Evidence: " + reviewReportSingleLine(ref.Kind)}
	details := make([]string, 0, 4)
	if location := reviewEvidenceRefLocation(ref); location != "" {
		details = append(details, location)
	}
	if probe := reviewEvidenceRefProbe(ref); probe != "" {
		details = append(details, probe)
	}
	if strings.TrimSpace(ref.Summary) != "" {
		details = append(details, reviewReportSingleLine(ref.Summary))
	}
	if strings.TrimSpace(ref.Snippet) != "" {
		details = append(details, "snippet: "+reviewReportSingleLine(ref.Snippet))
	}
	if len(details) > 0 {
		segments = append(segments, strings.Join(details, "; "))
	}
	return strings.Join(segments, " - ")
}

func reviewEvidenceRefLocation(ref review.ReviewEvidenceRef) string {
	if strings.TrimSpace(ref.Path) == "" {
		return ""
	}
	path := reviewReportSingleLine(ref.Path)
	if ref.Line > 0 {
		return fmt.Sprintf("%s:%d", path, ref.Line)
	}
	return path
}

func reviewEvidenceRefProbe(ref review.ReviewEvidenceRef) string {
	if strings.TrimSpace(ref.ProbeID) == "" {
		return ""
	}
	probe := "probe " + reviewReportSingleLine(ref.ProbeID)
	if ref.CommandIndex != nil {
		probe += fmt.Sprintf(" cmd %d", *ref.CommandIndex)
	}
	return probe
}

func reviewReportSingleLine(text string) string {
	return termtext.SanitizeSingleLineANSI(text)
}

func reviewReportComputedSummary(report review.ReviewReport) review.ReviewReportComputedSummary {
	if report.ComputedSummary != nil {
		return *report.ComputedSummary
	}
	return review.ComputeReviewReportComputedSummary(report, report.ProbeSummaries)
}
