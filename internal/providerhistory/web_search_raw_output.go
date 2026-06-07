package providerhistory

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/providerhistory/toolresults"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

func recordProviderHistoryWebSearchArtifactCandidate(report *ProjectionReport, policy Policy, entry ReductionCandidate, arguments, content string, messages []api.Message) bool {
	if report == nil {
		return false
	}
	analysis, reason, ok := toolresults.AnalyzeWebSearchResult(toolresults.NewReplacementRequestWithMessages(entry.ToolName, arguments, content, entry.ToolCallID, entry.HistoryIndex, messages))
	entry.FutureApplyCandidate = true
	if !ok {
		entry.CandidateOnly = true
		entry.KeepReason = reason
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return true
	}

	spec := providerHistoryDataBearingToolArtifactCandidateSpec{
		Surface: rawoutputs.SurfaceXelyonWebSearchToolResult,
		Source: rawoutputs.SourceMetadata{
			CommandHash:    analysis.QueryHash,
			CommandPreview: "web_search query=" + analysis.QueryPreview,
			ToolName:       entry.ToolName,
			ToolCallID:     entry.ToolCallID,
			EventID:        fmt.Sprintf("history:%d", entry.HistoryIndex),
			HistoryIndex:   entry.HistoryIndex,
		},
		Classification: rawoutputs.ClassificationMetadata{
			SemanticRole: "data_bearing",
			Family:       "web_search",
			Classifier:   "web_search_result",
		},
		Reason:                     "artifact_backed_xelyon_web_search_tool_result",
		ReplacementKind:            "compact_old_xelyon_web_search_tool_result",
		ArtifactsDisabledReason:    "web_search_raw_output_artifacts_disabled",
		MissingArtifactReason:      "web_search_raw_output_artifact_missing",
		RehydrateUnavailableReason: "web_search_raw_output_rehydrate_not_available",
		BuildPlaceholder: func(ref rawoutputs.RawOutputRef) string {
			return buildProviderHistoryWebSearchRawOutputPlaceholder(ref, analysis)
		},
	}
	recordProviderHistoryDataBearingToolArtifactCandidate(report, policy, entry, content, spec)
	return true
}

func buildProviderHistoryWebSearchRawOutputPlaceholder(ref rawoutputs.RawOutputRef, analysis toolresults.WebSearchAnalysis) string {
	const maxSelectedSources = 3
	var b strings.Builder
	fmt.Fprintf(
		&b,
		"[compacted old XELYON web_search tool result;\n raw_output_ref=%s;\n surface=%s;\n semantic_role=%s;\n family=%s;\n classifier=%s;\n bytes=%d;\n sha256=%s;\n query_hash=%s;\n query_preview=%q;\n results=%d;\n selected_urls=%d;\n duplicate_of=%s]\n",
		ref.RefID,
		ref.Surface,
		ref.SemanticRole,
		ref.Family,
		ref.Classifier,
		ref.ByteSize,
		providerHistoryRawOutputHashPrefix(ref.ContentHash),
		analysis.QueryHash,
		analysis.QueryPreview,
		len(analysis.Results),
		min(len(analysis.Results), maxSelectedSources),
		providerHistoryReductionSingleLine(analysis.DuplicateToolCallID),
	)
	b.WriteString("selected_sources:\n")
	limit := min(len(analysis.Results), maxSelectedSources)
	for _, result := range analysis.Results[:limit] {
		fmt.Fprintf(&b, "- %s (domain=%s)\n", result.URL, providerHistoryReductionSingleLine(result.Domain))
	}
	if omitted := len(analysis.Results) - limit; omitted > 0 {
		fmt.Fprintf(&b, "- +%d omitted raw sources\n", omitted)
	}
	b.WriteString("notes:\n")
	b.WriteString("- raw output is available through request-local active context when needed\n")
	b.WriteString("- source credibility is not upgraded by this compact summary")
	return b.String()
}
