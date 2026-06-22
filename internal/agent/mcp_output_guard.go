package agent

import (
	"context"
	"github.com/susugadx/xelyon-cli/internal/providerhistory"
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
	if a == nil || a.Runtime == nil || providerHistoryRawOutputArtifactsModeForRuntime(a.Runtime) != providerhistory.RawOutputArtifactsApply {
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
	if toolCall == nil || !strings.HasPrefix(toolCall.Tool, "mcp_") {
		return false
	}
	if len(result) <= mcpRuntimeResultInlineMaxBytes {
		return false
	}
	return true
}
