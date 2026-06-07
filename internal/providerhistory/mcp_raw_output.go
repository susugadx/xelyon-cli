package providerhistory

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

func recordProviderHistoryMCPArtifactCandidate(report *ProjectionReport, policy Policy, entry ReductionCandidate, content string) bool {
	if report == nil || !providerHistoryMCPLooksDataBearing(content) || providerHistoryMCPLooksSensitive(content) {
		return false
	}
	entry.CandidateOnly = false
	entry.FutureApplyCandidate = true
	spec := providerHistoryDataBearingToolArtifactCandidateSpec{
		Surface: rawoutputs.SurfaceMCPToolResult,
		Source: rawoutputs.SourceMetadata{
			CommandHash:    commandHash(entry.ToolName + "\x00" + entry.ToolCallID),
			CommandPreview: entry.ToolName,
			ToolName:       entry.ToolName,
			ToolCallID:     entry.ToolCallID,
			EventID:        fmt.Sprintf("history:%d", entry.HistoryIndex),
			HistoryIndex:   entry.HistoryIndex,
		},
		Classification: rawoutputs.ClassificationMetadata{
			SemanticRole: "data_bearing",
			Family:       "mcp",
			Classifier:   "mcp_json_result",
		},
		Reason:                     "artifact_backed_mcp_tool_result",
		ReplacementKind:            "compact_old_mcp_tool_result",
		ArtifactsDisabledReason:    "mcp_raw_output_artifacts_disabled",
		MissingArtifactReason:      "mcp_raw_output_artifact_missing",
		RehydrateUnavailableReason: "mcp_raw_output_rehydrate_not_available",
		BuildPlaceholder: func(ref rawoutputs.RawOutputRef) string {
			return buildProviderHistoryRawOutputPlaceholder("MCP tool result", ref, content)
		},
	}
	recordProviderHistoryDataBearingToolArtifactCandidate(report, policy, entry, content, spec)
	return true
}

func providerHistoryMCPLooksDataBearing(content string) bool {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) < 2048 {
		return false
	}
	return strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")
}
