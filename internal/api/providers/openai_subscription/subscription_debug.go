package openaisubscription

import (
	"encoding/json"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func subscriptionDebugRequestPreview(request ResponsesRequest) []byte {
	payload, err := json.MarshalIndent(subscriptionDebugRequestPreviewBody(request), "", "  ")
	if err != nil {
		return []byte(`{"preview_error":"failed to render subscription debug preview"}`)
	}
	return payload
}

func subscriptionDebugRequestPreviewBody(request ResponsesRequest) map[string]any {
	preview := map[string]any{
		"model":                        request.Model,
		"stream":                       request.Stream,
		"store":                        request.Store,
		"instructions":                 subscriptionDebugPresenceLabel(request.Instructions),
		"input_items":                  subscriptionDebugInputShape(request.Input),
		"tools_count":                  len(request.Tools),
		"tool_choice":                  subscriptionDebugPresenceLabel(request.ToolChoice),
		"prompt_cache_key":             subscriptionDebugPresenceLabel(request.PromptCacheKey),
		"prompt_cache_retention":       subscriptionDebugPresenceLabel(request.PromptCacheRetention),
		"previous_response_id_present": strings.TrimSpace(request.PreviousResponseID) != "",
		"context_management_count":     len(request.ContextManagement),
		"max_output_tokens":            subscriptionDebugMaxOutputTokensPreview(request.MaxOutputTokens),
	}
	if request.Reasoning != nil {
		preview["reasoning_effort"] = request.Reasoning.Effort
	}
	return preview
}

func subscriptionDebugInputShape(input any) []map[string]any {
	items, ok := input.([]api.InputItem)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"type":              subscriptionDebugKnownInputItemType(item.Type),
			"role":              subscriptionDebugKnownRole(item.Role),
			"id":                subscriptionDebugPresenceLabel(item.ID),
			"status":            subscriptionDebugPresenceLabel(item.Status),
			"content":           subscriptionDebugPresenceLabel(item.Content),
			"data":              subscriptionDebugPresenceLabel(item.Data),
			"summary":           subscriptionDebugPresenceLabel(item.Summary),
			"encrypted":         subscriptionDebugPresenceLabel(item.EncryptedContent),
			"call_id":           subscriptionDebugPresenceLabel(item.CallID),
			"name":              subscriptionDebugPresenceLabel(item.Name),
			"arguments":         subscriptionDebugPresenceLabel(item.Arguments),
			"output":            subscriptionDebugPresenceLabel(item.Output),
			"thought_signature": subscriptionDebugPresenceLabel(item.ThoughtSignature),
			"thought_parts":     subscriptionDebugPresenceLabel(item.ThoughtParts),
		})
	}
	return out
}

func subscriptionDebugKnownInputItemType(value string) string {
	switch strings.TrimSpace(value) {
	case "message", "reasoning", "function_call", "function_call_output", "compacted":
		return strings.TrimSpace(value)
	case "":
		return "omitted"
	default:
		return "present"
	}
}

func subscriptionDebugKnownRole(value string) string {
	switch strings.TrimSpace(value) {
	case "user", "assistant", "developer", "system", "tool":
		return strings.TrimSpace(value)
	case "":
		return "omitted"
	default:
		return "present"
	}
}

func subscriptionDebugMaxOutputTokensPreview(value int) any {
	if value <= 0 {
		return "omitted"
	}
	return value
}

func subscriptionDebugPresenceLabel(value any) string {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return "omitted"
		}
		return "present"
	case []map[string]any:
		if len(typed) == 0 {
			return "omitted"
		}
		return "present"
	case nil:
		return "omitted"
	default:
		return "present"
	}
}
