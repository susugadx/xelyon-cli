package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/susugadx/xelyon-cli/internal/providerhistory"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"strings"
)

const (
	mcpRuntimeRawOutputArtifactsDryRunReason        = "raw_output_artifacts_dry_run"
	mcpRuntimeRawOutputArtifactsDisabledReasonValue = "raw_output_artifacts_disabled"
)

func (a *Agent) createMCPRuntimeRawOutputArtifact(ctx context.Context, toolCall *tools.ToolCall, result string) (rawoutputs.RawOutputRef, string) {
	if reason := providerhistory.MCPRawOutputArtifactOmitReason(result); reason != "" {
		return rawoutputs.RawOutputRef{}, reason
	}
	if a == nil || a.Runtime == nil {
		return rawoutputs.RawOutputRef{}, "raw_output_artifact_runtime_missing"
	}
	if providerHistoryRawOutputArtifactsModeForRuntime(a.Runtime) != providerhistory.RawOutputArtifactsApply {
		return rawoutputs.RawOutputRef{}, mcpRuntimeRawOutputArtifactsDisabledReason(a.Runtime)
	}
	sessionID := strings.TrimSpace(a.providerHistoryRawOutputArtifactSessionID())
	if sessionID == "" {
		return rawoutputs.RawOutputRef{}, "raw_output_artifact_session_missing"
	}
	store := a.providerHistoryRawOutputArtifactStore()
	if store == nil {
		return rawoutputs.RawOutputRef{}, "raw_output_artifact_store_unavailable"
	}

	req := rawoutputs.CreateRequest{
		Surface:   rawoutputs.SurfaceMCPToolResult,
		SessionID: sessionID,
		Source: rawoutputs.SourceMetadata{
			Provider:       a.ProviderName,
			Model:          a.CurrentModel,
			CommandHash:    mcpRuntimeToolCallHash(toolCall),
			CommandPreview: rawoutputs.SanitizeDisplayPreview(mcpRuntimeToolCallPreview(toolCall), 200),
			ToolName:       mcpRuntimeToolName(toolCall),
			ToolCallID:     mcpRuntimeToolCallID(toolCall),
			EventID:        mcpRuntimeToolEventID(toolCall),
		},
		Classification: rawoutputs.ClassificationMetadata{
			SemanticRole: "data_bearing",
			Family:       "mcp",
			Classifier:   "mcp_runtime_large_result",
		},
		Body:          strings.NewReader(result),
		SizeHintBytes: int64(len(result)),
	}
	createResult, err := store.Create(ctx, req)
	if err != nil {
		return rawoutputs.RawOutputRef{}, mcpRuntimeRawOutputFailureReason(err, "raw_output_artifact_create_failed")
	}
	verifyResult, err := store.Verify(ctx, createResult.Ref)
	if err != nil {
		return rawoutputs.RawOutputRef{}, mcpRuntimeRawOutputFailureReason(err, "raw_output_artifact_verify_failed")
	}
	if !verifyResult.OK {
		if verifyResult.Reason != "" {
			return rawoutputs.RawOutputRef{}, string(verifyResult.Reason)
		}
		return rawoutputs.RawOutputRef{}, "raw_output_artifact_verify_failed"
	}
	return createResult.Ref, ""
}

func mcpRuntimeRawOutputArtifactsDisabledReason(runtime *AgentRuntime) string {
	switch providerHistoryRawOutputArtifactsModeForRuntime(runtime) {
	case providerhistory.RawOutputArtifactsDryRun:
		return mcpRuntimeRawOutputArtifactsDryRunReason
	default:
		return mcpRuntimeRawOutputArtifactsDisabledReasonValue
	}
}

func mcpRuntimeRawOutputFailureReason(err error, fallback string) string {
	if reason := rawoutputs.ReasonOf(err); reason != "" {
		return string(reason)
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "raw_output_artifact_failed"
}

func mcpRuntimeToolName(toolCall *tools.ToolCall) string {
	if toolCall == nil {
		return ""
	}
	return toolCall.Tool
}

func mcpRuntimeToolCallID(toolCall *tools.ToolCall) string {
	if toolCall == nil {
		return ""
	}
	return toolCall.ID
}

func mcpRuntimeToolEventID(toolCall *tools.ToolCall) string {
	if toolCall == nil {
		return ""
	}
	if strings.TrimSpace(toolCall.ID) != "" {
		return "tool_call:" + strings.TrimSpace(toolCall.ID)
	}
	return "tool_call:" + mcpRuntimeToolCallHash(toolCall)
}

func mcpRuntimeToolCallPreview(toolCall *tools.ToolCall) string {
	if toolCall == nil {
		return ""
	}
	if len(toolCall.Args) == 0 {
		return toolCall.Tool
	}
	argsBytes, err := json.Marshal(toolCall.Args)
	if err != nil {
		return toolCall.Tool
	}
	return toolCall.Tool + " " + string(argsBytes)
}

func mcpRuntimeToolCallHash(toolCall *tools.ToolCall) string {
	if toolCall == nil {
		return ""
	}
	payload := struct {
		Tool    string            `json:"tool"`
		Args    map[string]string `json:"args,omitempty"`
		RawArgs map[string]any    `json:"raw_args,omitempty"`
		ID      string            `json:"id,omitempty"`
	}{
		Tool:    toolCall.Tool,
		Args:    toolCall.Args,
		RawArgs: toolCall.RawArgs,
		ID:      toolCall.ID,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		data = []byte(toolCall.Tool)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
