package providerhistory

import (
	"encoding/json"
	"strings"
)

func isProviderHistoryCandidateOnlyTool(toolName string) bool {
	switch toolName {
	case "wait_agent", "ask_user_question":
		return true
	default:
		return providerHistoryIsMCPToolResult(toolName) || providerHistoryIsProviderNativeReplayTool(toolName)
	}
}

func providerHistoryFutureFamilyName(toolName string) string {
	switch {
	case toolName == "wait_agent":
		return "wait_agent"
	case providerHistoryIsMCPToolResult(toolName):
		return "mcp"
	case toolName == "ask_user_question":
		return "ask_user_question"
	case providerHistoryIsProviderNativeReplayTool(toolName):
		return "provider_native_replay"
	default:
		return ""
	}
}

func providerHistoryFutureApplyCandidate(toolName, content string) bool {
	switch {
	case toolName == "wait_agent":
		return strings.TrimSpace(content) != ""
	case providerHistoryIsMCPToolResult(toolName):
		return strings.TrimSpace(content) != ""
	default:
		return false
	}
}

func providerHistoryCandidateOnlyKeepReason(toolName, content string) string {
	switch {
	case toolName == "wait_agent":
		if providerHistoryWaitAgentLooksErrorContext(content) {
			return "wait_agent_error_context_keep"
		}
		return "wait_agent_freeform_output_keep"
	case providerHistoryIsMCPToolResult(toolName):
		if providerHistoryMCPLooksSensitive(content) {
			return "mcp_sensitive_or_private_result_keep"
		}
		return "mcp_unknown_schema_keep"
	case toolName == "ask_user_question":
		if providerHistoryAskUserQuestionLooksApprovalRefusalOrPreference(content) {
			return "user_answer_approval_refusal_preference_keep"
		}
		return "user_answer_contract_keep"
	case providerHistoryIsProviderNativeReplayTool(toolName):
		if providerHistoryProviderNativeReplayLooksUsageAccounting(content) {
			return "provider_native_replay_usage_accounting_keep"
		}
		return "provider_native_replay_contract_keep"
	default:
		return "tool_not_in_reduction_allowlist"
	}
}

func providerHistoryIsMCPToolResult(toolName string) bool {
	return strings.HasPrefix(strings.TrimSpace(toolName), "mcp_")
}

func providerHistoryIsProviderNativeReplayTool(toolName string) bool {
	switch strings.TrimSpace(toolName) {
	case "$web_search", "builtin_web_search", "provider_native_web_search", "provider_native_builtin_replay":
		return true
	default:
		return false
	}
}

func providerHistoryWaitAgentLooksErrorContext(content string) bool {
	lower := strings.ToLower(content)
	for _, marker := range []string{"\"status\":\"failed\"", "\"status\":\"blocked\"", "status: failed", "status: blocked", "error:", "failed", "blocked"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func providerHistoryMCPLooksSensitive(content string) bool {
	lower := strings.ToLower(content)
	for _, marker := range []string{
		"secret",
		"token",
		"api_key",
		"apikey",
		"authorization",
		"password",
		"customer",
		"email",
		"private",
		"issue body",
		"message body",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func providerHistoryAskUserQuestionLooksApprovalRefusalOrPreference(content string) bool {
	var object map[string]any
	if err := json.Unmarshal([]byte(content), &object); err == nil {
		for _, key := range []string{"answer", "answers", "choice", "selected"} {
			if value, ok := object[key]; ok && providerHistoryUserAnswerValueLooksContractual(value) {
				return true
			}
		}
	}
	return providerHistoryUserAnswerTextLooksContractual(content)
}

func providerHistoryUserAnswerValueLooksContractual(value any) bool {
	switch v := value.(type) {
	case string:
		return providerHistoryUserAnswerTextLooksContractual(v)
	case []any:
		for _, item := range v {
			if providerHistoryUserAnswerValueLooksContractual(item) {
				return true
			}
		}
	case map[string]any:
		for _, item := range v {
			if providerHistoryUserAnswerValueLooksContractual(item) {
				return true
			}
		}
	}
	return false
}

func providerHistoryUserAnswerTextLooksContractual(content string) bool {
	lower := strings.ToLower(content)
	for _, marker := range []string{
		"approve",
		"approved",
		"allow",
		"yes",
		"no",
		"deny",
		"refuse",
		"stop",
		"prefer",
		"recommended",
		"permission",
		"dangerous",
		"destructive",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func providerHistoryProviderNativeReplayLooksUsageAccounting(content string) bool {
	lower := strings.ToLower(content)
	for _, marker := range []string{"usage", "tokens", "cost", "fee", "cached_tokens", "web_search_call_count"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
