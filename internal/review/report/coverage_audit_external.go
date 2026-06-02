package report

import "strings"

func auditExternalEvidenceCoverage(input CoverageAuditInput) []CoverageIssue {
	if len(input.Plan.ImpactSurfaces)+len(input.Plan.CandidateRisks) == 0 {
		return nil
	}

	var issues []CoverageIssue
	issues = append(issues, auditUnsupportedExternalConfirmation(input)...)
	if input.PostPass1ExternalEvidence.hasPressureSignal() {
		issues = append(issues, auditUnreflectedExternalEvidence(input)...)
	}
	return issues
}

func auditUnsupportedExternalConfirmation(input CoverageAuditInput) []CoverageIssue {
	if input.ExternalSupport.hasConfirmedOfficialSupport() || !reportClaimsConfirmedExternalSpec(input.Report) {
		return nil
	}

	surfaceIDs, riskIDs := planScopeCoverageIDs(input.Plan)
	if len(surfaceIDs)+len(riskIDs) == 0 {
		return nil
	}

	return []CoverageIssue{{
		Kind:                CoverageIssueKindUnsupportedExternalConfirmation,
		Severity:            CoverageIssueSeverityHigh,
		SurfaceIDs:          surfaceIDs,
		RiskIDs:             riskIDs,
		Summary:             "The report asserts official confirmation or confirmed external spec coverage without sufficient external support.",
		RevisionInstruction: "Remove or qualify official confirmation / confirmed external spec wording unless external_support.official_confirmation=true and cited snippets support the claim; otherwise mark the relevant scope as unverified or residual, or explain the weak support.",
	}}
}

func auditUnreflectedExternalEvidence(input CoverageAuditInput) []CoverageIssue {
	delta := input.PostPass1ExternalEvidence
	requirements := buildCoverageExternalReflectionRequirements(delta)
	if len(requirements) == 0 {
		return nil
	}

	unreflected := unreflectedExternalEvidenceScopeIDs(input.Report.ScopeCoverage, requirements, input.ExternalSupport)
	var issues []CoverageIssue
	if len(unreflected.highSurfaceIDs)+len(unreflected.highRiskIDs) > 0 {
		issues = append(issues, newUnreflectedExternalEvidenceIssue(
			CoverageIssueSeverityHigh,
			unreflected.highSurfaceIDs,
			unreflected.highRiskIDs,
			delta,
		))
	}
	if len(unreflected.mediumSurfaceIDs)+len(unreflected.mediumRiskIDs) > 0 {
		issues = append(issues, newUnreflectedExternalEvidenceIssue(
			CoverageIssueSeverityMedium,
			unreflected.mediumSurfaceIDs,
			unreflected.mediumRiskIDs,
			delta,
		))
	}
	return issues
}

func newUnreflectedExternalEvidenceIssue(severity CoverageIssueSeverity, surfaceIDs, riskIDs []string, delta CoverageExternalEvidenceDelta) CoverageIssue {
	return CoverageIssue{
		Kind:                CoverageIssueKindUnreflectedExternalEvidence,
		Severity:            severity,
		SurfaceIDs:          surfaceIDs,
		RiskIDs:             riskIDs,
		Summary:             unreflectedExternalEvidenceSummary(delta),
		RevisionInstruction: "Revisit the specific Post-Pass1 external evidence delta and reflect its added docs, failed/no-result queries, or inconclusive support as a finding, a no-finding rationale in scope_coverage, residual/unverified status, or blocked status. Weak external evidence is revision feedback only and must not be auto-promoted into a finding.",
	}
}

func (support CoverageExternalSupport) hasConfirmedOfficialSupport() bool {
	if !support.OfficialConfirmation {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(support.Level)) {
	case "adequate", "strong":
		return support.OfficialCandidateCitationCapableDocs > 0
	default:
		return false
	}
}

func (delta CoverageExternalEvidenceDelta) hasPressureSignal() bool {
	return delta.AddedQueryCount > 0 ||
		delta.AddedFailedQueryCount > 0 ||
		delta.AddedNoResultCount > 0 ||
		len(delta.AddedQueries) > 0 ||
		len(delta.AddedFailedQueries) > 0 ||
		len(delta.AddedNoResultQueries) > 0 ||
		len(delta.AddedDocIDs) > 0 ||
		len(delta.AddedDocURLs) > 0 ||
		delta.EvidenceError ||
		delta.Inconclusive ||
		delta.Truncated ||
		len(delta.Warnings) > 0 ||
		len(delta.Reasons) > 0
}

func unreflectedExternalEvidenceSummary(delta CoverageExternalEvidenceDelta) string {
	switch {
	case delta.AddedFailedQueryCount > 0 || delta.AddedNoResultCount > 0 || delta.EvidenceError || delta.Inconclusive:
		return "Post-Pass1 external evidence added failed, no-result, or inconclusive search context, but the finalized report does not reflect that gap in evidence, scope rationale, residual risk, or unverified coverage."
	case len(delta.AddedDocIDs) > 0 || len(delta.AddedDocURLs) > 0:
		return "Post-Pass1 external docs were added but the finalized report does not reflect the added doc IDs, URLs, scope rationale, residual risk, or unverified coverage."
	default:
		return "Post-Pass1 external evidence was added but the finalized report does not reflect it in evidence, scope rationale, residual risk, or unverified coverage."
	}
}

type coverageExternalReflectionRequirementKind string

const (
	coverageExternalReflectionRequirementAddedDocs coverageExternalReflectionRequirementKind = "added_external_docs"
	coverageExternalReflectionRequirementSearchGap coverageExternalReflectionRequirementKind = "external_search_gap"
	coverageExternalReflectionRequirementGeneric   coverageExternalReflectionRequirementKind = "post_pass1_external_evidence"
)

type coverageExternalReflectionRequirement struct {
	kind        coverageExternalReflectionRequirementKind
	docIDs      []string
	docURLs     []string
	diagnostics []string
}

func buildCoverageExternalReflectionRequirements(delta CoverageExternalEvidenceDelta) []coverageExternalReflectionRequirement {
	if !delta.hasPressureSignal() {
		return nil
	}

	docIDs := cleanCoverageStrings(delta.AddedDocIDs)
	docURLs := cleanCoverageStrings(delta.AddedDocURLs)
	diagnostics := cleanCoverageStrings(append(append([]string(nil), delta.Warnings...), delta.Reasons...))

	var requirements []coverageExternalReflectionRequirement
	if len(docIDs)+len(docURLs) > 0 {
		requirements = append(requirements, coverageExternalReflectionRequirement{
			kind:        coverageExternalReflectionRequirementAddedDocs,
			docIDs:      docIDs,
			docURLs:     docURLs,
			diagnostics: diagnostics,
		})
	}
	if delta.hasSearchGapSignal() || delta.hasDiagnosticGapSignal() {
		requirements = append(requirements, coverageExternalReflectionRequirement{
			kind:        coverageExternalReflectionRequirementSearchGap,
			docIDs:      docIDs,
			docURLs:     docURLs,
			diagnostics: diagnostics,
		})
	}
	if len(requirements) == 0 && delta.AddedQueryCount > 0 {
		requirements = append(requirements, coverageExternalReflectionRequirement{
			kind:        coverageExternalReflectionRequirementGeneric,
			diagnostics: diagnostics,
		})
	}
	return requirements
}

type coverageExternalEvidenceUnreflectedScopeIDs struct {
	highSurfaceIDs   []string
	highRiskIDs      []string
	mediumSurfaceIDs []string
	mediumRiskIDs    []string
}

func unreflectedExternalEvidenceScopeIDs(coverage *ReviewReportScopeCoverage, requirements []coverageExternalReflectionRequirement, support CoverageExternalSupport) coverageExternalEvidenceUnreflectedScopeIDs {
	if coverage == nil {
		return coverageExternalEvidenceUnreflectedScopeIDs{}
	}
	var result coverageExternalEvidenceUnreflectedScopeIDs
	for _, surface := range coverage.ReviewedImpactSurfaces {
		if surface.Status != ReviewReportImpactSurfaceChecked {
			continue
		}
		if !scopeCoverageItemReflectsExternalEvidence(surface.Summary, surface.EvidenceRefs, requirements) {
			if scopeCoverageRepositoryEvidenceRefsCoverExternalDocNonCitation(surface.EvidenceRefs, requirements) {
				continue
			}
			if support.hasConfirmedOfficialSupport() && !scopeCoverageHasRepositoryEvidenceRef(surface.EvidenceRefs) {
				result.highSurfaceIDs = append(result.highSurfaceIDs, surface.SurfaceID)
				continue
			}
			result.mediumSurfaceIDs = append(result.mediumSurfaceIDs, surface.SurfaceID)
		}
	}
	for _, risk := range coverage.ReviewedCandidateRisks {
		if risk.Status != ReviewReportCandidateRiskDismissed {
			continue
		}
		if !scopeCoverageItemReflectsExternalEvidence(risk.Summary, risk.EvidenceRefs, requirements) {
			if scopeCoverageRepositoryEvidenceRefsCoverExternalDocNonCitation(risk.EvidenceRefs, requirements) {
				continue
			}
			if support.hasConfirmedOfficialSupport() && !scopeCoverageHasRepositoryEvidenceRef(risk.EvidenceRefs) {
				result.highRiskIDs = append(result.highRiskIDs, risk.RiskID)
				continue
			}
			result.mediumRiskIDs = append(result.mediumRiskIDs, risk.RiskID)
		}
	}
	return result
}

func scopeCoverageItemReflectsExternalEvidence(summary string, refs []ReviewEvidenceRef, requirements []coverageExternalReflectionRequirement) bool {
	for _, requirement := range requirements {
		if !scopeCoverageItemReflectsExternalEvidenceRequirement(summary, refs, requirement) {
			return false
		}
	}
	return true
}

func scopeCoverageItemReflectsExternalEvidenceRequirement(summary string, refs []ReviewEvidenceRef, requirement coverageExternalReflectionRequirement) bool {
	if requirement.kind == coverageExternalReflectionRequirementAddedDocs && evidenceRefsReflectExternalDocRequirement(refs, requirement) {
		return true
	}
	for _, text := range appendEvidenceRefSummaries([]string{summary}, refs) {
		if textReflectsExternalEvidenceRequirement(text, requirement) {
			return true
		}
	}
	return false
}

func evidenceRefsReflectExternalDocRequirement(refs []ReviewEvidenceRef, requirement coverageExternalReflectionRequirement) bool {
	for _, ref := range refs {
		if ref.Kind != ReviewEvidenceKindExternalDoc {
			continue
		}
		if stringSliceContains(requirement.docIDs, strings.TrimSpace(ref.DocID)) {
			return true
		}
	}
	return false
}

func scopeCoverageRepositoryEvidenceRefsCoverExternalDocNonCitation(refs []ReviewEvidenceRef, requirements []coverageExternalReflectionRequirement) bool {
	if !coverageExternalRequirementsOnlyAddedDocs(requirements) {
		return false
	}
	return scopeCoverageHasRepositoryEvidenceRef(refs)
}

func coverageExternalRequirementsOnlyAddedDocs(requirements []coverageExternalReflectionRequirement) bool {
	if len(requirements) == 0 {
		return false
	}
	for _, requirement := range requirements {
		if requirement.kind != coverageExternalReflectionRequirementAddedDocs {
			return false
		}
	}
	return true
}

func scopeCoverageHasRepositoryEvidenceRef(refs []ReviewEvidenceRef) bool {
	for _, ref := range refs {
		switch ref.Kind {
		case ReviewEvidenceKindFile, ReviewEvidenceKindDiff, ReviewEvidenceKindGitStatus, ReviewEvidenceKindRuleFile, ReviewEvidenceKindProbe, ReviewEvidenceKindProbeCommand:
			return true
		}
	}
	return false
}

func reportClaimsConfirmedExternalSpec(report ReviewReport) bool {
	for _, text := range collectReviewReportClaimTexts(report) {
		if textClaimsConfirmedExternalSpec(text) {
			return true
		}
	}
	return false
}

func (delta CoverageExternalEvidenceDelta) hasSearchGapSignal() bool {
	return delta.AddedFailedQueryCount > 0 ||
		delta.AddedNoResultCount > 0 ||
		delta.EvidenceError ||
		delta.Inconclusive ||
		delta.Truncated
}

func (delta CoverageExternalEvidenceDelta) hasDiagnosticGapSignal() bool {
	for _, value := range append(append([]string(nil), delta.Warnings...), delta.Reasons...) {
		normalized := strings.ToLower(value)
		if textReferencesExternalEvidenceGapState(normalized) || textReferencesExternalEvidenceGapState(strings.ReplaceAll(normalized, "_", " ")) {
			return true
		}
	}
	return false
}
