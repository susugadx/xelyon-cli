package toolresults

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/token"
)

const activateSkillToolName = "activate_skill"

func buildActivateSkillReplacement(req ReplacementRequest) (Replacement, string, bool) {
	skillName, ok := activateSkillNameArgument(req.arguments)
	if !ok {
		return Replacement{}, "activate_skill_current_behavior_contract_keep", false
	}
	if activateSkillContentIsError(req.content) {
		return Replacement{}, "activate_skill_error_keep", false
	}
	contentHash := providerHistoryToolResultHash(req.content)
	duplicate, mismatch := laterActivateSkillResult(req, skillName, contentHash)
	if mismatch {
		return Replacement{}, "activate_skill_hash_mismatch_keep", false
	}
	if duplicate.toolCallID == "" {
		return Replacement{}, "activate_skill_latest_activation_keep", false
	}

	replacementText := fmt.Sprintf(
		"[compacted duplicate activate_skill result; skill=%q; content_hash=%s; raw_size=%d; raw_tool_call_id=%s; duplicate_of=%s]",
		skillName,
		contentHash,
		len(req.content),
		singleLine(req.toolCallID),
		singleLine(duplicate.toolCallID),
	)
	return Replacement{
		kind:        "omit_duplicate_activate_skill_result",
		text:        replacementText,
		savedBytes:  savedBytes(len(req.content), len(replacementText)),
		savedTokens: savedTokens(token.EstimateTokenCount(req.content), token.EstimateTokenCount(replacementText)),
	}, "", true
}

func activateSkillNameArgument(arguments string) (string, bool) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(arguments), &fields); err != nil {
		return "", false
	}
	for _, key := range []string{"name", "skill", "skill_name"} {
		raw, ok := fields[key]
		if !ok {
			continue
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value, true
		}
	}
	return "", false
}

func activateSkillContentIsError(content string) bool {
	first := strings.TrimSpace(strings.SplitN(content, "\n", 2)[0])
	lower := strings.ToLower(first)
	return strings.HasPrefix(lower, "error:") ||
		strings.Contains(lower, "not found") ||
		strings.Contains(lower, "failed")
}

type duplicateActivateSkillResult struct {
	toolCallID string
}

func laterActivateSkillResult(req ReplacementRequest, skillName, contentHash string) (duplicateActivateSkillResult, bool) {
	if req.historyIndex < 0 || req.historyIndex >= len(req.messages)-1 {
		return duplicateActivateSkillResult{}, false
	}
	hashMismatch := false
	for i := req.historyIndex + 1; i < len(req.messages); i++ {
		msg := req.messages[i]
		if msg.Role != "tool" {
			continue
		}
		toolName := strings.TrimSpace(msg.ToolName)
		if toolName == "" {
			toolName = toolNameForToolResultAt(req.messages, i)
		}
		if toolName != activateSkillToolName {
			continue
		}
		args := toolArgumentsForToolResultAt(req.messages, i)
		laterSkillName, ok := activateSkillNameArgument(args)
		if !ok || laterSkillName != skillName {
			continue
		}
		if activateSkillContentIsError(msg.Content) {
			continue
		}
		if providerHistoryToolResultHash(msg.Content) != contentHash {
			hashMismatch = true
			continue
		}
		return duplicateActivateSkillResult{toolCallID: msg.ToolCallID}, false
	}
	return duplicateActivateSkillResult{}, hashMismatch
}
