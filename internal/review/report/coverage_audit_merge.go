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

	highIssues, mediumIssues := partitionCoverageIssuesBySeverity(issues)
	if len(highIssues) == 0 && check.Status != ReviewSaturationStatusNeedsRevision {
		return check
	}

	merged := check
	if len(highIssues) > 0 {
		merged.Status = ReviewSaturationStatusNeedsRevision
	}
	if len(highIssues) > 0 && (strings.TrimSpace(merged.CheckedSummary) == "" || check.Status == ReviewSaturationStatusSaturated) {
		merged.CheckedSummary = "Deterministic coverage audit found final report coverage gaps."
	}

	for _, issue := range highIssues {
		merged.MissingSurfaceIDs = appendMissingCoverageIDs(merged.MissingSurfaceIDs, issue.SurfaceIDs)
		merged.MissingRiskIDs = appendMissingCoverageIDs(merged.MissingRiskIDs, issue.RiskIDs)
		if candidate, ok := coverageIssueAdditionalFindingCandidate(issue); ok {
			merged.AdditionalFindingCandidates = appendCoverageFindingCandidate(merged.AdditionalFindingCandidates, candidate)
		}
	}

	merged.RevisionInstructions = mergeCoverageRevisionInstructions(merged.RevisionInstructions, highIssues, mediumIssues)
	return merged
}

func partitionCoverageIssuesBySeverity(issues []CoverageIssue) ([]CoverageIssue, []CoverageIssue) {
	var highIssues []CoverageIssue
	var mediumIssues []CoverageIssue
	seen := map[string]struct{}{}
	for _, issue := range issues {
		key := coverageIssueKey(issue)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		switch issue.Severity {
		case CoverageIssueSeverityHigh:
			highIssues = append(highIssues, issue)
		default:
			mediumIssues = append(mediumIssues, issue)
		}
	}
	return highIssues, mediumIssues
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

func mergeCoverageRevisionInstructions(existing string, highIssues, mediumIssues []CoverageIssue) string {
	var lines []string
	seen := map[string]struct{}{}
	if strings.TrimSpace(existing) != "" {
		lines = appendCoverageInstructionLine(lines, seen, strings.TrimSpace(existing))
	}
	if len(highIssues) > 0 {
		lines = appendCoverageInstructionLine(lines, seen, "Deterministic coverage audit requires revision:")
		for _, issue := range highIssues {
			lines = appendCoverageInstructionLine(lines, seen, coverageIssueRevisionInstructionLine(issue))
		}
	}
	if len(mediumIssues) > 0 {
		lines = appendCoverageInstructionLine(lines, seen, "Deterministic coverage audit advisory context:")
		for _, issue := range mediumIssues {
			lines = appendCoverageInstructionLine(lines, seen, coverageIssueRevisionInstructionLine(issue))
		}
	}
	return strings.Join(lines, "\n")
}

func appendCoverageInstructionLine(lines []string, seen map[string]struct{}, line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return lines
	}
	if _, ok := seen[line]; ok {
		return lines
	}
	seen[line] = struct{}{}
	return append(lines, line)
}

func coverageIssueRevisionInstructionLine(issue CoverageIssue) string {
	instruction := strings.TrimSpace(issue.RevisionInstruction)
	if instruction == "" {
		instruction = strings.TrimSpace(issue.Summary)
	}
	if instruction == "" {
		instruction = fmt.Sprintf("Revisit coverage issue %q.", issue.Kind)
	}
	return fmt.Sprintf("- %s (%s): %s", issue.Kind, issue.Severity, instruction)
}

func coverageIssueKey(issue CoverageIssue) string {
	var b strings.Builder
	b.WriteString(string(issue.Kind))
	b.WriteString("\x00")
	b.WriteString(string(issue.Severity))
	b.WriteString("\x00")
	b.WriteString(issue.ProbeID)
	b.WriteString("\x00")
	b.WriteString(strings.Join(issue.SurfaceIDs, "\x01"))
	b.WriteString("\x00")
	b.WriteString(strings.Join(issue.RiskIDs, "\x01"))
	b.WriteString("\x00")
	b.WriteString(strings.TrimSpace(issue.RevisionInstruction))
	b.WriteString("\x00")
	b.WriteString(strings.TrimSpace(issue.Summary))
	for _, ref := range issue.EvidenceRefs {
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

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
