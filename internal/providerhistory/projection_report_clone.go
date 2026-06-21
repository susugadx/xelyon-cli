package providerhistory

import "github.com/susugadx/xelyon-cli/internal/rawoutputs"

// CloneProjectionReport は projection report を defensive copy する。
func CloneProjectionReport(report ProjectionReport) ProjectionReport {
	if len(report.KeptReasonCounts) > 0 {
		counts := make(map[string]int, len(report.KeptReasonCounts))
		for reason, count := range report.KeptReasonCounts {
			counts[reason] = count
		}
		report.KeptReasonCounts = counts
	}
	if len(report.ContentReplacementToolCounts) > 0 {
		counts := make(map[string]int, len(report.ContentReplacementToolCounts))
		for toolName, count := range report.ContentReplacementToolCounts {
			counts[toolName] = count
		}
		report.ContentReplacementToolCounts = counts
	}
	if len(report.SkillReplacementToolCounts) > 0 {
		counts := make(map[string]int, len(report.SkillReplacementToolCounts))
		for toolName, count := range report.SkillReplacementToolCounts {
			counts[toolName] = count
		}
		report.SkillReplacementToolCounts = counts
	}
	if len(report.FutureFamilyCandidateCounts) > 0 {
		counts := make(map[string]int, len(report.FutureFamilyCandidateCounts))
		for family, count := range report.FutureFamilyCandidateCounts {
			counts[family] = count
		}
		report.FutureFamilyCandidateCounts = counts
	}
	if len(report.FutureFamilyKeptReasonCounts) > 0 {
		counts := make(map[string]int, len(report.FutureFamilyKeptReasonCounts))
		for reason, count := range report.FutureFamilyKeptReasonCounts {
			counts[reason] = count
		}
		report.FutureFamilyKeptReasonCounts = counts
	}
	if len(report.Candidates) > 0 {
		report.Candidates = cloneReductionCandidates(report.Candidates)
	}
	if len(report.Kept) > 0 {
		report.Kept = cloneReductionCandidates(report.Kept)
	}
	if len(report.RawOutputRefs) > 0 {
		report.RawOutputRefs = append([]rawoutputs.RawOutputRef(nil), report.RawOutputRefs...)
	}
	if len(report.RawOutputContextRefs) > 0 {
		report.RawOutputContextRefs = append([]rawoutputs.RawOutputRef(nil), report.RawOutputContextRefs...)
	}
	if len(report.RawOutputContextMissingRefIDs) > 0 {
		report.RawOutputContextMissingRefIDs = append([]string(nil), report.RawOutputContextMissingRefIDs...)
	}
	report.CommandEditDryRun = cloneCommandEditDryRunReport(report.CommandEditDryRun)
	return report
}

func cloneReductionCandidates(candidates []ReductionCandidate) []ReductionCandidate {
	if len(candidates) == 0 {
		return nil
	}
	cloned := make([]ReductionCandidate, len(candidates))
	for i, candidate := range candidates {
		cloned[i] = cloneReductionCandidate(candidate)
	}
	return cloned
}

func cloneCommandEditDryRunReport(report CommandEditDryRunReport) CommandEditDryRunReport {
	if len(report.CandidateReasonCounts) > 0 {
		counts := make(map[string]int, len(report.CandidateReasonCounts))
		for reason, count := range report.CandidateReasonCounts {
			counts[reason] = count
		}
		report.CandidateReasonCounts = counts
	}
	if len(report.CommandReplacementClassifierCounts) > 0 {
		counts := make(map[string]int, len(report.CommandReplacementClassifierCounts))
		for classifier, count := range report.CommandReplacementClassifierCounts {
			counts[classifier] = count
		}
		report.CommandReplacementClassifierCounts = counts
	}
	if len(report.KeptReasonCounts) > 0 {
		counts := make(map[string]int, len(report.KeptReasonCounts))
		for reason, count := range report.KeptReasonCounts {
			counts[reason] = count
		}
		report.KeptReasonCounts = counts
	}
	if len(report.ArtifactBackedKeptReasonCounts) > 0 {
		counts := make(map[string]int, len(report.ArtifactBackedKeptReasonCounts))
		for reason, count := range report.ArtifactBackedKeptReasonCounts {
			counts[reason] = count
		}
		report.ArtifactBackedKeptReasonCounts = counts
	}
	if len(report.RawOutputRefs) > 0 {
		report.RawOutputRefs = cloneProviderHistoryRawOutputRefs(report.RawOutputRefs)
	}
	if len(report.Candidates) > 0 {
		report.Candidates = append([]CommandEditDryRunCandidate(nil), report.Candidates...)
	}
	if len(report.Kept) > 0 {
		report.Kept = append([]CommandEditDryRunCandidate(nil), report.Kept...)
	}
	return report
}
