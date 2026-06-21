package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/susugadx/xelyon-cli/internal/providerhistory"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"strings"
)

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
	if !mcpRuntimePlaceholderAllowsExcerpt(omittedReason) {
		return metadata
	}
	excerpt := mcpRuntimeBoundedRedactedExcerpt(content, mcpRuntimeResultExcerptMaxRunes)
	if excerpt == "" {
		return metadata
	}
	return metadata + "\nexcerpt:\n" + excerpt
}

func mcpRuntimePlaceholderAllowsExcerpt(omittedReason string) bool {
	return providerhistory.MCPRawOutputArtifactOmitReasonAllowsRuntimeExcerpt(omittedReason)
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
