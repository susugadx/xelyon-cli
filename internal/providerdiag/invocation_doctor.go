package providerdiag

import "strings"

// InvocationSmokeRequest は invocation 系 provider doctor の smoke request descriptor を表す。
type InvocationSmokeRequest struct {
	Name            string
	ToolPayload     bool
	ImagePayload    bool
	ThinkingEnabled bool
}

// InvocationSmokeRequestResult は request_id を返す invocation 系 provider doctor の request 単位結果を表す。
type InvocationSmokeRequestResult struct {
	Name            string     `json:"name"`
	Ran             bool       `json:"ran"`
	Skipped         bool       `json:"skipped,omitempty"`
	SkipReason      string     `json:"skip_reason,omitempty"`
	ToolPayload     bool       `json:"tool_payload,omitempty"`
	ImagePayload    bool       `json:"image_payload,omitempty"`
	ThinkingEnabled bool       `json:"thinking_enabled,omitempty"`
	Content         string     `json:"content,omitempty"`
	RequestID       string     `json:"request_id"`
	Duration        string     `json:"duration,omitempty"`
	UsageObserved   bool       `json:"usage_observed"`
	Usage           SmokeUsage `json:"usage"`
	Cost            SmokeCost  `json:"cost"`
	Error           string     `json:"error,omitempty"`
}

// InvocationSmokeResult は invocation 系 provider doctor の smoke 実行結果を表す。
type InvocationSmokeResult struct {
	Ran           bool                           `json:"ran"`
	UsageObserved bool                           `json:"usage_observed"`
	Usage         SmokeUsage                     `json:"usage"`
	Cost          SmokeCost                      `json:"cost"`
	Requests      []InvocationSmokeRequestResult `json:"requests,omitempty"`
}

// NewSkippedInvocationSmokeRequest は skipped smoke entry を構築する。
func NewSkippedInvocationSmokeRequest(request InvocationSmokeRequest, skipReason string) InvocationSmokeRequestResult {
	return InvocationSmokeRequestResult{
		Name:            request.Name,
		Skipped:         true,
		SkipReason:      strings.TrimSpace(skipReason),
		ToolPayload:     request.ToolPayload,
		ImagePayload:    request.ImagePayload,
		ThinkingEnabled: request.ThinkingEnabled,
	}
}

// AddInvocationSmokeRequestResult は request-level smoke 結果を追加し summary の usage/cost を集約する。
func AddInvocationSmokeRequestResult(result *InvocationSmokeResult, request InvocationSmokeRequestResult) {
	if result == nil {
		return
	}
	result.Requests = append(result.Requests, request)
	if request.Skipped {
		return
	}
	result.Usage = AddSmokeUsage(result.Usage, request.Usage)
	result.Cost = AddSmokeCost(result.Cost, request.Cost)
	result.UsageObserved = AllInvocationSmokeRequestsObservedUsage(result.Requests)
}

// AllInvocationSmokeRequestsObservedUsage は skipped 以外の実行済み request すべてで usage が観測されたかを返す。
func AllInvocationSmokeRequestsObservedUsage(requests []InvocationSmokeRequestResult) bool {
	observedAnyRequest := false
	for _, request := range requests {
		if request.Skipped || !request.Ran {
			continue
		}
		observedAnyRequest = true
		if !request.UsageObserved {
			return false
		}
	}
	return observedAnyRequest
}
