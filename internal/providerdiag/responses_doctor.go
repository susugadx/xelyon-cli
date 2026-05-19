package providerdiag

import "strings"

// ResponsesSmokeRequest は Responses API 系 provider doctor の smoke request 単位を表す。
type ResponsesSmokeRequest struct {
	Name             string
	SystemPrompt     string
	UserContent      string
	ToolPayload      bool
	RetentionPayload bool
}

// ResponsesSmokeRequestResult は route を JSON contract に持たない Responses API 系 smoke request 結果を表す。
type ResponsesSmokeRequestResult struct {
	Name               string     `json:"name"`
	Ran                bool       `json:"ran"`
	Skipped            bool       `json:"skipped,omitempty"`
	SkipReason         string     `json:"skip_reason,omitempty"`
	ToolPayload        bool       `json:"tool_payload"`
	RetentionPayload   bool       `json:"retention_payload"`
	Content            string     `json:"content,omitempty"`
	ResponseID         string     `json:"response_id"`
	PreviousResponseID string     `json:"previous_response_id"`
	Duration           string     `json:"duration,omitempty"`
	UsageObserved      bool       `json:"usage_observed"`
	Usage              SmokeUsage `json:"usage"`
	Cost               SmokeCost  `json:"cost"`
	Error              string     `json:"error,omitempty"`
}

// ResponsesSmokeResult は route を JSON contract に持たない Responses API 系 smoke 実行結果を表す。
type ResponsesSmokeResult struct {
	Ran              bool                          `json:"ran"`
	ToolPayload      bool                          `json:"tool_payload"`
	RetentionPayload bool                          `json:"retention_payload"`
	Content          string                        `json:"content,omitempty"`
	ResponseID       string                        `json:"response_id"`
	Duration         string                        `json:"duration,omitempty"`
	UsageObserved    bool                          `json:"usage_observed"`
	Usage            SmokeUsage                    `json:"usage"`
	Cost             SmokeCost                     `json:"cost"`
	Requests         []ResponsesSmokeRequestResult `json:"requests,omitempty"`
}

// RoutedResponsesSmokeRequestResult は route を JSON contract に持つ Responses API 系 smoke request 結果を表す。
type RoutedResponsesSmokeRequestResult struct {
	Name               string     `json:"name"`
	Ran                bool       `json:"ran"`
	Skipped            bool       `json:"skipped,omitempty"`
	SkipReason         string     `json:"skip_reason,omitempty"`
	ToolPayload        bool       `json:"tool_payload"`
	RetentionPayload   bool       `json:"retention_payload"`
	Route              string     `json:"route"`
	Content            string     `json:"content,omitempty"`
	ResponseID         string     `json:"response_id"`
	PreviousResponseID string     `json:"previous_response_id"`
	Duration           string     `json:"duration,omitempty"`
	UsageObserved      bool       `json:"usage_observed"`
	Usage              SmokeUsage `json:"usage"`
	Cost               SmokeCost  `json:"cost"`
	Error              string     `json:"error,omitempty"`
}

// RoutedResponsesSmokeResult は route を JSON contract に持つ Responses API 系 smoke 実行結果を表す。
type RoutedResponsesSmokeResult struct {
	Ran              bool                                `json:"ran"`
	ToolPayload      bool                                `json:"tool_payload"`
	RetentionPayload bool                                `json:"retention_payload"`
	Route            string                              `json:"route"`
	Content          string                              `json:"content,omitempty"`
	ResponseID       string                              `json:"response_id"`
	Duration         string                              `json:"duration,omitempty"`
	UsageObserved    bool                                `json:"usage_observed"`
	Usage            SmokeUsage                          `json:"usage"`
	Cost             SmokeCost                           `json:"cost"`
	Requests         []RoutedResponsesSmokeRequestResult `json:"requests,omitempty"`
}

// ResponsesRequestPreviewRequest は Responses API 系 doctor preview request 単位結果を表す。
type ResponsesRequestPreviewRequest struct {
	Name               string            `json:"name"`
	Skipped            bool              `json:"skipped,omitempty"`
	SkipReason         string            `json:"skip_reason,omitempty"`
	ToolPayload        bool              `json:"tool_payload"`
	RetentionPayload   bool              `json:"retention_payload"`
	Route              string            `json:"route,omitempty"`
	Method             string            `json:"method,omitempty"`
	URL                string            `json:"url,omitempty"`
	Headers            map[string]string `json:"headers,omitempty"`
	PreviousResponseID string            `json:"previous_response_id,omitempty"`
	Body               any               `json:"body,omitempty"`
}

// RoutedResponsesRequestPreviewRequest は route を必須 field とする Responses API 系 doctor preview request 単位結果を表す。
type RoutedResponsesRequestPreviewRequest struct {
	Name               string            `json:"name"`
	Skipped            bool              `json:"skipped,omitempty"`
	SkipReason         string            `json:"skip_reason,omitempty"`
	ToolPayload        bool              `json:"tool_payload"`
	RetentionPayload   bool              `json:"retention_payload"`
	Route              string            `json:"route"`
	Method             string            `json:"method,omitempty"`
	URL                string            `json:"url,omitempty"`
	Headers            map[string]string `json:"headers,omitempty"`
	PreviousResponseID string            `json:"previous_response_id,omitempty"`
	Body               any               `json:"body,omitempty"`
}

// NewSkippedResponsesSmokeRequest は function calling 無効時の skipped tool smoke entry を構築する。
func NewSkippedResponsesSmokeRequest(request ResponsesSmokeRequest, skipReason string) ResponsesSmokeRequestResult {
	return ResponsesSmokeRequestResult{
		Name:             request.Name,
		Skipped:          true,
		SkipReason:       strings.TrimSpace(skipReason),
		ToolPayload:      request.ToolPayload,
		RetentionPayload: request.RetentionPayload,
	}
}

// NewSkippedRoutedResponsesSmokeRequest は route 付き skipped tool smoke entry を構築する。
func NewSkippedRoutedResponsesSmokeRequest(request ResponsesSmokeRequest, route, skipReason string) RoutedResponsesSmokeRequestResult {
	return RoutedResponsesSmokeRequestResult{
		Name:             request.Name,
		Skipped:          true,
		SkipReason:       strings.TrimSpace(skipReason),
		ToolPayload:      request.ToolPayload,
		RetentionPayload: request.RetentionPayload,
		Route:            route,
	}
}

// NewSkippedResponsesPreviewRequest は function calling 無効時の skipped tool preview entry を構築する。
func NewSkippedResponsesPreviewRequest(request ResponsesSmokeRequest, route, skipReason string) ResponsesRequestPreviewRequest {
	return ResponsesRequestPreviewRequest{
		Name:             request.Name,
		Skipped:          true,
		SkipReason:       strings.TrimSpace(skipReason),
		ToolPayload:      request.ToolPayload,
		RetentionPayload: request.RetentionPayload,
		Route:            route,
	}
}

// NewSkippedRoutedResponsesPreviewRequest は route 必須の skipped tool preview entry を構築する。
func NewSkippedRoutedResponsesPreviewRequest(request ResponsesSmokeRequest, route, skipReason string) RoutedResponsesRequestPreviewRequest {
	return RoutedResponsesRequestPreviewRequest{
		Name:             request.Name,
		Skipped:          true,
		SkipReason:       strings.TrimSpace(skipReason),
		ToolPayload:      request.ToolPayload,
		RetentionPayload: request.RetentionPayload,
		Route:            route,
	}
}

// NewResponsesPreviewRequest は Responses API 系 provider doctor の request preview entry を構築する。
func NewResponsesPreviewRequest(request ResponsesSmokeRequest, route string, transport RequestPreviewTransport) ResponsesRequestPreviewRequest {
	return ResponsesRequestPreviewRequest{
		Name:               request.Name,
		ToolPayload:        request.ToolPayload,
		RetentionPayload:   request.RetentionPayload,
		Route:              route,
		Method:             transport.Method,
		URL:                transport.URL,
		Headers:            transport.Headers,
		PreviousResponseID: strings.TrimSpace(transport.PreviousResponseID),
		Body:               transport.Body,
	}
}

// NewRoutedResponsesPreviewRequest は route 必須の Responses API 系 request preview entry を構築する。
func NewRoutedResponsesPreviewRequest(request ResponsesSmokeRequest, route string, transport RequestPreviewTransport) RoutedResponsesRequestPreviewRequest {
	return RoutedResponsesRequestPreviewRequest{
		Name:               request.Name,
		ToolPayload:        request.ToolPayload,
		RetentionPayload:   request.RetentionPayload,
		Route:              route,
		Method:             transport.Method,
		URL:                transport.URL,
		Headers:            transport.Headers,
		PreviousResponseID: strings.TrimSpace(transport.PreviousResponseID),
		Body:               transport.Body,
	}
}

// AddResponsesSmokeRequestResult は request-level smoke 結果を追加し summary に集約する。
func AddResponsesSmokeRequestResult(result *ResponsesSmokeResult, request ResponsesSmokeRequestResult) {
	if result == nil {
		return
	}
	result.Requests = append(result.Requests, request)
	if request.Skipped {
		return
	}
	addResponsesSmokeSummaryObservation(
		&result.ToolPayload,
		&result.RetentionPayload,
		&result.Content,
		&result.ResponseID,
		&result.Usage,
		&result.Cost,
		request.ToolPayload,
		request.RetentionPayload,
		request.Content,
		request.ResponseID,
		request.Usage,
		request.Cost,
	)
	result.UsageObserved = AllResponsesSmokeRequestsObservedUsage(result.Requests)
}

// AddRoutedResponsesSmokeRequestResult は route 付き request-level smoke 結果を追加し summary に集約する。
func AddRoutedResponsesSmokeRequestResult(result *RoutedResponsesSmokeResult, request RoutedResponsesSmokeRequestResult) {
	if result == nil {
		return
	}
	result.Requests = append(result.Requests, request)
	if request.Skipped {
		return
	}
	if result.Route == "" {
		result.Route = request.Route
	}
	addResponsesSmokeSummaryObservation(
		&result.ToolPayload,
		&result.RetentionPayload,
		&result.Content,
		&result.ResponseID,
		&result.Usage,
		&result.Cost,
		request.ToolPayload,
		request.RetentionPayload,
		request.Content,
		request.ResponseID,
		request.Usage,
		request.Cost,
	)
	result.UsageObserved = AllRoutedResponsesSmokeRequestsObservedUsage(result.Requests)
}

func addResponsesSmokeSummaryObservation(
	summaryToolPayload *bool,
	summaryRetentionPayload *bool,
	summaryContent *string,
	summaryResponseID *string,
	summaryUsage *SmokeUsage,
	summaryCost *SmokeCost,
	requestToolPayload bool,
	requestRetentionPayload bool,
	requestContent string,
	requestResponseID string,
	requestUsage SmokeUsage,
	requestCost SmokeCost,
) {
	if requestToolPayload {
		*summaryToolPayload = true
	}
	if requestRetentionPayload {
		*summaryRetentionPayload = true
	}
	if strings.TrimSpace(*summaryContent) == "" {
		*summaryContent = requestContent
	}
	if strings.TrimSpace(*summaryResponseID) == "" {
		*summaryResponseID = requestResponseID
	}
	*summaryUsage = AddSmokeUsage(*summaryUsage, requestUsage)
	*summaryCost = AddSmokeCost(*summaryCost, requestCost)
}

// AllResponsesSmokeRequestsObservedUsage は skipped 以外の実行済み request すべてで usage が観測されたかを返す。
func AllResponsesSmokeRequestsObservedUsage(requests []ResponsesSmokeRequestResult) bool {
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

// AllRoutedResponsesSmokeRequestsObservedUsage は route 付き request の usage 観測状態を集約する。
func AllRoutedResponsesSmokeRequestsObservedUsage(requests []RoutedResponsesSmokeRequestResult) bool {
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
