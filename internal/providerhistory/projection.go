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
		report := buildProviderHistoryReductionDetectionReport(original, projection, policy)
		applyProviderHistoryReduction(&report, original, projection, policy)
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
