package reviewscreen

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

// TimelineMessage は review report を agent timeline 用の plain text に整形する。
func TimelineMessage(report review.ReviewReport) string {
	lines := PlainLines(report)
	return strings.Join(lines, "\n")
}

// PlainLines は review report を plain text 行に整形する。
func PlainLines(report review.ReviewReport) []string {
	summary := reviewReportComputedSummary(report)
	lines := []string{
		"Review result",
		"Verdict: " + string(report.Verdict),
		"Verification: " + string(report.OverallVerificationStatus),
		fmt.Sprintf("Root cause groups: %d  Findings: %d", summary.RootCauseGroupCount, summary.FindingCount),
		fmt.Sprintf("Surfaces: checked %d  finding %d  unverified %d  residual %d", summary.CheckedSurfaceCount, summary.FindingSurfaceCount, summary.UnverifiedSurfaceCount, summary.ResidualSurfaceCount),
		fmt.Sprintf("Candidate risks: total %d  dismissed %d  finding %d  unverified %d  residual %d", summary.CandidateRiskCount, summary.DismissedRiskCount, summary.FindingRiskCount, summary.UnverifiedRiskCount, summary.ResidualRiskCount),
		fmt.Sprintf("New report-pass findings: %d", summary.NewReportPassFindingCount),
		fmt.Sprintf("Probes: total %d  passed %d  failed %d  timed_out %d  blocked %d  mutated %d", summary.ProbeCount, summary.PassedProbeCount, summary.FailedProbeCount, summary.TimedOutProbeCount, summary.BlockedProbeCount, summary.MutatedWorktreeProbeCount),
	}
	if strings.TrimSpace(report.Summary) != "" {
		lines = append(lines, "Summary: "+reviewReportSingleLine(report.Summary))
	}
	for _, group := range report.RootCauseGroups {
		lines = append(lines, reviewPlainGroupLines(group)...)
	}
	for _, risk := range report.ResidualRisks {
		lines = append(lines, "Residual risk: "+reviewReportSingleLine(risk.Summary))
	}
	for _, probe := range report.ProbeSummaries {
		lines = append(lines, fmt.Sprintf("Probe: %s %s %s", probe.ProbeID, probe.Mode, probe.Status))
	}
	if !report.GeneratedAt.IsZero() {
		lines = append(lines, "Generated at: "+report.GeneratedAt.Format("2006-01-02T15:04:05Z07:00"))
	}
	return lines
}

func reviewPlainGroupLines(group review.ReviewRootCauseGroup) []string {
	lines := []string{
		fmt.Sprintf("Group: %s [%s/%s]", reviewReportSingleLine(group.Title), group.Severity, group.VerificationStatus),
	}
	if strings.TrimSpace(group.Summary) != "" {
		lines = append(lines, "Group summary: "+reviewReportSingleLine(group.Summary))
	}
	if strings.TrimSpace(group.FixStrategy) != "" {
		lines = append(lines, "Fix strategy: "+reviewReportSingleLine(group.FixStrategy))
	}
	for _, finding := range group.Findings {
		lines = append(lines, reviewPlainFindingLines(group, finding)...)
	}
	return lines
}

func reviewPlainFindingLines(group review.ReviewRootCauseGroup, finding review.ReviewFinding) []string {
	lines := []string{
		fmt.Sprintf("Finding: %s [%s/%s]", reviewReportSingleLine(finding.Title), group.Severity, group.VerificationStatus),
	}
	if strings.TrimSpace(finding.Summary) != "" {
		lines = append(lines, "Finding summary: "+reviewReportSingleLine(finding.Summary))
	}
	for _, ref := range finding.EvidenceRefs {
		lines = append(lines, reviewEvidenceRefLine(ref))
	}
	return lines
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
