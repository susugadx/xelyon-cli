package report

import "testing"

func TestAuditReviewReportCoverageReportsUnsupportedOfficialConfirmationClaim(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "official documentation confirms this behavior."

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			AddedQueryCount: 1,
			AddedQueries:    []string{"OAuth redirect URI specification"},
			AddedDocIDs:     []string{"external-doc-post"},
			AddedDocURLs:    []string{"https://docs.example.test/oauth"},
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                "partial",
			DocCount:             1,
			OfficialConfirmation: false,
		},
	})

	assertCoverageIssueForTest(t, issues, CoverageIssueKindUnsupportedExternalConfirmation, "surface-1", "risk-1")
	assertCoverageIssueSeverityForTest(t, issues, CoverageIssueKindUnsupportedExternalConfirmation, "surface-1", "risk-1", CoverageIssueSeverityHigh)
}

func TestAuditReviewReportCoverageReportsUnsupportedOfficialConfirmationAssertion(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "official confirmation is available from the vendor docs."

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		ExternalSupport: CoverageExternalSupport{
			Level:                "weak",
			DocCount:             1,
			OfficialConfirmation: false,
		},
	})

	assertCoverageIssueForTest(t, issues, CoverageIssueKindUnsupportedExternalConfirmation, "surface-1", "risk-1")
	assertCoverageIssueSeverityForTest(t, issues, CoverageIssueKindUnsupportedExternalConfirmation, "surface-1", "risk-1", CoverageIssueSeverityHigh)
}

func TestAuditReviewReportCoverageReportsUnknownOrThirdPartyOfficialConfirmationClaim(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "confirmed official documentation covers surface-1."
	report.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed because confirmed external spec coverage applies."

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		ExternalSupport: CoverageExternalSupport{
			Level:                       "adequate",
			DocCount:                    2,
			CitationCapableDocCount:     2,
			CitationCapableSnippetCount: 2,
			OfficialCandidateDocCount:   0,
			OfficialConfirmation:        true,
			Reasons:                     []string{"source_credibility=unknown_or_third_party"},
		},
	})

	assertCoverageIssueSeverityForTest(t, issues, CoverageIssueKindUnsupportedExternalConfirmation, "surface-1", "risk-1", CoverageIssueSeverityHigh)
}

func TestAuditReviewReportCoverageTreatsAbsentOfficialConfirmationAsNegated(t *testing.T) {
	tests := []struct {
		name    string
		summary string
	}{
		{
			name:    "official confirmation absent",
			summary: "post-pass1 external evidence external-doc-post was reviewed; official confirmation is absent.",
		},
		{
			name:    "missing official confirmation",
			summary: "post-pass1 external evidence external-doc-post was reviewed; Missing official confirmation.",
		},
		{
			name:    "verify official confirmation before verified",
			summary: "post-pass1 external evidence external-doc-post was reviewed; Verify official confirmation before marking verified.",
		},
		{
			name:    "obtain confirmed external spec before verified",
			summary: "post-pass1 external evidence external-doc-post was reviewed; Obtain confirmed external spec coverage before marking verified.",
		},
		{
			name:    "not a confirmed external spec",
			summary: "post-pass1 external evidence external-doc-post was reviewed; this is not a confirmed external spec.",
		},
		{
			name:    "cannot establish official confirmation",
			summary: "post-pass1 external evidence external-doc-post was reviewed; cannot establish official confirmation.",
		},
		{
			name:    "needs official confirmation",
			summary: "post-pass1 external evidence external-doc-post was reviewed; risk remains unverified and needs official confirmation.",
		},
		{
			name:    "pending official confirmation",
			summary: "post-pass1 external evidence external-doc-post was reviewed; pending official confirmation before treating this as verified.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := newPlanAwareCleanReportForValidationTest()
			report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 checked: " + tt.summary
			report.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed: " + tt.summary

			issues := AuditReviewReportCoverage(CoverageAuditInput{
				Plan:   newNoProbePlanScopeForTest(),
				Report: report,
				PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
					AddedQueryCount: 1,
					AddedQueries:    []string{"OAuth redirect URI specification"},
					AddedDocIDs:     []string{"external-doc-post"},
					AddedDocURLs:    []string{"https://docs.example.test/oauth"},
				},
				ExternalSupport: CoverageExternalSupport{
					Level:                "partial",
					DocCount:             1,
					OfficialConfirmation: false,
				},
			})

			if len(issues) != 0 {
				t.Fatalf("AuditReviewReportCoverage() issues = %#v, want none", issues)
			}
		})
	}
}

func TestAuditReviewReportCoverageIgnoresPendingOfficialConfirmationInVerificationPlan(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.RootCauseGroups = []ReviewRootCauseGroup{
		{
			ID:               "rc-1",
			Title:            "pending confirmation",
			Summary:          "Residual behavior remains conservative.",
			VerificationPlan: []string{"Keep this unverified; needs official confirmation before treating it as verified."},
		},
	}

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		ExternalSupport: CoverageExternalSupport{
			Level:                "weak",
			DocCount:             1,
			OfficialConfirmation: false,
		},
	})

	assertNoCoverageIssueKindForTest(t, issues, CoverageIssueKindUnsupportedExternalConfirmation)
}

func TestAuditReviewReportCoverageIgnoresConfirmationGuidanceFields(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.RootCauseGroups = []ReviewRootCauseGroup{
		{
			ID:          "rc-1",
			Title:       "external spec remains unverified",
			Summary:     "Residual behavior remains conservative.",
			FixStrategy: "Obtain confirmed external spec coverage before marking this fix verified.",
			DoNotFixBy: []string{
				"Do not mark this verified until confirmed external spec coverage exists.",
			},
			VerificationPlan: []string{
				"Obtain confirmed external spec coverage before marking verified.",
			},
		},
	}

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		ExternalSupport: CoverageExternalSupport{
			Level:                "weak",
			DocCount:             1,
			OfficialConfirmation: false,
		},
	})

	assertNoCoverageIssueKindForTest(t, issues, CoverageIssueKindUnsupportedExternalConfirmation)
}

func TestAuditReviewReportCoverageReportsLaterUnnegatedConfirmationClaim(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "post-pass1 external evidence external-doc-post is not a confirmed external spec, but confirmed external spec coverage is claimed for surface-1."
	report.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed after reviewing post-pass1 external evidence external-doc-post."

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			AddedQueryCount: 1,
			AddedQueries:    []string{"OAuth redirect URI specification"},
			AddedDocIDs:     []string{"external-doc-post"},
			AddedDocURLs:    []string{"https://docs.example.test/oauth"},
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                "weak",
			DocCount:             1,
			OfficialConfirmation: false,
		},
	})

	assertCoverageIssueForTest(t, issues, CoverageIssueKindUnsupportedExternalConfirmation, "surface-1", "risk-1")
}

func TestAuditReviewReportCoverageReportsLaterUnnegatedConfirmationClaimWithoutComma(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "post-pass1 external evidence external-doc-post is not a confirmed external spec but confirmed external spec coverage is claimed for surface-1."
	report.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed after reviewing post-pass1 external evidence external-doc-post."

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			AddedQueryCount: 1,
			AddedQueries:    []string{"OAuth redirect URI specification"},
			AddedDocIDs:     []string{"external-doc-post"},
			AddedDocURLs:    []string{"https://docs.example.test/oauth"},
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                "weak",
			DocCount:             1,
			OfficialConfirmation: false,
		},
	})

	assertCoverageIssueForTest(t, issues, CoverageIssueKindUnsupportedExternalConfirmation, "surface-1", "risk-1")
}

func TestAuditReviewReportCoverageReportsUnsupportedConfirmationWithoutPostPassDelta(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.Summary = "official documentation confirms this behavior."

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		ExternalSupport: CoverageExternalSupport{
			Level:                "weak",
			DocCount:             1,
			OfficialConfirmation: false,
		},
	})

	assertCoverageIssueForTest(t, issues, CoverageIssueKindUnsupportedExternalConfirmation, "surface-1", "risk-1")
	assertNoCoverageIssueKindForTest(t, issues, CoverageIssueKindUnreflectedExternalEvidence)
	merged := MergeCoverageIssuesIntoSaturationCheck(newSaturatedReviewSaturationCheckForTest(), issues)
	if err := ValidateReviewSaturationCheck(merged, newNoProbePlanScopeForTest(), report); err != nil {
		t.Fatalf("ValidateReviewSaturationCheck() error = %v, want nil", err)
	}
}

func TestAuditReviewReportCoverageReportsUnsupportedConfirmationForNonCleanScope(t *testing.T) {
	report := newPlanAwareHasFindingsReportForValidationTest()
	report.Summary = "official documentation confirms this behavior."
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceFinding
	report.ScopeCoverage.ReviewedImpactSurfaces[0].FindingIDs = []string{"finding-1"}

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			AddedQueryCount: 1,
			AddedQueries:    []string{"OAuth redirect URI specification"},
			AddedDocIDs:     []string{"external-doc-post"},
			AddedDocURLs:    []string{"https://docs.example.test/oauth"},
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                "weak",
			DocCount:             1,
			OfficialConfirmation: false,
		},
	})

	assertCoverageIssueForTest(t, issues, CoverageIssueKindUnsupportedExternalConfirmation, "surface-1", "risk-1")
	assertNoCoverageIssueKindForTest(t, issues, CoverageIssueKindUnreflectedExternalEvidence)
	merged := MergeCoverageIssuesIntoSaturationCheck(newSaturatedReviewSaturationCheckForTest(), issues)
	if err := ValidateReviewSaturationCheck(merged, newNoProbePlanScopeForTest(), report); err != nil {
		t.Fatalf("ValidateReviewSaturationCheck() error = %v, want nil", err)
	}
}
