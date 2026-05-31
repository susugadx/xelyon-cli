package providerhistory

import "github.com/susugadx/xelyon-cli/internal/api"

// ProjectionInput は provider-facing history projection の入力を表す。
type ProjectionInput struct {
	Messages []api.Message
	Policy   Policy
}

// ProjectionResult は provider-facing history projection の出力を表す。
type ProjectionResult struct {
	History []api.Message
	Report  ProjectionReport
}

// Project は raw history から provider-facing projection と report を構築する。
func Project(input ProjectionInput) ProjectionResult {
	policy := normalizePolicy(input.Policy)
	original := cloneProjectionMessages(input.Messages)
	projection := cloneProjectionMessages(input.Messages)

	if policy.Mode == Apply && len(original) > 0 {
		report := buildProviderHistoryReductionDetectionReport(original, projection, policy.Mode)
		applyProviderHistoryReduction(&report, projection, policy)
		finalizeProjectionReport(&report, original, projection)
		return ProjectionResult{
			History: projection,
			Report:  report,
		}
	}

	return ProjectionResult{
		History: projection,
		Report:  buildProjectionReport(original, projection, policy),
	}
}

// CloneProjectionReport は projection report を defensive copy する。
func CloneProjectionReport(report ProjectionReport) ProjectionReport {
	if len(report.KeptReasonCounts) > 0 {
		counts := make(map[string]int, len(report.KeptReasonCounts))
		for reason, count := range report.KeptReasonCounts {
			counts[reason] = count
		}
		report.KeptReasonCounts = counts
	}
	if len(report.Candidates) > 0 {
		report.Candidates = cloneReductionCandidates(report.Candidates)
	}
	if len(report.Kept) > 0 {
		report.Kept = cloneReductionCandidates(report.Kept)
	}
	report.CommandEditDryRun = cloneCommandEditDryRunReport(report.CommandEditDryRun)
	return report
}

// ProjectionDisablesResponseIDChain は projection 適用により previous response id chain を止めるべきかを返す。
func ProjectionDisablesResponseIDChain(report ProjectionReport) bool {
	return report.ResponsesChainDisabled
}

func cloneProjectionMessages(messages []api.Message) []api.Message {
	if messages == nil {
		return nil
	}
	if len(messages) == 0 {
		return []api.Message{}
	}
	return api.CloneMessages(messages)
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
	if len(report.KeptReasonCounts) > 0 {
		counts := make(map[string]int, len(report.KeptReasonCounts))
		for reason, count := range report.KeptReasonCounts {
			counts[reason] = count
		}
		report.KeptReasonCounts = counts
	}
	if len(report.Candidates) > 0 {
		report.Candidates = append([]CommandEditDryRunCandidate(nil), report.Candidates...)
	}
	if len(report.Kept) > 0 {
		report.Kept = append([]CommandEditDryRunCandidate(nil), report.Kept...)
	}
	return report
}
