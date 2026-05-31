package providerhistory

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ledger"
)

func applyProviderHistoryReduction(report *ProjectionReport, projection []api.Message, policy Policy) {
	if report == nil || report.Mode != Apply || len(report.Candidates) == 0 {
		return
	}
	if policy.EvidenceReductionRequiresActiveContext && !policy.ActiveContextTransportAvailable {
		for i := range report.Candidates {
			keepReductionCandidate(report, i, "active_context_transport_unsupported")
		}
		return
	}

	pointers := policy.EvidencePointers
	evidenceKeyCounts := countProviderHistoryReductionEvidenceKeys(report.Candidates, report.Kept)
	for i := range report.Candidates {
		candidate := report.Candidates[i]
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

		replacementKind, replacementText := buildProviderHistoryReplacement(candidate, evidencePointers)
		report.Candidates[i].SuggestedReplacementKind = replacementKind
		report.Candidates[i].SuggestedReplacementText = replacementText
		if len(replacementText) >= candidate.OriginalByteSize {
			keepReductionCandidate(report, i, "replacement_not_smaller")
			continue
		}

		applyReductionCandidateProjection(&projection[candidate.HistoryIndex], candidate, replacementText)
		report.Candidates[i].ReplacementApplied = true
		report.Candidates[i].EvidencePointers = cloneProviderHistoryReductionEvidencePointers(evidencePointers)
	}
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
			key := providerHistoryReductionEvidenceKeyForCandidate(entry)
			if key.toolCallID == "" || key.toolName == "" {
				continue
			}
			counts[key]++
		}
	}
	return counts
}

func providerHistoryEvidencePointersForCandidate(pointers []ledger.EvidencePointer, candidate ReductionCandidate) []ledger.EvidencePointer {
	if len(pointers) == 0 {
		return nil
	}
	matched := make([]ledger.EvidencePointer, 0, len(pointers))
	for _, pointer := range pointers {
		if pointer.ToolCallID == candidate.ToolCallID && pointer.Source == candidate.ToolName {
			matched = append(matched, pointer)
		}
	}
	return matched
}

func buildProviderHistoryReplacement(candidate ReductionCandidate, evidencePointers []ledger.EvidencePointer) (string, string) {
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

func providerHistoryEvidencePointerSummary(evidencePointers []ledger.EvidencePointer) string {
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

func providerHistoryEvidencePointerLineRange(pointer ledger.EvidencePointer) string {
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
