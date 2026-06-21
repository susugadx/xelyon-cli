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
