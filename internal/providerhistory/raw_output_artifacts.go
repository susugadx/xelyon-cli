package providerhistory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/token"
)

const (
	providerHistoryRawOutputPlaceholderExcerptRunes       = 1200
	providerHistoryRawOutputArtifactMinSavedTokens        = 2048
	providerHistoryRawOutputArtifactReplacementMaxRatio   = 0.75
	providerHistoryRawOutputMaterializationFallbackReason = "raw_output_artifact_materialization_failed"
	providerHistoryRawOutputMaterializationReadOnlyReason = "raw_output_artifact_materialization_read_only"
	providerHistoryRawOutputVerifyFallbackReason          = "raw_output_artifact_missing"
)

func commandHash(command string) string {
	sum := sha256.Sum256([]byte(command))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func recordArtifactBackedKept(report *CommandEditDryRunReport, reason string) {
	if report == nil || reason == "" {
		return
	}
	if report.ArtifactBackedKeptReasonCounts == nil {
		report.ArtifactBackedKeptReasonCounts = make(map[string]int)
	}
	report.ArtifactBackedKeptReasonCounts[reason]++
}

func providerHistoryRawOutputMaterializationDeniedReason(policy Policy) string {
	if policy.RawOutputApplyDisabledReason != "" {
		return policy.RawOutputApplyDisabledReason
	}
	return providerHistoryRawOutputMaterializationReadOnlyReason
}

func buildProviderHistoryArtifactBackedCommandPlaceholder(ref rawoutputs.RawOutputRef, content string) string {
	return buildProviderHistoryRawOutputPlaceholder("data-bearing command output", ref, content)
}

func buildProviderHistoryRawOutputPlaceholder(label string, ref rawoutputs.RawOutputRef, content string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		label = "data-bearing tool result"
	}
	parts := []string{
		fmt.Sprintf("[compacted old %s;", label),
		fmt.Sprintf("raw_output_ref=%s;", ref.RefID),
		fmt.Sprintf("surface=%s;", ref.Surface),
		fmt.Sprintf("semantic_role=%s;", ref.SemanticRole),
		fmt.Sprintf("family=%s;", ref.Family),
		fmt.Sprintf("classifier=%s;", ref.Classifier),
		fmt.Sprintf("bytes=%d;", ref.ByteSize),
		fmt.Sprintf("sha256=%s]", providerHistoryRawOutputHashPrefix(ref.ContentHash)),
	}
	if strings.TrimSpace(ref.Subfamily) != "" {
		parts = append(parts[:5], append([]string{fmt.Sprintf("subfamily=%s;", ref.Subfamily)}, parts[5:]...)...)
	}
	metadata := strings.Join(parts, "\n ")
	excerpt := providerHistoryBoundedRawOutputExcerpt(content, providerHistoryRawOutputPlaceholderExcerptRunes)
	if excerpt == "" {
		return metadata
	}
	return metadata + "\nexcerpt:\n" + excerpt
}

func providerHistoryRawOutputHashPrefix(hash string) string {
	trimmed := strings.TrimSpace(hash)
	if strings.HasPrefix(trimmed, "sha256:") {
		value := strings.TrimPrefix(trimmed, "sha256:")
		if len(value) > 12 {
			value = value[:12]
		}
		return "sha256:" + value
	}
	if len(trimmed) > 12 {
		return trimmed[:12]
	}
	return trimmed
}

func providerHistoryBoundedRawOutputExcerpt(content string, maxRunes int) string {
	content = strings.TrimSpace(content)
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

func providerHistoryArtifactBackedReplacementEligibility(originalContent, replacementText string) (int, int, string, bool) {
	if replacementText == "" || len(replacementText) >= len(originalContent) {
		return 0, 0, "raw_output_placeholder_not_smaller", false
	}
	originalTokens := token.EstimateTokenCount(originalContent)
	replacementTokens := token.EstimateTokenCount(replacementText)
	savedTokens := clampProviderHistorySavedTokens(originalTokens, replacementTokens)
	if savedTokens < providerHistoryRawOutputArtifactMinSavedTokens {
		return len(originalContent) - len(replacementText), savedTokens, "raw_output_artifact_saved_tokens_below_threshold", false
	}
	if originalTokens > 0 {
		ratio := float64(replacementTokens) / float64(originalTokens)
		if ratio > providerHistoryRawOutputArtifactReplacementMaxRatio {
			return len(originalContent) - len(replacementText), savedTokens, "raw_output_artifact_replacement_ratio_too_high", false
		}
	}
	return len(originalContent) - len(replacementText), savedTokens, "threshold_passed", true
}

func providerHistoryRawOutputRefForCandidate(refs []rawoutputs.RawOutputRef, refID string) (rawoutputs.RawOutputRef, string, bool) {
	if strings.TrimSpace(refID) == "" {
		return rawoutputs.RawOutputRef{}, "raw_output_ref_missing", false
	}
	var found rawoutputs.RawOutputRef
	count := 0
	for _, ref := range refs {
		if ref.RefID != refID {
			continue
		}
		found = ref
		count++
	}
	switch count {
	case 0:
		return rawoutputs.RawOutputRef{}, "raw_output_ref_report_metadata_missing", false
	case 1:
		return found, "", true
	default:
		return rawoutputs.RawOutputRef{}, "raw_output_ref_report_metadata_duplicate", false
	}
}

func providerHistoryRawOutputCreateFailureReason(err error) string {
	reason := string(rawoutputs.ReasonOf(err))
	if reason != "" {
		return reason
	}
	return providerHistoryRawOutputMaterializationFallbackReason
}

func providerHistoryRawOutputVerifyFailureReason(result rawoutputs.VerifyResult, err error) string {
	reason := string(rawoutputs.ReasonOf(err))
	if reason != "" {
		return reason
	}
	reason = string(result.Reason)
	if reason != "" {
		return reason
	}
	return providerHistoryRawOutputVerifyFallbackReason
}
