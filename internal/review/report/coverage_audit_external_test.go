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

func TestAuditReviewReportCoverageTracksSpecificPostPass1ExternalDocDelta(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.CheckedSurfaces = []ReviewSurfaceCoverage{
		{
			SurfaceID: "unrelated-external-doc",
			Summary:   "An unrelated external_doc was cited before Post-Pass1 search.",
			EvidenceRefs: []ReviewEvidenceRef{
				{Kind: ReviewEvidenceKindExternalDoc, DocID: "external-doc-old"},
			},
		},
	}
	report.ResidualRisks = []ReviewResidualRisk{
		{ID: "old-risk", Summary: "An unrelated web search evidence gap was already unverified."},
	}

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
			Level:                       "weak",
			DocCount:                    2,
			CitationCapableDocCount:     1,
			CitationCapableSnippetCount: 1,
		},
	})

	assertCoverageIssueForTest(t, issues, CoverageIssueKindUnreflectedExternalEvidence, "surface-1", "risk-1")
	assertCoverageIssueSeverityForTest(t, issues, CoverageIssueKindUnreflectedExternalEvidence, "surface-1", "risk-1", CoverageIssueSeverityMedium)
}

func TestAuditReviewReportCoverageReportsAddedExternalDocsWhenOnlyQueryTermsMentioned(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "OAuth redirect URI validation was checked."
	report.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "OAuth redirect URI validation risk was dismissed."

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			AddedQueryCount: 1,
			AddedQueries:    []string{"OAuth 2.0 redirect URI specification"},
			AddedDocIDs:     []string{"external-doc-post"},
			AddedDocURLs:    []string{"https://docs.example.test/oauth"},
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                       "weak",
			DocCount:                    1,
			CitationCapableDocCount:     1,
			CitationCapableSnippetCount: 1,
			OfficialConfirmation:        false,
		},
	})

	assertCoverageIssueForTest(t, issues, CoverageIssueKindUnreflectedExternalEvidence, "surface-1", "risk-1")
	assertCoverageIssueSeverityForTest(t, issues, CoverageIssueKindUnreflectedExternalEvidence, "surface-1", "risk-1", CoverageIssueSeverityMedium)
}

func TestAuditReviewReportCoverageDoesNotRequireExternalDocCitationWhenRepositoryEvidenceCoversScope(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	ref := ReviewEvidenceRef{Kind: ReviewEvidenceKindFile, Path: "internal/review/report/coverage_audit_external.go", Line: 1}
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{ref}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []ReviewEvidenceRef{ref}

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			AddedQueryCount: 1,
			AddedQueries:    []string{"OAuth 2.0 redirect URI specification"},
			AddedDocIDs:     []string{"external-doc-post"},
			AddedDocURLs:    []string{"https://docs.example.test/oauth"},
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                       "weak",
			DocCount:                    1,
			CitationCapableDocCount:     1,
			CitationCapableSnippetCount: 1,
			OfficialConfirmation:        false,
		},
	})

	assertNoCoverageIssueKindForTest(t, issues, CoverageIssueKindUnreflectedExternalEvidence)
}

func TestAuditReviewReportCoverageReportsAdequateOfficialSupportAsHighWhenUnreflected(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 checked from repository context."
	report.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed from repository context."

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			AddedQueryCount: 1,
			AddedQueries:    []string{"OAuth 2.0 redirect URI specification"},
			AddedDocIDs:     []string{"external-doc-official"},
			AddedDocURLs:    []string{"https://docs.example.test/oauth"},
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                                "adequate",
			DocCount:                             2,
			CitationCapableDocCount:              2,
			CitationCapableSnippetCount:          2,
			OfficialCandidateDocCount:            2,
			OfficialCandidateCitationCapableDocs: 2,
			OfficialConfirmation:                 true,
		},
	})

	assertCoverageIssueSeverityForTest(t, issues, CoverageIssueKindUnreflectedExternalEvidence, "surface-1", "risk-1", CoverageIssueSeverityHigh)
	merged := MergeCoverageIssuesIntoSaturationCheck(newSaturatedReviewSaturationCheckForTest(), issues)
	if merged.Status != ReviewSaturationStatusNeedsRevision {
		t.Fatalf("merged Status = %q, want %q", merged.Status, ReviewSaturationStatusNeedsRevision)
	}
}

func TestAuditReviewReportCoverageAcceptsAddedExternalDocEvidenceRef(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{
		{Kind: ReviewEvidenceKindExternalDoc, DocID: "external-doc-post"},
	}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []ReviewEvidenceRef{
		{Kind: ReviewEvidenceKindExternalDoc, DocID: "external-doc-post"},
	}

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			AddedQueryCount: 1,
			AddedQueries:    []string{"OAuth 2.0 redirect URI specification"},
			AddedDocIDs:     []string{"external-doc-post"},
			AddedDocURLs:    []string{"https://docs.example.test/oauth"},
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                       "weak",
			DocCount:                    1,
			CitationCapableDocCount:     1,
			CitationCapableSnippetCount: 1,
			OfficialConfirmation:        false,
		},
	})

	if len(issues) != 0 {
		t.Fatalf("AuditReviewReportCoverage() issues = %#v, want none", issues)
	}
}

func TestAuditReviewReportCoverageReportsUnreflectedExternalEvidencePerCleanScopeItem(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 checked after reviewing post-pass1 external evidence external-doc-post."
	report.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed before post-pass1 docs were considered."

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			AddedQueryCount: 1,
			AddedQueries:    []string{"OAuth 2.0 redirect URI specification"},
			AddedDocIDs:     []string{"external-doc-post"},
			AddedDocURLs:    []string{"https://docs.example.test/oauth"},
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                       "weak",
			DocCount:                    1,
			CitationCapableDocCount:     1,
			CitationCapableSnippetCount: 1,
			OfficialConfirmation:        false,
		},
	})

	var issue *CoverageIssue
	for i := range issues {
		if issues[i].Kind == CoverageIssueKindUnreflectedExternalEvidence {
			issue = &issues[i]
			break
		}
	}
	if issue == nil {
		t.Fatalf("AuditReviewReportCoverage() issues = %#v, want unreflected external evidence issue", issues)
	}
	if stringSliceContains(issue.SurfaceIDs, "surface-1") {
		t.Fatalf("SurfaceIDs = %#v, want reflected surface omitted", issue.SurfaceIDs)
	}
	if !stringSliceContains(issue.RiskIDs, "risk-1") {
		t.Fatalf("RiskIDs = %#v, want risk-1", issue.RiskIDs)
	}
}

func TestAuditReviewReportCoverageReportsFailedPostPass1SearchWithoutDocs(t *testing.T) {
	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: newPlanAwareCleanReportForValidationTest(),
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			AddedQueryCount:       1,
			AddedFailedQueryCount: 1,
			AddedNoResultCount:    1,
			AddedQueries:          []string{"OAuth redirect URI specification"},
			AddedFailedQueries:    []string{"OAuth redirect URI specification"},
			AddedNoResultQueries:  []string{"OAuth redirect URI specification"},
			EvidenceError:         true,
			Inconclusive:          true,
			Warnings:              []string{"web_search_query_error", "no_external_docs"},
			Reasons:               []string{"level=none: no external docs were fetched"},
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                "none",
			OfficialConfirmation: false,
			Warnings:             []string{"web_search_query_error", "no_external_docs"},
			Reasons:              []string{"level=none: no external docs were fetched"},
		},
	})

	assertCoverageIssueForTest(t, issues, CoverageIssueKindUnreflectedExternalEvidence, "surface-1", "risk-1")
}

func TestAuditReviewReportCoverageReportsFailedPostPass1SearchWhenOnlyQueryTermsMentioned(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "OAuth redirect URI validation was checked."
	report.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "OAuth redirect URI validation risk was dismissed."

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			AddedQueryCount:       1,
			AddedFailedQueryCount: 1,
			AddedNoResultCount:    1,
			AddedQueries:          []string{"OAuth 2.0 redirect URI specification"},
			AddedFailedQueries:    []string{"OAuth 2.0 redirect URI specification"},
			AddedNoResultQueries:  []string{"OAuth 2.0 redirect URI specification"},
			EvidenceError:         true,
			Inconclusive:          true,
			Warnings:              []string{"web_search_query_error", "no_external_docs"},
			Reasons:               []string{"level=none: no external docs were fetched"},
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                "none",
			OfficialConfirmation: false,
			Warnings:             []string{"web_search_query_error", "no_external_docs"},
			Reasons:              []string{"level=none: no external docs were fetched"},
		},
	})

	assertCoverageIssueForTest(t, issues, CoverageIssueKindUnreflectedExternalEvidence, "surface-1", "risk-1")
}

func TestAuditReviewReportCoverageAcceptsFailedPostPass1SearchGapReflection(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 checked; post-pass1 external search for OAuth redirect URI specification returned no results, so external spec confirmation remains unverified."
	report.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed; no external docs were fetched for OAuth redirect URI specification and official confirmation is absent."

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			AddedQueryCount:       1,
			AddedFailedQueryCount: 1,
			AddedNoResultCount:    1,
			AddedQueries:          []string{"OAuth 2.0 redirect URI specification"},
			AddedFailedQueries:    []string{"OAuth 2.0 redirect URI specification"},
			AddedNoResultQueries:  []string{"OAuth 2.0 redirect URI specification"},
			EvidenceError:         true,
			Inconclusive:          true,
			Warnings:              []string{"web_search_query_error", "no_external_docs"},
			Reasons:               []string{"level=none: no external docs were fetched"},
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                "none",
			OfficialConfirmation: false,
			Warnings:             []string{"web_search_query_error", "no_external_docs"},
			Reasons:              []string{"level=none: no external docs were fetched"},
		},
	})

	if len(issues) != 0 {
		t.Fatalf("AuditReviewReportCoverage() issues = %#v, want none", issues)
	}
}

func TestAuditReviewReportCoverageAcceptsConfirmationGapReflection(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 checked; post-pass1 external search was reviewed and official confirmation cannot be established."
	report.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed; post-pass1 external evidence was reviewed and the external spec cannot be confirmed."

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			AddedQueryCount:       1,
			AddedFailedQueryCount: 1,
			AddedNoResultCount:    1,
			AddedQueries:          []string{"OAuth 2.0 redirect URI specification"},
			AddedFailedQueries:    []string{"OAuth 2.0 redirect URI specification"},
			AddedNoResultQueries:  []string{"OAuth 2.0 redirect URI specification"},
			EvidenceError:         true,
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                "none",
			OfficialConfirmation: false,
			Warnings:             []string{"web_search_query_error"},
		},
	})

	if len(issues) != 0 {
		t.Fatalf("AuditReviewReportCoverage() issues = %#v, want none", issues)
	}
}

func TestAuditReviewReportCoverageAcceptsTruncatedPostPass1SearchGapReflection(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 checked; post-pass1 external search was truncated, so external support remains inconclusive."
	report.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed; post-pass1 external evidence truncation keeps this as weak support."

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			Truncated: true,
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                "weak",
			OfficialConfirmation: false,
			Warnings:             []string{"web_search_evidence_truncated"},
			Reasons:              []string{"level=partial: truncation prevents adequate support"},
		},
	})

	if len(issues) != 0 {
		t.Fatalf("AuditReviewReportCoverage() issues = %#v, want none", issues)
	}
}

func TestAuditReviewReportCoverageAcceptsSpecificPostPass1DeltaReflection(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 checked after reviewing post-pass1 external evidence external-doc-post."
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
			Level:                       "weak",
			DocCount:                    1,
			CitationCapableDocCount:     1,
			CitationCapableSnippetCount: 1,
		},
	})

	if len(issues) != 0 {
		t.Fatalf("AuditReviewReportCoverage() issues = %#v, want none", issues)
	}
}
