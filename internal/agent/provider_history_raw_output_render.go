package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/token"
)

func renderProviderHistoryRawOutputContextEntry(ref rawoutputs.RawOutputRef, body string, availableTokens int, hints []string) (string, string) {
	if availableTokens <= 0 {
		return "", providerHistoryRawOutputRequiredRefsMissingReason
	}
	body = providerHistoryRawOutputContextDisplayBody(ref, body)
	metadata := fmt.Sprintf(
		"- ref: %s\n  surface: %s\n  tool_name: %s\n  command_preview: %s\n  family: %s\n  classifier: %s\n  byte_size: %d\n  content_hash: %s\n  body:\n",
		ref.RefID,
		ref.Surface,
		ref.ToolName,
		ref.CommandPreview,
		ref.Family,
		ref.Classifier,
		ref.ByteSize,
		ref.ContentHash,
	)
	metadataTokens := token.EstimateTokenCount(metadata)
	bodyBudget := availableTokens - metadataTokens
	if bodyBudget <= 0 {
		return "", providerHistoryRawOutputRequiredRefsMissingReason
	}
	for bodyBudget > 0 {
		excerpt, reason := providerHistoryRawOutputBodyCoverageExcerpt(body, bodyBudget, hints)
		if strings.TrimSpace(excerpt) == "" {
			return "", reason
		}
		entry := metadata + indentRawOutputBody(excerpt)
		if token.EstimateTokenCount(entry) <= availableTokens {
			return entry, ""
		}
		if reason == providerHistoryRawOutputActiveContextCoverageInsufficientReason {
			return "", reason
		}
		bodyBudget = bodyBudget * 3 / 4
	}
	return "", providerHistoryRawOutputRequiredRefsMissingReason
}

func providerHistoryRawOutputContextDisplayBody(ref rawoutputs.RawOutputRef, body string) string {
	if ref.Surface == string(rawoutputs.SurfaceXelyonWebSearchToolResult) {
		return rawoutputs.RedactDisplaySecrets(body)
	}
	return body
}

func indentRawOutputBody(body string) string {
	lines := strings.Split(strings.TrimSpace(body), "\n")
	for i, line := range lines {
		lines[i] = "    " + line
	}
	return strings.Join(lines, "\n")
}
