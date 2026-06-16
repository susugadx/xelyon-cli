package providerhistory

import (
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestProjectApplyCompactsOnlyDuplicateOldActivateSkillResult(t *testing.T) {
	body := providerHistoryTestLargeSkillBody("go-contract-design")
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_skill_old", "activate_skill", map[string]string{"name": "go-contract-design"})),
		providerHistoryTestToolResult("call_skill_old", "activate_skill", body),
		{Role: "assistant", Content: "skill activated"},
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_skill_dup", "activate_skill", map[string]string{"name": "go-contract-design"})),
		providerHistoryTestToolResult("call_skill_dup", "activate_skill", body),
		{Role: "assistant", Content: "duplicate skill remains"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{Messages: history, Policy: Policy{Mode: Apply}})

	if got := result.History[1].Content; got == body || !strings.Contains(got, "[compacted duplicate activate_skill result;") || !strings.Contains(got, "duplicate_of=call_skill_dup") {
		t.Fatalf("old activate_skill projection = %q, want duplicate compact placeholder", got)
	}
	if result.History[4].Content != body {
		t.Fatalf("later activate_skill raw body changed")
	}
	if result.Report.ReplacedCount != 1 || !result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want one replacement and response chain disabled", result.Report)
	}
	if got := result.Report.SkillReplacementToolCounts["activate_skill_duplicate"]; got != 1 {
		t.Fatalf("SkillReplacementToolCounts = %#v, want activate_skill_duplicate:1", result.Report.SkillReplacementToolCounts)
	}
	if got := result.Report.ContentReplacementToolCounts["activate_skill"]; got != 0 {
		t.Fatalf("ContentReplacementToolCounts[activate_skill] = %d in %#v, want 0; skill body replacements are reported via SkillReplacementToolCounts", got, result.Report.ContentReplacementToolCounts)
	}
}

func TestProjectApplyCompactsActivateSkillDuplicateFromAssistantLinkageVariants(t *testing.T) {
	body := "# test-coverage-improvement\n\n" + strings.Repeat("Skill instruction line that must stay raw only in the latest result.\nDO_NOT_LEAK_RAW_SKILL_CONTENT\n", 260)
	tests := []struct {
		name          string
		argumentKey   string
		oldToolName   string
		laterToolName string
	}{
		{
			name:          "skill argument with missing old and later tool result names",
			argumentKey:   "skill",
			oldToolName:   "",
			laterToolName: "",
		},
		{
			name:          "skill_name argument with missing later tool result name",
			argumentKey:   "skill_name",
			oldToolName:   "activate_skill",
			laterToolName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := []api.Message{
				providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_skill_old", "activate_skill", map[string]string{tt.argumentKey: "test-coverage-improvement"})),
				providerHistoryTestToolResult("call_skill_old", tt.oldToolName, body),
				{Role: "assistant", Content: "skill activated"},
				providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_skill_dup", "activate_skill", map[string]string{tt.argumentKey: "test-coverage-improvement"})),
				providerHistoryTestToolResult("call_skill_dup", tt.laterToolName, body),
				{Role: "assistant", Content: "duplicate skill remains"},
				providerHistoryTestAssistantToolCall("call_latest", "read_file"),
				providerHistoryTestToolResult("call_latest", "read_file", "latest"),
				{Role: "assistant", Content: "done"},
			}

			result := Project(ProjectionInput{Messages: history, Policy: Policy{Mode: Apply}})

			projected := result.History[1].Content
			if projected == body || !strings.Contains(projected, "[compacted duplicate activate_skill result;") {
				t.Fatalf("old activate_skill projection = %q, want duplicate compact placeholder", projected)
			}
			for _, want := range []string{
				`skill="test-coverage-improvement"`,
				"content_hash=sha256:",
				"raw_tool_call_id=call_skill_old",
				"duplicate_of=call_skill_dup",
			} {
				if !strings.Contains(projected, want) {
					t.Fatalf("activate_skill placeholder missing %q:\n%s", want, projected)
				}
			}
			for _, reject := range []string{"DO_NOT_LEAK_RAW_SKILL_CONTENT", "Skill instruction line that must stay raw only in the latest result"} {
				if strings.Contains(projected, reject) {
					t.Fatalf("activate_skill placeholder leaked raw skill content %q:\n%s", reject, projected)
				}
			}
			if result.History[1].ToolName != "activate_skill" {
				t.Fatalf("projected old tool name = %q, want activate_skill inferred from assistant tool call", result.History[1].ToolName)
			}
			if result.History[4].Content != body || result.History[4].ToolName != tt.laterToolName {
				t.Fatalf("later activate_skill result changed to name=%q content=%q, want raw latest duplicate", result.History[4].ToolName, result.History[4].Content)
			}
			if history[1].Content != body || history[1].ToolName != tt.oldToolName {
				t.Fatalf("raw history was mutated: name=%q content=%q", history[1].ToolName, history[1].Content)
			}
			candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_skill_old")
			if candidate == nil ||
				candidate.ToolName != "activate_skill" ||
				candidate.Reason != "old_duplicate_activate_skill_result" ||
				!candidate.ReplacementApplied ||
				candidate.SuggestedReplacementKind != "omit_duplicate_activate_skill_result" {
				t.Fatalf("candidate = %#v, want applied activate_skill duplicate candidate", candidate)
			}
			if result.Report.ReplacedCount != 1 ||
				result.Report.ReplacementStatus != providerHistoryReplacementStatusPartialApply ||
				!result.Report.ResponsesChainDisabled {
				t.Fatalf("report = %#v, want one applied skill replacement and response chain disabled", result.Report)
			}
			keptLatest := providerHistoryTestKeptByToolCallID(result.Report, "call_skill_dup")
			if keptLatest == nil || keptLatest.KeepReason != "activate_skill_latest_activation_keep" {
				t.Fatalf("latest duplicate kept entry = %#v, want activate_skill_latest_activation_keep", keptLatest)
			}
			if got := result.Report.SkillReplacementToolCounts["activate_skill_duplicate"]; got != 1 {
				t.Fatalf("SkillReplacementToolCounts = %#v, want activate_skill_duplicate:1", result.Report.SkillReplacementToolCounts)
			}
			if got := result.Report.ContentReplacementToolCounts["activate_skill"]; got != 0 {
				t.Fatalf("ContentReplacementToolCounts[activate_skill] = %d in %#v, want 0", got, result.Report.ContentReplacementToolCounts)
			}
		})
	}
}

func TestProjectApplyKeepsActivateSkillWhenDuplicateContractIsNotMet(t *testing.T) {
	tests := []struct {
		name    string
		oldBody string
		newBody string
		want    string
	}{
		{
			name:    "content hash mismatch",
			oldBody: providerHistoryTestLargeSkillBody("old"),
			newBody: providerHistoryTestLargeSkillBody("new"),
			want:    "activate_skill_hash_mismatch_keep",
		},
		{
			name:    "activation error",
			oldBody: "Error: skill not found\n" + strings.Repeat("details\n", 260),
			newBody: providerHistoryTestLargeSkillBody("valid"),
			want:    "activate_skill_error_keep",
		},
		{
			name:    "no later duplicate",
			oldBody: providerHistoryTestLargeSkillBody("only"),
			want:    "activate_skill_latest_activation_keep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := providerHistoryTestActivateSkillHistory(t, tt.oldBody, tt.newBody)
			result := Project(ProjectionInput{Messages: history, Policy: Policy{Mode: Apply}})
			if !reflect.DeepEqual(result.History, history) {
				t.Fatalf("projection changed kept activate_skill:\n got %#v\nwant %#v", result.History, history)
			}
			candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_skill")
			if candidate == nil || candidate.ReplacementApplied || candidate.KeepReason != tt.want {
				t.Fatalf("candidate = %#v, want kept reason %q", candidate, tt.want)
			}
		})
	}
}

func TestProjectApplyKeepsActivateSkillMissingNameAsCurrentBehaviorContract(t *testing.T) {
	body := providerHistoryTestLargeSkillBody("missing-name")
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_skill", "activate_skill", map[string]string{"unknown": "goal-plan-author"})),
		providerHistoryTestToolResult("call_skill", "activate_skill", body),
		{Role: "assistant", Content: "after skill"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{Messages: history, Policy: Policy{Mode: Apply}})

	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("projection changed activate_skill with missing name:\n got %#v\nwant %#v", result.History, history)
	}
	candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_skill")
	if candidate == nil || candidate.ReplacementApplied || candidate.KeepReason != "activate_skill_current_behavior_contract_keep" {
		t.Fatalf("candidate = %#v, want current behavior contract keep", candidate)
	}
}

func TestProjectApplyReportsCandidateOnlyFamiliesWithoutReplacingPayload(t *testing.T) {
	history := []api.Message{
		providerHistoryTestAssistantToolCall("call_wait", "wait_agent"),
		providerHistoryTestToolResult("call_wait", "wait_agent", strings.Repeat("finding: check internal/providerhistory/reduction.go\n", 220)),
		{Role: "assistant", Content: "wait result considered"},
		providerHistoryTestAssistantToolCall("call_wait_error", "wait_agent"),
		providerHistoryTestToolResult("call_wait_error", "wait_agent", `{"status":"failed","error":"sub-agent failed"}`),
		{Role: "assistant", Content: "wait error considered"},
		providerHistoryTestAssistantToolCall("call_script", "run_skill_script"),
		providerHistoryTestToolResult("call_script", "run_skill_script", strings.Repeat("ok script output\n", 220)),
		{Role: "assistant", Content: "script considered"},
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
		"call_script":          "run_skill_script_command_owner_unconfirmed",
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
		"run_skill_script":       1,
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
		"run_skill_script_command_owner_unconfirmed":   1,
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

func providerHistoryTestWebSearchHistory(t *testing.T, callID, query, content string, duplicate bool) []api.Message {
	t.Helper()
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, callID, "web_search", map[string]string{"query": query})),
		providerHistoryTestToolResult(callID, "web_search", content),
		{Role: "assistant", Content: "after web search"},
	}
	if duplicate {
		history = append(history,
			providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_web_later", "web_search", map[string]string{"query": query})),
			providerHistoryTestToolResult("call_web_later", "web_search", content),
			api.Message{Role: "assistant", Content: "after duplicate"},
		)
	}
	history = append(history,
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		api.Message{Role: "assistant", Content: "done"},
	)
	return history
}

func providerHistoryTestActivateSkillHistory(t *testing.T, oldBody, newBody string) []api.Message {
	t.Helper()
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_skill", "activate_skill", map[string]string{"name": "goal-plan-author"})),
		providerHistoryTestToolResult("call_skill", "activate_skill", oldBody),
		{Role: "assistant", Content: "after skill"},
	}
	if newBody != "" {
		history = append(history,
			providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_skill_later", "activate_skill", map[string]string{"name": "goal-plan-author"})),
			providerHistoryTestToolResult("call_skill_later", "activate_skill", newBody),
			api.Message{Role: "assistant", Content: "after later skill"},
		)
	}
	history = append(history,
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		api.Message{Role: "assistant", Content: "done"},
	)
	return history
}

func providerHistoryTestLargeWebSearchResult() string {
	return strings.Repeat(`1. OpenAI Responses API guide
URL: https://platform.openai.com/docs/guides/responses
The official docs describe response ids and follow-up calls.
2. OpenAI API reference
URL: https://platform.openai.com/docs/api-reference/responses
Reference material for Responses API fields.
`, 120)
}

func providerHistoryTestLargeSkillBody(name string) string {
	return "# " + name + "\n\n" + strings.Repeat("Skill instruction line for "+name+".\n", 260)
}

func providerHistoryTestMCPHistory(callID, content string) []api.Message {
	return []api.Message{
		providerHistoryTestAssistantToolCall(callID, "mcp_context7_get_library_docs"),
		providerHistoryTestToolResult(callID, "mcp_context7_get_library_docs", content),
		{Role: "assistant", Content: "after mcp result"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}
}

func providerHistoryTestLargeSafeMCPResult() string {
	return `{"items":[` + strings.Repeat(`{"title":"public metadata","value":"safe documentation result","score":1},`, 2600) + `{"title":"tail","value":"safe"}]}`
}

func providerHistoryTestLargeSensitiveMCPResult() string {
	return `{"items":[` + strings.Repeat(`{"title":"private issue body","email":"customer@example.test","token":"secret-token","value":"customer private message body"},`, 2600) + `{"title":"tail","value":"private customer"}]}`
}
