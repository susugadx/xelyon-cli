package providerhistory

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/providerhistory/toolresults"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

func applyProviderHistoryReduction(report *ProjectionReport, original []api.Message, projection []api.Message, policy Policy) {
	if report == nil || report.Mode != Apply || len(report.Candidates) == 0 {
		return
	}

	pointers := policy.EvidencePointers
	evidenceKeyCounts := countProviderHistoryReductionEvidenceKeys(report.Candidates, report.Kept)
	for i := range report.Candidates {
		candidate := report.Candidates[i]
		if candidate.KeepReason != "" {
			continue
		}
		if candidate.CandidateOnly {
			continue
		}
		if candidate.ArtifactBackedCandidate {
			applyProviderHistoryRawOutputReductionCandidate(report, policy, i, projection)
			continue
		}
		if isStructuredToolResultReductionTool(candidate.ToolName) {
			applyStructuredToolResultReductionCandidate(report, i, original, projection)
			continue
		}
		if !isEvidenceBackedReductionTool(candidate.ToolName) {
			keepReductionCandidate(report, i, "tool_not_in_reduction_allowlist")
			continue
		}
		if policy.EvidenceReductionRequiresActiveContext && !policy.ActiveContextTransportAvailable {
			keepReductionCandidate(report, i, "active_context_transport_unsupported")
			continue
		}
		key := providerHistoryReductionEvidenceKeyForCandidate(candidate)
		if evidenceKeyCounts[key] > 1 {
			keepReductionCandidate(report, i, "ambiguous_evidence_pointer")
			continue
		}

		evidencePointers := providerHistoryEvidencePointersForCandidate(pointers, candidate)
		if len(evidencePointers) == 0 {
			keepReductionCandidate(report, i, "missing_evidence_pointer")
			continue
		}

		if candidate.HistoryIndex < 0 || candidate.HistoryIndex >= len(projection) {
			keepReductionCandidate(report, i, "missing_projection_message")
			continue
		}
		if !providerHistoryProjectionMessageMatchesCandidate(projection[candidate.HistoryIndex], candidate) {
			keepReductionCandidate(report, i, "mismatched_projection_message")
			continue
		}
		if candidate.HistoryIndex >= len(original) {
			keepReductionCandidate(report, i, "missing_original_message")
			continue
		}

		replacementKind, replacementText := buildProviderHistoryReplacement(candidate, evidencePointers)
		report.Candidates[i].SuggestedReplacementKind = replacementKind
		report.Candidates[i].SuggestedReplacementText = replacementText
		_, _, keepReason, ok := providerHistoryContentReplacementEligibility(original[candidate.HistoryIndex].Content, replacementText)
		if !ok {
			if keepReason != "" {
				keepReductionCandidate(report, i, keepReason)
			} else {
				keepReductionCandidate(report, i, "replacement_not_smaller")
			}
			continue
		}

		applyReductionCandidateProjection(&projection[candidate.HistoryIndex], candidate, replacementText)
		report.Candidates[i].ReplacementApplied = true
		report.Candidates[i].EvidencePointers = cloneProviderHistoryReductionEvidencePointers(evidencePointers)
	}
}

func applyProviderHistoryRawOutputReductionCandidate(report *ProjectionReport, policy Policy, candidateIndex int, projection []api.Message) {
	if report == nil || candidateIndex < 0 || candidateIndex >= len(report.Candidates) {
		return
	}
	candidate := report.Candidates[candidateIndex]
	if !candidate.ArtifactBackedApplyEligible || candidate.SuggestedReplacementText == "" {
		return
	}
	if candidate.HistoryIndex < 0 || candidate.HistoryIndex >= len(projection) {
		keepReductionCandidate(report, candidateIndex, "missing_projection_message")
		return
	}
	if !providerHistoryProjectionMessageMatchesCandidate(projection[candidate.HistoryIndex], candidate) {
		keepReductionCandidate(report, candidateIndex, "mismatched_projection_message")
		return
	}
	if !rawOutputArtifactMaterializationAllowed(policy) {
		reason := providerHistoryRawOutputMaterializationReadOnlyReason
		keepReductionCandidate(report, candidateIndex, reason)
		report.Candidates[candidateIndex].FailClosedReason = reason
		return
	}
	ref, reason, ok := providerHistoryRawOutputRefForCandidate(report.RawOutputRefs, candidate.RawOutputRefID)
	if !ok {
		keepReductionCandidate(report, candidateIndex, reason)
		report.Candidates[candidateIndex].FailClosedReason = reason
		return
	}
	if policy.RawOutputArtifactStore == nil {
		reason := "raw_output_artifact_missing"
		keepReductionCandidate(report, candidateIndex, reason)
		report.Candidates[candidateIndex].FailClosedReason = reason
		return
	}
	verifyResult, err := policy.RawOutputArtifactStore.Verify(context.Background(), ref)
	if err != nil || !verifyResult.OK {
		reason := providerHistoryRawOutputVerifyFailureReason(verifyResult, err)
		keepReductionCandidate(report, candidateIndex, reason)
		report.Candidates[candidateIndex].FailClosedReason = reason
		return
	}
	savedBytes, savedTokens, thresholdStatus, thresholdOK := providerHistoryArtifactBackedReplacementEligibility(projection[candidate.HistoryIndex].Content, candidate.SuggestedReplacementText)
	if !thresholdOK {
		keepReductionCandidate(report, candidateIndex, thresholdStatus)
		report.Candidates[candidateIndex].FailClosedReason = thresholdStatus
		return
	}
	applyReductionCandidateProjection(&projection[candidate.HistoryIndex], candidate, candidate.SuggestedReplacementText)
	report.Candidates[candidateIndex].ReplacementApplied = true
	report.Candidates[candidateIndex].ArtifactBackedActualSavedBytes = savedBytes
	report.Candidates[candidateIndex].ApproxArtifactBackedActualSavedTokens = savedTokens
}

func applyStructuredToolResultReductionCandidate(report *ProjectionReport, candidateIndex int, original, projection []api.Message) {
	if report == nil || candidateIndex < 0 || candidateIndex >= len(report.Candidates) {
		return
	}
	candidate := report.Candidates[candidateIndex]
	if candidate.HistoryIndex < 0 || candidate.HistoryIndex >= len(original) {
		keepReductionCandidate(report, candidateIndex, "missing_original_message")
		return
	}
	if candidate.HistoryIndex >= len(projection) {
		keepReductionCandidate(report, candidateIndex, "missing_projection_message")
		return
	}
	if !providerHistoryProjectionMessageMatchesCandidate(projection[candidate.HistoryIndex], candidate) {
		keepReductionCandidate(report, candidateIndex, "mismatched_projection_message")
		return
	}

	msg := original[candidate.HistoryIndex]
	linkage := providerHistoryToolResultLinkageForIndex(original, candidate.HistoryIndex, msg)
	if linkage.KeepReason != "" {
		keepReductionCandidate(report, candidateIndex, linkage.KeepReason)
		return
	}
	replacement, reason, ok := toolresults.BuildStructuredReplacement(toolresults.NewReplacementRequestWithMessages(candidate.ToolName, linkage.Ref.arguments, msg.Content, candidate.ToolCallID, candidate.HistoryIndex, original))
	if !ok {
		keepReductionCandidate(report, candidateIndex, reason)
		return
	}
	report.Candidates[candidateIndex].SuggestedReplacementKind = replacement.Kind()
	report.Candidates[candidateIndex].SuggestedReplacementText = replacement.Text()
	_, _, keepReason, ok := providerHistoryContentReplacementEligibility(msg.Content, replacement.Text())
	if !ok {
		keepReductionCandidate(report, candidateIndex, keepReason)
		return
	}

	applyReductionCandidateProjection(&projection[candidate.HistoryIndex], candidate, replacement.Text())
	report.Candidates[candidateIndex].ReplacementApplied = true
}

func providerHistoryToolResultLinkageForIndex(messages []api.Message, historyIndex int, msg api.Message) providerHistoryToolResultLinkage {
	assistantToolCallsByID := collectProviderHistoryAssistantToolCalls(messages)
	return resolveProviderHistoryToolResultLinkage(
		messages,
		historyIndex,
		msg,
		assistantToolCallsByID,
		providerHistoryTrailingToolSuffixStart(messages),
		providerHistoryLatestToolResultIndex(messages),
	)
}

func providerHistoryProjectionMessageMatchesCandidate(msg api.Message, candidate ReductionCandidate) bool {
	return msg.Role == "tool" && msg.ToolCallID == candidate.ToolCallID
}

func applyReductionCandidateProjection(msg *api.Message, candidate ReductionCandidate, replacementText string) {
	if msg == nil {
		return
	}
	if msg.ToolName == "" {
		msg.ToolName = candidate.ToolName
	}
	msg.Content = replacementText
	msg.ReplaceOpenAIResponsesFunctionCallOutput(candidate.ToolCallID, replacementText)
}

func keepReductionCandidate(report *ProjectionReport, candidateIndex int, reason string) {
	report.Candidates[candidateIndex].KeepReason = reason
	report.Candidates[candidateIndex].ReplacementApplied = false
	report.Kept = append(report.Kept, report.Candidates[candidateIndex])
}

type providerHistoryReductionEvidenceKey struct {
	toolCallID string
	toolName   string
}

func providerHistoryReductionEvidenceKeyForCandidate(candidate ReductionCandidate) providerHistoryReductionEvidenceKey {
	return providerHistoryReductionEvidenceKey{
		toolCallID: candidate.ToolCallID,
		toolName:   candidate.ToolName,
	}
}

func countProviderHistoryReductionEvidenceKeys(entrySets ...[]ReductionCandidate) map[providerHistoryReductionEvidenceKey]int {
	counts := make(map[providerHistoryReductionEvidenceKey]int)
	for _, entries := range entrySets {
		for _, entry := range entries {
			if !isEvidenceBackedReductionTool(entry.ToolName) {
				continue
			}
			key := providerHistoryReductionEvidenceKeyForCandidate(entry)
			if key.toolCallID == "" || key.toolName == "" {
				continue
			}
			counts[key]++
		}
	}
	return counts
}

func providerHistoryEvidencePointersForCandidate(pointers []taskstate.EvidencePointer, candidate ReductionCandidate) []taskstate.EvidencePointer {
	if len(pointers) == 0 {
		return nil
	}
	matched := make([]taskstate.EvidencePointer, 0, len(pointers))
	for _, pointer := range pointers {
		if pointer.ToolCallID == candidate.ToolCallID && pointer.Source == candidate.ToolName {
			matched = append(matched, pointer)
		}
	}
	return matched
}

func buildProviderHistoryReplacement(candidate ReductionCandidate, evidencePointers []taskstate.EvidencePointer) (string, string) {
	toolName := providerHistoryReductionSingleLine(candidate.ToolName)
	replacementKind := providerHistoryReductionReplacementKind(toolName)
	return replacementKind, fmt.Sprintf(
		"[omitted old %s result; evidence: %s]",
		toolName,
		providerHistoryEvidencePointerSummary(evidencePointers),
	)
}

func providerHistoryReductionReplacementKind(toolName string) string {
	return fmt.Sprintf("omit_old_%s_result", providerHistoryReductionSingleLine(toolName))
}

func providerHistoryEvidencePointerSummary(evidencePointers []taskstate.EvidencePointer) string {
	const maxInlineEvidencePointers = 3
	if len(evidencePointers) == 0 {
		return "missing"
	}

	limit := len(evidencePointers)
	if limit > maxInlineEvidencePointers {
		limit = maxInlineEvidencePointers
	}
	parts := make([]string, 0, limit+1)
	for _, pointer := range evidencePointers[:limit] {
		parts = append(parts, fmt.Sprintf(
			"%s:%s source=%s",
			providerHistoryReductionSingleLine(pointer.Path),
			providerHistoryEvidencePointerLineRange(pointer),
			providerHistoryReductionSingleLine(pointer.Source),
		))
	}
	if remaining := len(evidencePointers) - limit; remaining > 0 {
		parts = append(parts, fmt.Sprintf("+%d more", remaining))
	}
	return strings.Join(parts, "; ")
}

func providerHistoryEvidencePointerLineRange(pointer taskstate.EvidencePointer) string {
	if pointer.EndLine > pointer.StartLine {
		return fmt.Sprintf("L%d-L%d", pointer.StartLine, pointer.EndLine)
	}
	return fmt.Sprintf("L%d", pointer.StartLine)
}

func providerHistoryReductionSingleLine(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(value)
	if value == "" {
		return "unknown"
	}
	return value
}
