package report

import "testing"

func assertCoverageIssueForTest(t *testing.T, issues []CoverageIssue, kind CoverageIssueKind, surfaceID, riskID string) {
	t.Helper()

	if issue := findCoverageIssueForTest(issues, kind, surfaceID, riskID); issue != nil {
		return
	}
	t.Fatalf("issues = %#v, want kind=%q surface=%q risk=%q", issues, kind, surfaceID, riskID)
}

func assertCoverageIssueSeverityForTest(t *testing.T, issues []CoverageIssue, kind CoverageIssueKind, surfaceID, riskID string, severity CoverageIssueSeverity) {
	t.Helper()

	issue := findCoverageIssueForTest(issues, kind, surfaceID, riskID)
	if issue == nil {
		t.Fatalf("issues = %#v, want kind=%q surface=%q risk=%q severity=%q", issues, kind, surfaceID, riskID, severity)
	}
	if issue.Severity != severity {
		t.Fatalf("issue severity = %q, want %q: %#v", issue.Severity, severity, *issue)
	}
}

func findCoverageIssueForTest(issues []CoverageIssue, kind CoverageIssueKind, surfaceID, riskID string) *CoverageIssue {
	for _, issue := range issues {
		if issue.Kind != kind {
			continue
		}
		if surfaceID != "" && !stringSliceContains(issue.SurfaceIDs, surfaceID) {
			continue
		}
		if riskID != "" && !stringSliceContains(issue.RiskIDs, riskID) {
			continue
		}
		return &issue
	}
	return nil
}

func assertNoCoverageIssueKindForTest(t *testing.T, issues []CoverageIssue, kind CoverageIssueKind) {
	t.Helper()

	for _, issue := range issues {
		if issue.Kind == kind {
			t.Fatalf("issues = %#v, want no kind=%q", issues, kind)
		}
	}
}
