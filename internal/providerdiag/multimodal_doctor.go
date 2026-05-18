package providerdiag

import "strings"

// MultimodalSmokeRequest は multimodal provider doctor の smoke request descriptor を表す。
type MultimodalSmokeRequest struct {
	Name             string
	ToolPayload      bool
	ImagePayload     bool
	ThinkingPayload  bool
	WebSearchPayload bool
	Route            string
}

// MultimodalSmokeRequestResult は multimodal provider doctor の request 単位結果を表す。
type MultimodalSmokeRequestResult struct {
	Name             string     `json:"name"`
	Ran              bool       `json:"ran"`
	Skipped          bool       `json:"skipped,omitempty"`
	SkipReason       string     `json:"skip_reason,omitempty"`
	ToolPayload      bool       `json:"tool_payload,omitempty"`
	ImagePayload     bool       `json:"image_payload,omitempty"`
	ThinkingPayload  bool       `json:"thinking_payload,omitempty"`
	WebSearchPayload bool       `json:"web_search_payload,omitempty"`
	Route            string     `json:"route"`
	Content          string     `json:"content,omitempty"`
	Duration         string     `json:"duration,omitempty"`
	UsageObserved    bool       `json:"usage_observed"`
	Usage            SmokeUsage `json:"usage"`
	Cost             SmokeCost  `json:"cost"`
	Error            string     `json:"error,omitempty"`
}

// MultimodalSmokeResult は thinking summary field を持たない multimodal provider doctor の smoke 実行結果を表す。
type MultimodalSmokeResult struct {
	Ran              bool                           `json:"ran"`
	ToolPayload      bool                           `json:"tool_payload"`
	ImagePayload     bool                           `json:"image_payload"`
	WebSearchPayload bool                           `json:"web_search_payload"`
	Route            string                         `json:"route"`
	Content          string                         `json:"content,omitempty"`
	Duration         string                         `json:"duration,omitempty"`
	UsageObserved    bool                           `json:"usage_observed"`
	Usage            SmokeUsage                     `json:"usage"`
	Cost             SmokeCost                      `json:"cost"`
	Requests         []MultimodalSmokeRequestResult `json:"requests,omitempty"`
}

// ThinkingMultimodalSmokeResult は thinking summary field を持つ multimodal provider doctor の smoke 実行結果を表す。
type ThinkingMultimodalSmokeResult struct {
	Ran              bool                           `json:"ran"`
	ToolPayload      bool                           `json:"tool_payload"`
	ImagePayload     bool                           `json:"image_payload"`
	ThinkingPayload  bool                           `json:"thinking_payload"`
	WebSearchPayload bool                           `json:"web_search_payload"`
	Route            string                         `json:"route"`
	Content          string                         `json:"content,omitempty"`
	Duration         string                         `json:"duration,omitempty"`
	UsageObserved    bool                           `json:"usage_observed"`
	Usage            SmokeUsage                     `json:"usage"`
	Cost             SmokeCost                      `json:"cost"`
	Requests         []MultimodalSmokeRequestResult `json:"requests,omitempty"`
}

// MultimodalRequestPreviewRequest は multimodal provider doctor の request preview 単位結果を表す。
type MultimodalRequestPreviewRequest struct {
	Name             string            `json:"name"`
	Skipped          bool              `json:"skipped,omitempty"`
	SkipReason       string            `json:"skip_reason,omitempty"`
	ToolPayload      bool              `json:"tool_payload,omitempty"`
	ImagePayload     bool              `json:"image_payload,omitempty"`
	ThinkingPayload  bool              `json:"thinking_payload,omitempty"`
	WebSearchPayload bool              `json:"web_search_payload,omitempty"`
	Route            string            `json:"route"`
	Method           string            `json:"method,omitempty"`
	URL              string            `json:"url,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	Body             any               `json:"body,omitempty"`
}

// NewMultimodalSmokeRequestResult は request descriptor から smoke request result の base を構築する。
func NewMultimodalSmokeRequestResult(request MultimodalSmokeRequest) MultimodalSmokeRequestResult {
	return MultimodalSmokeRequestResult{
		Name:             request.Name,
		ToolPayload:      request.ToolPayload,
		ImagePayload:     request.ImagePayload,
		ThinkingPayload:  request.ThinkingPayload,
		WebSearchPayload: request.WebSearchPayload,
		Route:            request.Route,
	}
}

// NewSkippedMultimodalSmokeRequest は skipped smoke entry を構築する。
func NewSkippedMultimodalSmokeRequest(request MultimodalSmokeRequest, skipReason string) MultimodalSmokeRequestResult {
	result := NewMultimodalSmokeRequestResult(request)
	result.Skipped = true
	result.SkipReason = strings.TrimSpace(skipReason)
	return result
}

// NewMultimodalPreviewRequest は request descriptor から request preview の base を構築する。
func NewMultimodalPreviewRequest(request MultimodalSmokeRequest) MultimodalRequestPreviewRequest {
	return MultimodalRequestPreviewRequest{
		Name:             request.Name,
		ToolPayload:      request.ToolPayload,
		ImagePayload:     request.ImagePayload,
		ThinkingPayload:  request.ThinkingPayload,
		WebSearchPayload: request.WebSearchPayload,
		Route:            request.Route,
	}
}

// NewSkippedMultimodalPreviewRequest は skipped preview entry を構築する。
func NewSkippedMultimodalPreviewRequest(request MultimodalSmokeRequest, skipReason string) MultimodalRequestPreviewRequest {
	result := NewMultimodalPreviewRequest(request)
	result.Skipped = true
	result.SkipReason = strings.TrimSpace(skipReason)
	return result
}

// AddMultimodalSmokeRequestResult は request-level smoke 結果を追加し summary に集約する。
func AddMultimodalSmokeRequestResult(result *MultimodalSmokeResult, request MultimodalSmokeRequestResult) {
	if result == nil {
		return
	}
	result.Requests = append(result.Requests, request)
	if request.Skipped {
		return
	}
	addMultimodalSmokeSummaryObservation(
		&result.ToolPayload,
		&result.ImagePayload,
		nil,
		&result.WebSearchPayload,
		&result.Route,
		&result.Content,
		&result.Usage,
		&result.Cost,
		request,
	)
	result.UsageObserved = AllMultimodalSmokeRequestsObservedUsage(result.Requests)
}

// AddThinkingMultimodalSmokeRequestResult は thinking field 付き summary に request-level smoke 結果を集約する。
func AddThinkingMultimodalSmokeRequestResult(result *ThinkingMultimodalSmokeResult, request MultimodalSmokeRequestResult) {
	if result == nil {
		return
	}
	result.Requests = append(result.Requests, request)
	if request.Skipped {
		return
	}
	addMultimodalSmokeSummaryObservation(
		&result.ToolPayload,
		&result.ImagePayload,
		&result.ThinkingPayload,
		&result.WebSearchPayload,
		&result.Route,
		&result.Content,
		&result.Usage,
		&result.Cost,
		request,
	)
	result.UsageObserved = AllMultimodalSmokeRequestsObservedUsage(result.Requests)
}

func addMultimodalSmokeSummaryObservation(
	summaryToolPayload *bool,
	summaryImagePayload *bool,
	summaryThinkingPayload *bool,
	summaryWebSearchPayload *bool,
	summaryRoute *string,
	summaryContent *string,
	summaryUsage *SmokeUsage,
	summaryCost *SmokeCost,
	request MultimodalSmokeRequestResult,
) {
	if request.ToolPayload {
		*summaryToolPayload = true
	}
	if request.ImagePayload {
		*summaryImagePayload = true
	}
	if summaryThinkingPayload != nil && request.ThinkingPayload {
		*summaryThinkingPayload = true
	}
	if request.WebSearchPayload {
		*summaryWebSearchPayload = true
	}
	switch {
	case *summaryRoute == "":
		*summaryRoute = request.Route
	case *summaryRoute != request.Route:
		*summaryRoute = "mixed"
	}
	if strings.TrimSpace(*summaryContent) == "" {
		*summaryContent = request.Content
	}
	*summaryUsage = AddSmokeUsage(*summaryUsage, request.Usage)
	*summaryCost = AddSmokeCost(*summaryCost, request.Cost)
}

// AllMultimodalSmokeRequestsObservedUsage は skipped 以外の実行済み request すべてで usage が観測されたかを返す。
func AllMultimodalSmokeRequestsObservedUsage(requests []MultimodalSmokeRequestResult) bool {
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
