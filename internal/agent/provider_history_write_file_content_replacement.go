package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ledger"
)

var providerHistoryWriteFileSuccessPattern = regexp.MustCompile(`(?m)^Successfully wrote \d+ bytes \(\d+ lines?\) to (.+?)(?:\r?\n|$)`)

type providerHistoryWriteFileContentReplacement struct {
	path            string
	originalContent string
	replacementText string
	replacementArgs string
	savedBytes      int
	savedTokens     int
}

func applyProviderHistoryWriteFileContentReplacementCandidate(report *ProviderHistoryCommandEditDryRunReport, candidateIndex int, ref providerHistoryAssistantToolCallRef, toolResultContent string, projection []api.Message) {
	if report == nil || candidateIndex < 0 || candidateIndex >= len(report.Candidates) {
		return
	}
	candidate := report.Candidates[candidateIndex]
	if candidate.Reason != "write_file_content" {
		return
	}
	if candidate.HistoryIndex < 0 || candidate.HistoryIndex >= len(projection) {
		return
	}
	if !providerHistoryCommandProjectionMessageMatchesCandidate(projection[candidate.HistoryIndex], candidate) {
		return
	}
	if ref.historyIndex < 0 || ref.historyIndex >= len(projection) {
		return
	}

	replacement, ok := buildProviderHistoryWriteFileContentReplacement(ref.arguments, toolResultContent)
	if !ok {
		return
	}
	if !applyProviderHistoryWriteFileContentReplacementProjection(&projection[ref.historyIndex], candidate.ToolCallID, replacement) {
		return
	}

	report.EditArgReplacedCount++
	report.EditArgReplacementSavedBytes += replacement.savedBytes
	report.ApproxEditArgReplacementSavedTokens += replacement.savedTokens
	report.ReplacementStatus = providerHistoryCommandEditReplacementStatusPartialApply
}

func buildProviderHistoryWriteFileContentReplacement(arguments, toolResultContent string) (providerHistoryWriteFileContentReplacement, bool) {
	fields, err := providerHistoryToolCallArgumentFields(arguments)
	if err != nil {
		return providerHistoryWriteFileContentReplacement{}, false
	}
	rawPath, ok := providerHistoryJSONStringArgument(fields, "path")
	if !ok || rawPath == "" {
		return providerHistoryWriteFileContentReplacement{}, false
	}
	path, ok := ledger.NormalizeRepoRelativePath(rawPath)
	if !ok {
		return providerHistoryWriteFileContentReplacement{}, false
	}
	content, ok := providerHistoryJSONStringArgument(fields, "content")
	if !ok || content == "" {
		return providerHistoryWriteFileContentReplacement{}, false
	}
	if !providerHistoryWriteFileResultSucceededForPath(toolResultContent, path) {
		return providerHistoryWriteFileContentReplacement{}, false
	}

	replacementText := buildProviderHistoryWriteFileContentPlaceholder(path)
	if len(replacementText) >= len(content) {
		return providerHistoryWriteFileContentReplacement{}, false
	}
	savedTokens := clampProviderHistorySavedTokens(token.EstimateTokenCount(content), token.EstimateTokenCount(replacementText))
	if savedTokens < providerHistoryEditArgReplacementMinSavedTokens {
		return providerHistoryWriteFileContentReplacement{}, false
	}

	encodedReplacement, err := json.Marshal(replacementText)
	if err != nil {
		return providerHistoryWriteFileContentReplacement{}, false
	}
	fields["content"] = encodedReplacement
	replacementArgs, err := json.Marshal(fields)
	if err != nil {
		return providerHistoryWriteFileContentReplacement{}, false
	}

	return providerHistoryWriteFileContentReplacement{
		path:            path,
		originalContent: content,
		replacementText: replacementText,
		replacementArgs: string(replacementArgs),
		savedBytes:      len(content) - len(replacementText),
		savedTokens:     savedTokens,
	}, true
}

func providerHistoryWriteFileResultSucceededForPath(content, path string) bool {
	resultPath, ok := providerHistoryWriteFileSuccessResultPath(content)
	return ok && resultPath == path
}

func providerHistoryWriteFileSuccessResultPath(content string) (string, bool) {
	matches := providerHistoryWriteFileSuccessPattern.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", false
	}
	return ledger.NormalizeRepoRelativePath(matches[1])
}

func buildProviderHistoryWriteFileContentPlaceholder(path string) string {
	return fmt.Sprintf("[omitted old write_file.content; path=%s]", providerHistoryReductionSingleLine(path))
}

func applyProviderHistoryWriteFileContentReplacementProjection(msg *api.Message, toolCallID string, replacement providerHistoryWriteFileContentReplacement) bool {
	if msg == nil || msg.Role != "assistant" || toolCallID == "" {
		return false
	}
	toolCallIndex := providerHistoryWriteFileToolCallIndex(*msg, toolCallID)
	if toolCallIndex < 0 {
		return false
	}
	if !syncProviderHistoryWriteFileContentAnthropicState(msg, toolCallID, replacement) {
		return false
	}

	msg.ToolCalls[toolCallIndex].Function.Arguments = replacement.replacementArgs
	return true
}

func providerHistoryWriteFileToolCallIndex(msg api.Message, toolCallID string) int {
	for i, toolCall := range msg.ToolCalls {
		if toolCall.ID != toolCallID {
			continue
		}
		if toolCall.Function.Name != "write_file" {
			return -1
		}
		return i
	}
	return -1
}

func syncProviderHistoryWriteFileContentAnthropicState(msg *api.Message, toolCallID string, replacement providerHistoryWriteFileContentReplacement) bool {
	if msg == nil {
		return false
	}
	blocks := msg.AnthropicContentBlocks()
	if len(blocks) == 0 {
		return true
	}

	updated := false
	for i := range blocks {
		block := &blocks[i]
		if block.Type != "tool_use" || block.ID != toolCallID {
			continue
		}
		if block.Name != "" && block.Name != "write_file" {
			return false
		}
		if !providerHistoryUpdateWriteFileInputContent(block.Input, replacement) {
			return false
		}
		updated = true
	}
	if !updated {
		return false
	}
	msg.SetAnthropicContentBlocks(blocks)
	return true
}

func providerHistoryUpdateWriteFileInputContent(input map[string]any, replacement providerHistoryWriteFileContentReplacement) bool {
	if len(input) == 0 {
		return false
	}
	pathValue, ok := input["path"].(string)
	if !ok || strings.TrimSpace(pathValue) == "" {
		return false
	}
	path, ok := ledger.NormalizeRepoRelativePath(pathValue)
	if !ok || path != replacement.path {
		return false
	}
	contentValue, ok := input["content"].(string)
	if !ok || contentValue != replacement.originalContent {
		return false
	}
	input["content"] = replacement.replacementText
	return true
}
