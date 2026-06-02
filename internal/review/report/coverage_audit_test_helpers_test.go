package report

import "testing"

func assertCoverageIssueForTest(t *testing.T, issues []CoverageIssue, kind CoverageIssueKind, surfaceID, riskID string) {
	t.Helper()

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
		return
	}
	t.Fatalf("issues = %#v, want kind=%q surface=%q risk=%q", issues, kind, surfaceID, riskID)
}

func assertNoCoverageIssueKindForTest(t *testing.T, issues []CoverageIssue, kind CoverageIssueKind) {
	t.Helper()

	for _, issue := range issues {
		if issue.Kind == kind {
			t.Fatalf("issues = %#v, want no kind=%q", issues, kind)
		}
	}
}
