package agent

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/mcpnames"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

const (
	mcpRuntimeResultInlineMaxBytes   = 64 * 1024
	mcpRuntimeResultExcerptMaxRunes  = 12000
	mcpRuntimeResultHashPrefixLength = 12
)

func (a *Agent) guardMCPToolExecutionResult(ctx context.Context, toolCall *tools.ToolCall, execResult tools.ExecutionResult) tools.ExecutionResult {
	if !shouldCompactMCPToolResult(toolCall, execResult.Result) {
		return execResult
	}
	ref, omittedReason := a.createMCPRuntimeRawOutputArtifact(ctx, toolCall, execResult.Result)
	if strings.TrimSpace(ref.RefID) == "" {
		if strings.TrimSpace(omittedReason) == "" {
			omittedReason = "raw_output_ref_missing"
		}
		execResult.Result = buildMCPRuntimeResultPlaceholder(ref, omittedReason, execResult.Result)
		return execResult
	}
	execResult.Result = buildMCPRuntimeResultPlaceholder(ref, "", execResult.Result)
	return execResult
}

func shouldCompactMCPToolResult(toolCall *tools.ToolCall, result string) bool {
	if toolCall == nil || !mcpnames.IsExportedToolName(toolCall.Tool) {
		return false
	}
	if len(result) <= mcpRuntimeResultInlineMaxBytes {
		return false
	}
	return true
}
