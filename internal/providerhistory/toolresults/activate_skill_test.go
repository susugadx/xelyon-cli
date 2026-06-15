package toolresults

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestBuildStructuredReplacementActivateSkillCompactsOnlyOlderDuplicate(t *testing.T) {
	content := strings.Repeat("Skill instructions for codebase-specific workflow.\nDO_NOT_LEAK_RAW_SKILL_CONTENT\n", 80)

	for _, tt := range []struct {
		name        string
		argumentKey string
		emptyName   bool
	}{
		{name: "name argument with tool name", argumentKey: "name"},
		{name: "skill argument with inferred tool name", argumentKey: "skill", emptyName: true},
		{name: "skill_name argument", argumentKey: "skill_name"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			messages := activateSkillMessages(t, "call_old", "call_later", tt.argumentKey, "test-coverage-improvement", content, content, tt.emptyName)

			replacement, reason, ok := BuildStructuredReplacement(NewReplacementRequestWithMessages(
				"activate_skill",
				activateSkillArgs(t, tt.argumentKey, "test-coverage-improvement"),
				content,
				"call_old",
				1,
				messages,
			))
			if !ok || reason != "" {
				t.Fatalf("BuildStructuredReplacement() = (%#v, %q, %v), want duplicate compact", replacement, reason, ok)
			}
			if replacement.Kind() != "omit_duplicate_activate_skill_result" {
				t.Fatalf("Kind = %q, want omit_duplicate_activate_skill_result", replacement.Kind())
			}
			text := replacement.Text()
			for _, want := range []string{
				`skill="test-coverage-improvement"`,
				"content_hash=sha256:",
				"raw_tool_call_id=call_old",
				"duplicate_of=call_later",
			} {
				if !strings.Contains(text, want) {
					t.Fatalf("replacement text missing %q:\n%s", want, text)
				}
			}
			for _, reject := range []string{"DO_NOT_LEAK_RAW_SKILL_CONTENT", "Skill instructions for codebase-specific workflow"} {
				if strings.Contains(text, reject) {
					t.Fatalf("replacement text leaked raw skill content %q:\n%s", reject, text)
				}
			}
			if replacement.SavedBytes() <= 0 || replacement.SavedTokens() <= 0 {
				t.Fatalf("saved metrics = bytes %d tokens %d, want positive", replacement.SavedBytes(), replacement.SavedTokens())
			}
		})
	}
}

func TestBuildStructuredReplacementActivateSkillKeepsCurrentOrUnsafeResults(t *testing.T) {
	content := strings.Repeat("Skill instructions for provider history workflow.\n", 80)
	tests := []struct {
		name       string
		arguments  string
		content    string
		messages   []api.Message
		wantReason string
	}{
		{
			name:       "latest activation",
			arguments:  activateSkillArgs(t, "name", "providerhistory"),
			content:    content,
			messages:   activateSkillMessages(t, "call_old", "", "name", "providerhistory", content, "", false),
			wantReason: "activate_skill_latest_activation_keep",
		},
		{
			name:       "error content",
			arguments:  activateSkillArgs(t, "name", "providerhistory"),
			content:    "Error: skill providerhistory not found",
			messages:   activateSkillMessages(t, "call_old", "call_later", "name", "providerhistory", "Error: skill providerhistory not found", content, false),
			wantReason: "activate_skill_error_keep",
		},
		{
			name:       "missing skill argument",
			arguments:  `{"path":"providerhistory"}`,
			content:    content,
			messages:   activateSkillMessages(t, "call_old", "call_later", "name", "providerhistory", content, content, false),
			wantReason: "activate_skill_current_behavior_contract_keep",
		},
		{
			name:       "invalid skill argument",
			arguments:  `{"name":42}`,
			content:    content,
			messages:   activateSkillMessages(t, "call_old", "call_later", "name", "providerhistory", content, content, false),
			wantReason: "activate_skill_current_behavior_contract_keep",
		},
		{
			name:       "same skill different content hash",
			arguments:  activateSkillArgs(t, "name", "providerhistory"),
			content:    content,
			messages:   activateSkillMessages(t, "call_old", "call_later", "name", "providerhistory", content, content+"\nnew instruction", false),
			wantReason: "activate_skill_hash_mismatch_keep",
		},
		{
			name:       "different skill",
			arguments:  activateSkillArgs(t, "name", "providerhistory"),
			content:    content,
			messages:   activateSkillMessages(t, "call_old", "call_later", "name", "other-skill", content, content, false),
			wantReason: "activate_skill_latest_activation_keep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replacement, reason, ok := BuildStructuredReplacement(NewReplacementRequestWithMessages(
				"activate_skill",
				tt.arguments,
				tt.content,
				"call_old",
				1,
				tt.messages,
			))
			if ok || reason != tt.wantReason {
				t.Fatalf("BuildStructuredReplacement() = (%#v, %q, %v), want keep reason %q", replacement, reason, ok, tt.wantReason)
			}
			if replacement.Text() != "" || replacement.SavedBytes() != 0 || replacement.SavedTokens() != 0 {
				t.Fatalf("replacement = %#v, want empty replacement for keep", replacement)
			}
		})
	}
}

func activateSkillMessages(t *testing.T, oldCallID, laterCallID, argumentKey, skillName, oldContent, laterContent string, emptyLaterToolName bool) []api.Message {
	t.Helper()
	messages := []api.Message{
		activateSkillAssistantMessage(t, oldCallID, argumentKey, skillName),
		activateSkillToolMessage(oldCallID, "activate_skill", oldContent),
		{Role: "assistant", Content: "loaded skill"},
	}
	if laterCallID != "" {
		laterToolName := "activate_skill"
		if emptyLaterToolName {
			laterToolName = ""
		}
		messages = append(messages,
			activateSkillAssistantMessage(t, laterCallID, argumentKey, skillName),
			activateSkillToolMessage(laterCallID, laterToolName, laterContent),
			api.Message{Role: "assistant", Content: "loaded duplicate skill"},
		)
	}
	return messages
}

func activateSkillAssistantMessage(t *testing.T, callID, argumentKey, skillName string) api.Message {
	t.Helper()
	return api.Message{
		Role:    "assistant",
		Content: "activating skill",
		ToolCalls: []api.OpenAIToolCall{{
			ID:   callID,
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      "activate_skill",
				Arguments: activateSkillArgs(t, argumentKey, skillName),
			},
		}},
	}
}

func activateSkillToolMessage(callID, toolName, content string) api.Message {
	return api.Message{
		Role:       "tool",
		ToolName:   toolName,
		ToolCallID: callID,
		Content:    content,
	}
}

func activateSkillArgs(t *testing.T, key, skillName string) string {
	t.Helper()
	data, err := json.Marshal(map[string]string{key: skillName})
	if err != nil {
		t.Fatalf("json.Marshal(skill args) error = %v", err)
	}
	return string(data)
}
