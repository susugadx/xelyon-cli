package review

import (
	"strings"
	"testing"
)

func TestValidateReviewReport(t *testing.T) {
	tests := []struct {
		name        string
		report      ReviewReport
		wantErr     bool
		errContains string
	}{
		{
			name: "clean with findings is invalid",
			report: ReviewReport{
				Verdict: ReviewVerdictClean,
				RootCauseGroups: []ReviewRootCauseGroup{
					newRootCauseGroupForValidationTest(ReviewVerificationVerified),
				},
			},
			wantErr:     true,
			errContains: "requires root_cause_groups to be empty",
		},
		{
			name: "has_findings without groups is invalid",
			report: ReviewReport{
				Verdict:                   ReviewVerdictHasFindings,
				OverallVerificationStatus: ReviewVerificationVerified,
				RootCauseGroups:           nil,
			},
			wantErr:     true,
			errContains: "requires at least one root_cause_group",
		},
		{
			name: "has_findings with unverified group is invalid",
			report: ReviewReport{
				Verdict:                   ReviewVerdictHasFindings,
				OverallVerificationStatus: ReviewVerificationVerified,
				RootCauseGroups: []ReviewRootCauseGroup{
					newRootCauseGroupForValidationTest(ReviewVerificationUnverified),
				},
			},
			wantErr:     true,
			errContains: "verification_status",
		},
		{
			name: "has_findings with overall verified is valid",
			report: ReviewReport{
				Verdict:                   ReviewVerdictHasFindings,
				OverallVerificationStatus: ReviewVerificationVerified,
				RootCauseGroups: []ReviewRootCauseGroup{
					newRootCauseGroupForValidationTest(ReviewVerificationVerified),
				},
			},
		},
		{
			name: "has_findings with overall partially_verified is valid",
			report: ReviewReport{
				Verdict:                   ReviewVerdictHasFindings,
				OverallVerificationStatus: ReviewVerificationPartiallyVerified,
				RootCauseGroups: []ReviewRootCauseGroup{
					newRootCauseGroupForValidationTest(ReviewVerificationPartiallyVerified),
				},
			},
		},
		{
			name: "has_findings with overall unverified is invalid",
			report: ReviewReport{
				Verdict:                   ReviewVerdictHasFindings,
				OverallVerificationStatus: ReviewVerificationUnverified,
				RootCauseGroups: []ReviewRootCauseGroup{
					newRootCauseGroupForValidationTest(ReviewVerificationVerified),
				},
			},
			wantErr:     true,
			errContains: "overall_verification_status",
		},
		{
			name: "has_findings with overall not_applicable is invalid",
			report: ReviewReport{
				Verdict:                   ReviewVerdictHasFindings,
				OverallVerificationStatus: ReviewVerificationNotApplicable,
				RootCauseGroups: []ReviewRootCauseGroup{
					newRootCauseGroupForValidationTest(ReviewVerificationVerified),
				},
			},
			wantErr:     true,
			errContains: "overall_verification_status",
		},
		{
			name: "has_findings with overall blocked_or_inconclusive is invalid",
			report: ReviewReport{
				Verdict:                   ReviewVerdictHasFindings,
				OverallVerificationStatus: ReviewVerificationBlockedOrInconclusive,
				RootCauseGroups: []ReviewRootCauseGroup{
					newRootCauseGroupForValidationTest(ReviewVerificationVerified),
				},
			},
			wantErr:     true,
			errContains: "overall_verification_status",
		},
		{
			name: "blocked with blocked probe summary is valid",
			report: ReviewReport{
				Verdict: ReviewVerdictBlocked,
				ProbeSummaries: []ReviewProbeSummary{
					{ProbeID: "probe-1", Status: ReviewProbeBlocked},
				},
			},
		},
		{
			name: "blocked with timed_out probe summary is valid",
			report: ReviewReport{
				Verdict: ReviewVerdictBlocked,
				ProbeSummaries: []ReviewProbeSummary{
					{ProbeID: "probe-1", Status: ReviewProbeTimedOut},
				},
			},
		},
		{
			name: "blocked with mutated_worktree probe summary is valid",
			report: ReviewReport{
				Verdict: ReviewVerdictBlocked,
				ProbeSummaries: []ReviewProbeSummary{
					{ProbeID: "probe-1", Status: ReviewProbeMutatedWorktree},
				},
			},
		},
		{
			name: "blocked with failed probe summary only is invalid",
			report: ReviewReport{
				Verdict: ReviewVerdictBlocked,
				ProbeSummaries: []ReviewProbeSummary{
					{ProbeID: "probe-1", Status: ReviewProbeFailed},
				},
			},
			wantErr:     true,
			errContains: "requires blocked reason",
		},
		{
			name: "blocked with empty summary and no signals is invalid",
			report: ReviewReport{
				Verdict: ReviewVerdictBlocked,
				Summary: "   ",
			},
			wantErr:     true,
			errContains: "requires blocked reason",
		},
		{
			name: "clean with checked surfaces and no groups is valid",
			report: ReviewReport{
				Verdict: ReviewVerdictClean,
				CheckedSurfaces: []ReviewSurfaceCoverage{
					{SurfaceID: "surface-1", Summary: "checked"},
				},
			},
		},
		{
			name: "suspected issue in residual risks is valid without findings",
			report: ReviewReport{
				Verdict: ReviewVerdictBlocked,
				ResidualRisks: []ReviewResidualRisk{
					{Summary: "未検証の境界条件が残る"},
				},
			},
		},
		{
			name: "unknown verdict is invalid",
			report: ReviewReport{
				Verdict: ReviewVerdict("unexpected"),
			},
			wantErr:     true,
			errContains: "must be one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReviewReport(tt.report)
			if tt.wantErr {
				if err == nil {
					t.Fatal("ValidateReviewReport() error = nil, want error")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("ValidateReviewReport() error = %q, want substring %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateReviewReport() error = %v, want nil", err)
			}
		})
	}
}

func newRootCauseGroupForValidationTest(status ReviewVerificationStatus) ReviewRootCauseGroup {
	return ReviewRootCauseGroup{
		ID:                 "rc-1",
		Title:              "test group",
		Severity:           ReviewGroupSeverityLow,
		VerificationStatus: status,
	}
}
