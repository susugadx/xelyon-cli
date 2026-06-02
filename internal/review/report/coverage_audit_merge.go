package report

import (
	"fmt"
	"strings"
)

// MergeCoverageIssuesIntoSaturationCheck は deterministic coverage issue を既存 saturation check DTO へ反映する。
func MergeCoverageIssuesIntoSaturationCheck(check ReviewSaturationCheck, issues []CoverageIssue) ReviewSaturationCheck {
	if check.Status == ReviewSaturationStatusBlocked || len(issues) == 0 {
		return check
	}

	merged := check
	merged.Status = ReviewSaturationStatusNeedsRevision
	if strings.TrimSpace(merged.CheckedSummary) == "" || check.Status == ReviewSaturationStatusSaturated {
		merged.CheckedSummary = "Deterministic coverage audit found final report coverage gaps."
	}

	for _, issue := range issues {
		merged.MissingSurfaceIDs = appendMissingCoverageIDs(merged.MissingSurfaceIDs, issue.SurfaceIDs)
		merged.MissingRiskIDs = appendMissingCoverageIDs(merged.MissingRiskIDs, issue.RiskIDs)
		if candidate, ok := coverageIssueAdditionalFindingCandidate(issue); ok {
			merged.AdditionalFindingCandidates = appendCoverageFindingCandidate(merged.AdditionalFindingCandidates, candidate)
		}
	}

	merged.RevisionInstructions = mergeCoverageRevisionInstructions(merged.RevisionInstructions, issues)
	return merged
}

func appendMissingCoverageIDs(existing []string, ids []string) []string {
	for _, id := range ids {
		if strings.TrimSpace(id) == "" || stringSliceContains(existing, id) {
			continue
		}
		existing = append(existing, id)
	}
	return existing
}

func coverageIssueAdditionalFindingCandidate(issue CoverageIssue) (ReviewSaturationAdditionalFindingCandidate, bool) {
	if issue.Kind != CoverageIssueKindUnreflectedProbeOutcome || !coverageIssueHasConcreteProbeEvidenceRefs(issue.EvidenceRefs) {
		return ReviewSaturationAdditionalFindingCandidate{}, false
	}
	return ReviewSaturationAdditionalFindingCandidate{
		Summary:      issue.Summary,
		EvidenceRefs: append([]ReviewEvidenceRef(nil), issue.EvidenceRefs...),
		Reason:       "A non-passing linked probe outcome was not reflected in the finalized report; revise the report before deciding whether this remains a finding candidate.",
	}, true
}

func coverageIssueHasConcreteProbeEvidenceRefs(refs []ReviewEvidenceRef) bool {
	if len(refs) == 0 {
		return false
	}
	for _, ref := range refs {
		switch ref.Kind {
		case ReviewEvidenceKindProbe:
			if ref.ProbeID == "" {
				return false
			}
		case ReviewEvidenceKindProbeCommand:
			if ref.ProbeID == "" || ref.CommandIndex == nil {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func appendCoverageFindingCandidate(existing []ReviewSaturationAdditionalFindingCandidate, candidate ReviewSaturationAdditionalFindingCandidate) []ReviewSaturationAdditionalFindingCandidate {
	key := coverageFindingCandidateKey(candidate)
	for _, current := range existing {
		if coverageFindingCandidateKey(current) == key {
			return existing
		}
	}
	return append(existing, candidate)
}

func coverageFindingCandidateKey(candidate ReviewSaturationAdditionalFindingCandidate) string {
	var b strings.Builder
	b.WriteString(candidate.Summary)
	b.WriteString("\x00")
	b.WriteString(candidate.Reason)
	for _, ref := range candidate.EvidenceRefs {
		b.WriteString("\x00")
		b.WriteString(ref.Kind)
		b.WriteString("\x00")
		b.WriteString(ref.ProbeID)
		b.WriteString("\x00")
		if ref.CommandIndex != nil {
			fmt.Fprintf(&b, "%d", *ref.CommandIndex)
		}
		b.WriteString("\x00")
		b.WriteString(ref.Path)
		b.WriteString("\x00")
		b.WriteString(ref.DocID)
		b.WriteString("\x00")
		b.WriteString(ref.SnippetID)
	}
	return b.String()
}

func mergeCoverageRevisionInstructions(existing string, issues []CoverageIssue) string {
	var lines []string
	if strings.TrimSpace(existing) != "" {
		lines = append(lines, strings.TrimSpace(existing))
	}
	lines = append(lines, "Deterministic coverage audit requires revision:")
	for _, issue := range issues {
		instruction := strings.TrimSpace(issue.RevisionInstruction)
		if instruction == "" {
			instruction = strings.TrimSpace(issue.Summary)
		}
		if instruction == "" {
			instruction = fmt.Sprintf("Revisit coverage issue %q.", issue.Kind)
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", issue.Kind, instruction))
	}
	return strings.Join(lines, "\n")
}

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
