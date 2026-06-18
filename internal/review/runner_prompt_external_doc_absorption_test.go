package review

import (
	"context"
	"strings"
	"testing"
)

func TestReviewRunnerSaturationCompactsAbsorbedExternalDocSnippet(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	absorbedSnippet := strings.Repeat("EXTERNAL_DOC_ABSORBED_RAW_SNIPPET ", 180)
	absorbedDoc := newRunnerOfficialCandidateExternalDocForSaturationAuditTest(
		"external-doc-absorbed",
		"https://docs.example.test/provider-history",
		absorbedSnippet,
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	)
	supportDoc := newRunnerOfficialCandidateExternalDocForSaturationAuditTest(
		"external-doc-support",
		"https://reference.example.test/provider-history",
		strings.Repeat("supporting official reference snippet ", 40),
		"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	)
	evidence.bundle.WebSearchEvidence = ReviewWebSearchEvidence{
		Enabled:      true,
		ExternalDocs: []ReviewExternalDocEvidence{absorbedDoc, supportDoc},
	}
	report := newRunnerCleanReportForTest(nil)
	ref := newExternalDocEvidenceRefForSaturationCompactTest(absorbedDoc)
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{ref}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []ReviewEvidenceRef{ref}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerNoProbePlanForTest()))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, report))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner, err := NewReviewRunner(ReviewRunnerOptions{
		EvidenceBuilder:     evidence,
		ProbeRunner:         &runnerFakeProbeRunner{},
		Model:               model,
		PromptReductionMode: ReviewPromptReductionModeApply,
	})
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v, want nil", err)
	}

	if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	reportPrompt := model.requests[1].Prompt
	if !strings.Contains(reportPrompt, absorbedSnippet) {
		t.Fatalf("report prompt should keep citation-capable external_doc snippet before absorption:\n%s", reportPrompt)
	}
	saturationPrompt := model.requests[2].Prompt
	if strings.Contains(saturationPrompt, absorbedSnippet) {
		t.Fatalf("saturation prompt leaked absorbed external_doc snippet:\n%s", saturationPrompt)
	}
	for _, want := range []string{
		"compacted absorbed external_doc snippet",
		"external-doc-absorbed",
		"external-doc-absorbed-snippet-1",
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"scope_coverage.risk.risk-1",
		"scope_coverage.surface.surface-1",
		"external_support.official_confirmation=true",
		"raw_artifact_ref=web_search_evidence*.json",
	} {
		if !strings.Contains(saturationPrompt, want) {
			t.Fatalf("saturation prompt missing external_doc compact marker %q:\n%s", want, saturationPrompt)
		}
	}
	reductionReport := runner.PromptReductionReport()
	if reductionReport.ClassifierCounts["review_external_doc_absorbed"] != 1 || reductionReport.ReplacedCount != 1 {
		t.Fatalf("PromptReductionReport() = %#v, want one external_doc absorption replacement", reductionReport)
	}
}

func TestReviewPromptEvidenceMarkdownForAbsorbedReportPreservesDiscoveryCompactionWithoutAbsorption(t *testing.T) {
	rawDiscoverySnippet := strings.Repeat("RAW_DISCOVERY_SNIPPET_MUST_NOT_REEXPAND_WHEN_ABSORPTION_UNAVAILABLE ", 400)
	bundle := newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")
	bundle.WebSearchEvidence = reviewWebSearchDiscoveryCompactEvidenceForTest(rawDiscoverySnippet, []ReviewExternalDocEvidence{
		reviewExternalDocEvidenceForDiscoveryCompactTest("external-doc-official-1", ReviewExternalDocSourceCredibilityOfficialCandidate, false, "official source snippet 1"),
		reviewExternalDocEvidenceForDiscoveryCompactTest("external-doc-official-2", ReviewExternalDocSourceCredibilityOfficialCandidate, false, "official source snippet 2"),
	})
	rawMarkdown := RenderReviewEvidenceMarkdown(bundle)
	report := newRunnerCleanReportForTest(nil)
	runner := &ReviewRunner{promptReductionMode: ReviewPromptReductionModeApply}

	got := runner.reviewPromptEvidenceMarkdownForAbsorbedReport(ReviewModelPhaseSaturationCheck, bundle, rawMarkdown, report)

	if strings.Contains(got, rawDiscoverySnippet) {
		t.Fatalf("absorbed report prompt re-expanded raw discovery snippet:\n%s", got)
	}
	if !strings.Contains(got, "[compacted discovery-only web_search snippet") ||
		!strings.Contains(got, "raw_result_preserved=review_artifact") {
		t.Fatalf("absorbed report prompt missing discovery compact placeholder:\n%s", got)
	}
	if strings.Contains(got, "compacted absorbed external_doc snippet") {
		t.Fatalf("absorbed report prompt compacted external docs without absorbable refs:\n%s", got)
	}
	reductionReport := runner.PromptReductionReport()
	if reductionReport.ClassifierCounts["review_web_search_discovery"] != 1 ||
		reductionReport.ClassifierCounts["review_external_doc_absorbed"] != 0 ||
		reductionReport.ReplacedCount != 1 {
		t.Fatalf("PromptReductionReport() = %#v, want one discovery replacement only", reductionReport)
	}
}

func TestReviewExternalDocAbsorptionKeepsFindingEvidenceSnippet(t *testing.T) {
	rawSnippet := strings.Repeat("FINDING_EXTERNAL_DOC_RAW_SNIPPET_MUST_STAY ", 180)
	doc := newRunnerOfficialCandidateExternalDocForSaturationAuditTest(
		"external-doc-finding",
		"https://docs.example.test/finding",
		rawSnippet,
		"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	)
	supportDoc := newRunnerOfficialCandidateExternalDocForSaturationAuditTest(
		"external-doc-support",
		"https://reference.example.test/finding",
		strings.Repeat("supporting official reference snippet ", 40),
		"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	)
	bundle := newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")
	bundle.WebSearchEvidence = ReviewWebSearchEvidence{
		Enabled:      true,
		ExternalDocs: []ReviewExternalDocEvidence{doc, supportDoc},
	}
	report := newRunnerCleanReportForTest(nil)
	ref := newExternalDocEvidenceRefForSaturationCompactTest(doc)
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{ref}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []ReviewEvidenceRef{ref}
	report.RootCauseGroups = []ReviewRootCauseGroup{
		{
			ID:                 "group-1",
			Title:              "Finding group",
			Severity:           ReviewGroupSeverityHigh,
			VerificationStatus: ReviewVerificationVerified,
			Findings: []ReviewFinding{
				{
					ID:           "finding-1",
					Title:        "Finding cites external doc",
					EvidenceRefs: []ReviewEvidenceRef{ref},
				},
			},
		},
	}

	_, _, _, _, ok := compactReviewExternalDocAbsorbedEvidence(ReviewModelPhaseSaturationCheck, bundle, report)
	if ok {
		t.Fatal("compactReviewExternalDocAbsorbedEvidence() ok = true, want finding evidence snippet kept")
	}
}

func TestReviewRunnerRevisionCompactsAbsorbedExternalDocSnippet(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	absorbedSnippet := strings.Repeat("REVISION_EXTERNAL_DOC_ABSORBED_RAW_SNIPPET ", 180)
	absorbedDoc := newRunnerOfficialCandidateExternalDocForSaturationAuditTest(
		"external-doc-revision",
		"https://docs.example.test/revision",
		absorbedSnippet,
		"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
	)
	supportDoc := newRunnerOfficialCandidateExternalDocForSaturationAuditTest(
		"external-doc-revision-support",
		"https://reference.example.test/revision",
		strings.Repeat("supporting revision official reference snippet ", 40),
		"sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	)
	evidence.bundle.WebSearchEvidence = ReviewWebSearchEvidence{
		Enabled:      true,
		ExternalDocs: []ReviewExternalDocEvidence{absorbedDoc, supportDoc},
	}
	initialReport := newRunnerCleanReportForTest(nil)
	ref := newExternalDocEvidenceRefForSaturationCompactTest(absorbedDoc)
	initialReport.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{ref}
	initialReport.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []ReviewEvidenceRef{ref}
	revisedReport := initialReport
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerNoProbePlanForTest()))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, initialReport))},
			{content: string(mustMarshalReviewSaturationCheckForTest(t, needsRevisionMissingRiskCheckForRunnerTest()))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, revisedReport))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner, err := NewReviewRunner(ReviewRunnerOptions{
		EvidenceBuilder:     evidence,
		ProbeRunner:         &runnerFakeProbeRunner{},
		Model:               model,
		PromptReductionMode: ReviewPromptReductionModeApply,
	})
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v, want nil", err)
	}

	if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	reportPrompt := model.requests[1].Prompt
	if !strings.Contains(reportPrompt, absorbedSnippet) {
		t.Fatalf("report prompt should keep external_doc snippet before revision absorption:\n%s", reportPrompt)
	}
	revisionPrompt := model.requests[3].Prompt
	if strings.Contains(revisionPrompt, absorbedSnippet) {
		t.Fatalf("revision prompt leaked absorbed external_doc snippet:\n%s", revisionPrompt)
	}
	for _, want := range []string{
		"compacted absorbed external_doc snippet",
		"external-doc-revision",
		"scope_coverage.risk.risk-1",
		"scope_coverage.surface.surface-1",
	} {
		if !strings.Contains(revisionPrompt, want) {
			t.Fatalf("revision prompt missing external_doc compact marker %q:\n%s", want, revisionPrompt)
		}
	}
	item := findReviewPromptReductionItemForTest(runner, "external_doc:external-doc-revision:external-doc-revision-snippet-1", ReviewModelPhaseReportRevision)
	if item == nil ||
		item.Family != ReviewPromptReductionFamilyExternalDoc ||
		item.Status != ReviewPromptReductionItemAbsorbed ||
		item.RawArtifactRef != reviewWebSearchEvidenceRawArtifactRef ||
		len(item.AbsorbedBy) != 2 ||
		len(item.EvidenceRefs) != 1 ||
		item.EvidenceRefs[0].Kind != ReviewEvidenceKindExternalDoc ||
		item.EvidenceRefs[0].DocID != "external-doc-revision" ||
		item.EvidenceRefs[0].SnippetID != "external-doc-revision-snippet-1" ||
		item.EvidenceRefs[0].ContentHash == "" ||
		item.OriginalBytes <= item.ReplacementBytes {
		t.Fatalf("absorbed external_doc prompt reduction item = %#v, want external_doc state item with refs and savings", item)
	}
	reductionReport := runner.PromptReductionReport()
	if reductionReport.FamilyCounts[string(ReviewPromptReductionFamilyExternalDoc)] == 0 ||
		reductionReport.StatusCounts[string(ReviewPromptReductionItemAbsorbed)] == 0 {
		t.Fatalf("PromptReductionReport() = %#v, want external_doc absorbed family/status counts", reductionReport)
	}
}
