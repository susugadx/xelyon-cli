package providerhistory

import (
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestProjectApplyReportsCandidateOnlyFamiliesWithoutReplacingPayload(t *testing.T) {
	history := []api.Message{
		providerHistoryTestAssistantToolCall("call_wait", "wait_agent"),
		providerHistoryTestToolResult("call_wait", "wait_agent", strings.Repeat("finding: check internal/providerhistory/reduction.go\n", 220)),
		{Role: "assistant", Content: "wait result considered"},
		providerHistoryTestAssistantToolCall("call_wait_error", "wait_agent"),
		providerHistoryTestToolResult("call_wait_error", "wait_agent", `{"status":"failed","error":"sub-agent failed"}`),
		{Role: "assistant", Content: "wait error considered"},
		providerHistoryTestAssistantToolCall("call_mcp", "mcp_github_get_issue"),
		providerHistoryTestToolResult("call_mcp", "mcp_github_get_issue", `{"url":"https://github.com/org/repo/issues/1","title":"bug","summary":"private customer email body"}`),
		{Role: "assistant", Content: "mcp considered"},
		providerHistoryTestAssistantToolCall("call_mcp_unknown", "mcp_figma_get_file"),
		providerHistoryTestToolResult("call_mcp_unknown", "mcp_figma_get_file", `{"url":"https://figma.example/file","title":"public frame","summary":"layout metadata only"}`),
		{Role: "assistant", Content: "mcp unknown considered"},
		providerHistoryTestAssistantToolCall("call_ask", "ask_user_question"),
		providerHistoryTestToolResult("call_ask", "ask_user_question", `{"question":"Proceed?","answer":"approved"}`),
		{Role: "assistant", Content: "answer considered"},
		providerHistoryTestAssistantToolCall("call_ask_contract", "ask_user_question"),
		providerHistoryTestToolResult("call_ask_contract", "ask_user_question", `{"question":"Which label?","answer":"Use compact labels"}`),
		{Role: "assistant", Content: "answer contract considered"},
		providerHistoryTestAssistantToolCall("call_native", "$web_search"),
		providerHistoryTestToolResult("call_native", "$web_search", `{"usage":{"tokens":123},"result":"provider-native replay"}`),
		{Role: "assistant", Content: "native replay considered"},
		providerHistoryTestAssistantToolCall("call_native_contract", "$web_search"),
		providerHistoryTestToolResult("call_native_contract", "$web_search", `{"result":"provider-native replay payload"}`),
		{Role: "assistant", Content: "native replay contract considered"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{Messages: history, Policy: Policy{Mode: Apply}})

	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("candidate-only projection changed payload:\n got %#v\nwant %#v", result.History, history)
	}
	if result.Report.ReplacedCount != 0 || result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want no replacement and response chain preserved", result.Report)
	}
	for id, want := range map[string]string{
		"call_wait":            "wait_agent_freeform_output_keep",
		"call_wait_error":      "wait_agent_error_context_keep",
		"call_mcp":             "mcp_sensitive_or_private_result_keep",
		"call_mcp_unknown":     "mcp_unknown_schema_keep",
		"call_ask":             "user_answer_approval_refusal_preference_keep",
		"call_ask_contract":    "user_answer_contract_keep",
		"call_native":          "provider_native_replay_usage_accounting_keep",
		"call_native_contract": "provider_native_replay_contract_keep",
	} {
		candidate := providerHistoryTestCandidateByToolCallID(result.Report, id)
		if candidate == nil || !candidate.CandidateOnly || candidate.KeepReason != want {
			t.Fatalf("%s candidate = %#v, want candidate-only kept reason %q", id, candidate, want)
		}
	}
	for family, want := range map[string]int{
		"wait_agent":             2,
		"mcp":                    2,
		"ask_user_question":      2,
		"provider_native_replay": 2,
	} {
		if got := result.Report.FutureFamilyCandidateCounts[family]; got != want {
			t.Fatalf("FutureFamilyCandidateCounts[%s] = %d in %#v, want %d", family, got, result.Report.FutureFamilyCandidateCounts, want)
		}
	}
	for reason, want := range map[string]int{
		"wait_agent_freeform_output_keep":              1,
		"wait_agent_error_context_keep":                1,
		"mcp_sensitive_or_private_result_keep":         1,
		"mcp_unknown_schema_keep":                      1,
		"user_answer_approval_refusal_preference_keep": 1,
		"user_answer_contract_keep":                    1,
		"provider_native_replay_usage_accounting_keep": 1,
		"provider_native_replay_contract_keep":         1,
	} {
		if got := result.Report.FutureFamilyKeptReasonCounts[reason]; got != want {
			t.Fatalf("FutureFamilyKeptReasonCounts[%s] = %d in %#v, want %d", reason, got, result.Report.FutureFamilyKeptReasonCounts, want)
		}
	}
	if got := result.Report.FutureFamilyKeptReasonCounts["latest_tool_result"]; got != 0 {
		t.Fatalf("FutureFamilyKeptReasonCounts includes latest_tool_result: %#v", result.Report.FutureFamilyKeptReasonCounts)
	}
}
