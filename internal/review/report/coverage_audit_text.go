package report

import "strings"

func textReflectsProbeOutcome(text string, probe ReviewProbeSummary) bool {
	normalized := strings.ToLower(text)
	if normalized == "" {
		return false
	}
	statusWords := []string{
		strings.ReplaceAll(string(canonicalReviewProbeSummaryStatusForValidation(probe)), "_", " "),
		"failed",
		"blocked",
		"timed out",
		"timed-out",
		"mutated worktree",
		"changed the working tree",
	}
	hasStatusWord := false
	for _, word := range statusWords {
		word = strings.TrimSpace(word)
		if word != "" && strings.Contains(normalized, word) {
			hasStatusWord = true
			break
		}
	}
	if !hasStatusWord {
		return false
	}
	if probe.ProbeID != "" && strings.Contains(normalized, strings.ToLower(probe.ProbeID)) {
		return true
	}
	return strings.Contains(normalized, "probe")
}

func collectReviewReportCoverageTexts(report ReviewReport) []string {
	return collectReviewReportTexts(report, coverageReportTextCollectionOptions{
		includeActionGuidance: true,
	})
}

func collectReviewReportClaimTexts(report ReviewReport) []string {
	return collectReviewReportTexts(report, coverageReportTextCollectionOptions{})
}

type coverageReportTextCollectionOptions struct {
	includeActionGuidance bool
}

func collectReviewReportTexts(report ReviewReport, options coverageReportTextCollectionOptions) []string {
	texts := []string{report.Summary}
	for _, surface := range report.CheckedSurfaces {
		texts = append(texts, surface.Summary)
		texts = appendEvidenceRefSummaries(texts, surface.EvidenceRefs)
	}
	for _, surface := range report.UnverifiedSurfaces {
		texts = append(texts, surface.Summary)
		texts = appendEvidenceRefSummaries(texts, surface.EvidenceRefs)
	}
	for _, risk := range report.ResidualRisks {
		texts = append(texts, risk.ID, risk.Summary, risk.SuggestedMitigation)
		texts = appendEvidenceRefSummaries(texts, risk.EvidenceRefs)
	}
	if report.ScopeCoverage != nil {
		for _, surface := range report.ScopeCoverage.ReviewedImpactSurfaces {
			texts = append(texts, surface.SurfaceID, surface.Summary)
			texts = appendEvidenceRefSummaries(texts, surface.EvidenceRefs)
		}
		for _, risk := range report.ScopeCoverage.ReviewedCandidateRisks {
			texts = append(texts, risk.RiskID, risk.Summary)
			texts = appendEvidenceRefSummaries(texts, risk.EvidenceRefs)
		}
		for _, finding := range report.ScopeCoverage.NewFindingsFromReportPass {
			texts = append(texts, finding.Summary)
			texts = appendEvidenceRefSummaries(texts, finding.EvidenceRefs)
		}
	}
	for _, group := range report.RootCauseGroups {
		texts = append(texts, group.ID, group.Title, group.Summary)
		if options.includeActionGuidance {
			texts = append(texts, group.FixStrategy)
			texts = append(texts, group.DoNotFixBy...)
			texts = append(texts, group.VerificationPlan...)
		}
		for _, finding := range group.Findings {
			texts = append(texts, finding.ID, finding.Title, finding.Summary)
			texts = appendEvidenceRefSummaries(texts, finding.EvidenceRefs)
			for _, surface := range finding.CheckedSurfaces {
				texts = append(texts, surface.SurfaceID, surface.Summary)
				texts = appendEvidenceRefSummaries(texts, surface.EvidenceRefs)
			}
			for _, surface := range finding.UnverifiedSurfaces {
				texts = append(texts, surface.SurfaceID, surface.Summary)
				texts = appendEvidenceRefSummaries(texts, surface.EvidenceRefs)
			}
			for _, risk := range finding.ResidualRisks {
				texts = append(texts, risk.ID, risk.Summary, risk.SuggestedMitigation)
				texts = appendEvidenceRefSummaries(texts, risk.EvidenceRefs)
			}
		}
		for _, surface := range group.CheckedSurfaces {
			texts = append(texts, surface.SurfaceID, surface.Summary)
			texts = appendEvidenceRefSummaries(texts, surface.EvidenceRefs)
		}
		for _, surface := range group.UnverifiedSurfaces {
			texts = append(texts, surface.SurfaceID, surface.Summary)
			texts = appendEvidenceRefSummaries(texts, surface.EvidenceRefs)
		}
		for _, risk := range group.ResidualRisks {
			texts = append(texts, risk.ID, risk.Summary, risk.SuggestedMitigation)
			texts = appendEvidenceRefSummaries(texts, risk.EvidenceRefs)
		}
	}
	return texts
}

func appendEvidenceRefSummaries(texts []string, refs []ReviewEvidenceRef) []string {
	for _, ref := range refs {
		texts = append(texts, ref.Summary)
	}
	return texts
}

func cleanCoverageStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
