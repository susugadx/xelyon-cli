package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/providerhistory"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
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
	ref, reason := a.createMCPRuntimeRawOutputArtifact(ctx, toolCall, execResult.Result)
	execResult.Result = buildMCPRuntimeResultPlaceholder(ref, reason, execResult.Result)
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

func (a *Agent) createMCPRuntimeRawOutputArtifact(ctx context.Context, toolCall *tools.ToolCall, result string) (rawoutputs.RawOutputRef, string) {
	if a == nil || a.Runtime == nil {
		return rawoutputs.RawOutputRef{}, "raw_output_artifact_runtime_missing"
	}
	if providerHistoryRawOutputArtifactsModeForRuntime(a.Runtime) != providerhistory.RawOutputArtifactsApply {
		return rawoutputs.RawOutputRef{}, mcpRuntimeRawOutputArtifactsDisabledReason(a.Runtime)
	}
	if rawoutputs.LooksSensitiveContent(result) {
		return rawoutputs.RawOutputRef{}, string(rawoutputs.ReasonSensitiveArtifactForbidden)
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
		return "raw_output_artifacts_dry_run"
	default:
		return "raw_output_artifacts_disabled"
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

func buildMCPRuntimeResultPlaceholder(ref rawoutputs.RawOutputRef, omittedReason string, content string) string {
	contentHash := mcpRuntimeContentHash(content)
	parts := []string{
		"[compacted MCP tool result;",
		fmt.Sprintf("surface=%s;", rawoutputs.SurfaceMCPToolResult),
		fmt.Sprintf("bytes=%d;", len(content)),
		fmt.Sprintf("runes=%d;", len([]rune(content))),
		fmt.Sprintf("sha256=%s;", mcpRuntimeHashPrefix(contentHash)),
	}
	if strings.TrimSpace(ref.RefID) != "" {
		parts = append(parts,
			fmt.Sprintf("raw_output_ref=%s;", ref.RefID),
			fmt.Sprintf("artifact_bytes=%d;", ref.ByteSize),
		)
	}
	if strings.TrimSpace(omittedReason) != "" {
		parts = append(parts, fmt.Sprintf("full_output_omitted_reason=%s;", omittedReason))
	}
	metadata := strings.Join(parts, "\n ") + "\n]"
	excerpt := mcpRuntimeBoundedRedactedExcerpt(content, mcpRuntimeResultExcerptMaxRunes)
	if excerpt == "" {
		return metadata
	}
	return metadata + "\nexcerpt:\n" + excerpt
}

func mcpRuntimeContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func mcpRuntimeHashPrefix(hash string) string {
	hash = strings.TrimSpace(hash)
	if strings.HasPrefix(hash, "sha256:") {
		value := strings.TrimPrefix(hash, "sha256:")
		if len(value) > mcpRuntimeResultHashPrefixLength {
			value = value[:mcpRuntimeResultHashPrefixLength]
		}
		return "sha256:" + value
	}
	if len(hash) > mcpRuntimeResultHashPrefixLength {
		return hash[:mcpRuntimeResultHashPrefixLength]
	}
	return hash
}

func mcpRuntimeBoundedRedactedExcerpt(content string, maxRunes int) string {
	content = strings.TrimSpace(rawoutputs.RedactDisplaySecrets(content))
	if content == "" || maxRunes <= 0 {
		return ""
	}
	runes := []rune(content)
	if len(runes) <= maxRunes {
		return content
	}
	headRunes := maxRunes / 2
	tailRunes := maxRunes - headRunes
	head := strings.TrimSpace(string(runes[:headRunes]))
	tail := strings.TrimSpace(string(runes[len(runes)-tailRunes:]))
	if head == "" {
		return tail
	}
	if tail == "" {
		return head
	}
	return head + "\n...\n" + tail
}
