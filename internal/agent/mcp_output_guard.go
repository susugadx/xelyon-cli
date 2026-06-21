package agent

import (
	"context"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"strings"
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
		if mcpRuntimeOmitReasonKeepsOriginalResult(omittedReason) {
			return execResult
		}
		if strings.TrimSpace(omittedReason) == "" {
			omittedReason = "raw_output_ref_missing"
		}
		execResult.Result = buildMCPRuntimeResultPlaceholder(ref, omittedReason, execResult.Result)
		return execResult
	}
	execResult.Result = buildMCPRuntimeResultPlaceholder(ref, "", execResult.Result)
	return execResult
}

func mcpRuntimeOmitReasonKeepsOriginalResult(reason string) bool {
	switch strings.TrimSpace(reason) {
	case mcpRuntimeRawOutputArtifactsDryRunReason, mcpRuntimeRawOutputArtifactsDisabledReasonValue:
		return true
	default:
		return false
	}
}

func shouldCompactMCPToolResult(toolCall *tools.ToolCall, result string) bool {
	if toolCall == nil || !strings.HasPrefix(toolCall.Tool, "mcp_") {
		return false
	}
	if len(result) <= mcpRuntimeResultInlineMaxBytes {
		return false
	}
	return true
}
