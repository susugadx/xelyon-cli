package agent

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

const (
	ledgerCommandExcerptLimit       = 180
	ledgerCommandRehydratePlanLimit = 5
	ledgerRawOutputCandidateLimit   = 10
)

func renderLedgerCommandOutput(out io.Writer, state taskstate.RuntimeTaskState) {
	_, _ = fmt.Fprintln(out, "Runtime task ledger")
	_, _ = fmt.Fprintln(out)

	renderLedgerPathSection(out, "Changed files", state.ChangedFiles.Paths(), "No changed files recorded.")
	renderLedgerPathSection(out, "Touched files", state.TouchedFiles.Paths(), "No touched files recorded.")
	renderLedgerEvidenceSection(out, state.Evidence.Items())
	renderLedgerRecommendedReadsSection(out, state.RecommendedReads.Items())
	renderLedgerTestResultsSection(out, "Last failed tests", state.LastFailedTests.Results(), "No failed tests recorded.")
	renderLedgerTestResultsSection(out, "Last passed tests", state.LastPassedTests.Results(), "No passed tests recorded.")
}

func renderLedgerPathSection(out io.Writer, title string, paths []string, emptyMessage string) {
	_, _ = fmt.Fprintf(out, "%s:\n", title)
	if len(paths) == 0 {
		_, _ = fmt.Fprintf(out, "  %s\n\n", emptyMessage)
		return
	}
	for _, path := range paths {
		_, _ = fmt.Fprintf(out, "  - %s\n", path)
	}
	_, _ = fmt.Fprintln(out)
}

func renderLedgerEvidenceSection[T ledgerEvidenceItem](out io.Writer, items []T) {
	_, _ = fmt.Fprintln(out, "Evidence:")
	if len(items) == 0 {
		_, _ = fmt.Fprintln(out, "  No evidence recorded.")
		_, _ = fmt.Fprintln(out)
		return
	}
	for _, item := range items {
		_, _ = fmt.Fprintf(out, "  - %s\n", strings.Join(ledgerEvidenceFields(item), " | "))
	}
	_, _ = fmt.Fprintln(out)
}

func renderLedgerRecommendedReadsSection[T ledgerRecommendedReadItem](out io.Writer, items []T) {
	_, _ = fmt.Fprintln(out, "Recommended reads:")
	if len(items) == 0 {
		_, _ = fmt.Fprintln(out, "  No recommended reads recorded.")
		_, _ = fmt.Fprintln(out)
		return
	}
	for _, item := range items {
		_, _ = fmt.Fprintf(out, "  - %s\n", strings.Join(ledgerRecommendedReadFields(item), " | "))
	}
	_, _ = fmt.Fprintln(out)
}

func renderLedgerTestResultsSection(out io.Writer, title string, results []taskstate.TestResult, emptyMessage string) {
	_, _ = fmt.Fprintf(out, "%s:\n", title)
	if len(results) == 0 {
		_, _ = fmt.Fprintf(out, "  %s\n\n", emptyMessage)
		return
	}
	for _, result := range results {
		_, _ = fmt.Fprintf(out, "  - %s\n", strings.Join(ledgerTestResultFields(result), " | "))
	}
	_, _ = fmt.Fprintln(out)
}

func renderLedgerRehydratePlanSection(out io.Writer, plan taskstate.RehydratePlan) {
	if len(plan.Items) == 0 {
		return
	}
	_, _ = fmt.Fprintln(out, "Rehydrate candidates:")
	limit := len(plan.Items)
	if limit > ledgerCommandRehydratePlanLimit {
		limit = ledgerCommandRehydratePlanLimit
	}
	for _, item := range plan.Items[:limit] {
		_, _ = fmt.Fprintf(out, "  - %s:%s\n", item.Path, ledgerRehydratePlanLineRange(item))
		_, _ = fmt.Fprintf(out, "    source: %s | reason: %s | stale: %t\n",
			ledgerDisplayValue(item.Source, "unknown"),
			ledgerDisplayValue(item.Reason, "unknown"),
			item.Stale,
		)
	}
	if remaining := len(plan.Items) - limit; remaining > 0 {
		_, _ = fmt.Fprintf(out, "  ... +%d more\n", remaining)
	}
	_, _ = fmt.Fprintln(out)
}

func renderLedgerProviderHistoryRawOutputSection(out io.Writer, report ProviderHistoryProjectionReport) {
	candidates := ledgerRawOutputCandidates(report)
	if len(candidates) == 0 {
		return
	}
	_, _ = fmt.Fprintln(out, "Provider history raw output candidates:")
	limit := len(candidates)
	if limit > ledgerRawOutputCandidateLimit {
		limit = ledgerRawOutputCandidateLimit
	}
	for _, candidate := range candidates[:limit] {
		_, _ = fmt.Fprintf(out, "  - %s\n", strings.Join(candidate, " | "))
	}
	if remaining := len(candidates) - limit; remaining > 0 {
		_, _ = fmt.Fprintf(out, "  ... +%d more\n", remaining)
	}
	_, _ = fmt.Fprintln(out)
}

func ledgerRawOutputCandidates(report ProviderHistoryProjectionReport) [][]string {
	refs := ledgerRawOutputRefsByID(report.RawOutputRefs)
	candidates := make([][]string, 0)
	for _, candidate := range report.CommandEditDryRun.Candidates {
		if !candidate.ArtifactBackedCandidate && strings.TrimSpace(candidate.RawOutputRefID) == "" {
			continue
		}
		candidates = append(candidates, ledgerCommandRawOutputCandidateFields(candidate, refs))
	}
	for _, candidate := range report.Candidates {
		if !candidate.ArtifactBackedCandidate && strings.TrimSpace(candidate.RawOutputRefID) == "" {
			continue
		}
		candidates = append(candidates, ledgerToolRawOutputCandidateFields(candidate, refs))
	}
	return candidates
}

func ledgerRawOutputRefsByID(refs []rawoutputs.RawOutputRef) map[string]rawoutputs.RawOutputRef {
	byID := make(map[string]rawoutputs.RawOutputRef, len(refs))
	for _, ref := range refs {
		refID := strings.TrimSpace(ref.RefID)
		if refID == "" {
			continue
		}
		byID[refID] = ref
	}
	return byID
}

func ledgerCommandRawOutputCandidateFields(candidate ProviderHistoryCommandEditDryRunCandidate, refs map[string]rawoutputs.RawOutputRef) []string {
	ref := refs[strings.TrimSpace(candidate.RawOutputRefID)]
	fields := ledgerRawOutputBaseFields(
		"command_output",
		candidate.HistoryIndex,
		candidate.ToolName,
		candidate.ToolCallID,
		candidate.RawOutputRefID,
		ref,
	)
	fields = append(fields,
		"apply_state: "+ledgerCommandRawOutputApplyState(candidate),
		"artifact: "+ledgerDisplayValue(candidate.ArtifactGateStatus, "unknown"),
		"rehydrate: "+ledgerDisplayValue(candidate.RehydrateGateStatus, "unknown"),
		"threshold: "+ledgerDisplayValue(candidate.ThresholdStatus, "unknown"),
		"safety: "+ledgerDisplayValue(candidate.SafetyStatus, "unknown"),
		"reason: "+ledgerDisplayValue(ledgerFirstNonEmpty(candidate.FailClosedReason, candidate.KeepReason), "none"),
		fmt.Sprintf("estimated_saved: %d B/%d tok", candidate.EstimatedSavedBytes, candidate.ApproxEstimatedSavedTokens),
	)
	return fields
}

func ledgerToolRawOutputCandidateFields(candidate ProviderHistoryReductionCandidate, refs map[string]rawoutputs.RawOutputRef) []string {
	ref := refs[strings.TrimSpace(candidate.RawOutputRefID)]
	fields := ledgerRawOutputBaseFields(
		"tool_result",
		candidate.HistoryIndex,
		candidate.ToolName,
		candidate.ToolCallID,
		candidate.RawOutputRefID,
		ref,
	)
	fields = append(fields,
		"apply_state: "+ledgerToolRawOutputApplyState(candidate),
		"artifact: "+ledgerDisplayValue(candidate.ArtifactGateStatus, "unknown"),
		"rehydrate: "+ledgerDisplayValue(candidate.RehydrateGateStatus, "unknown"),
		"threshold: "+ledgerDisplayValue(candidate.ThresholdStatus, "unknown"),
		"safety: "+ledgerDisplayValue(candidate.SafetyStatus, "unknown"),
		"reason: "+ledgerDisplayValue(ledgerFirstNonEmpty(candidate.FailClosedReason, candidate.KeepReason), "none"),
		fmt.Sprintf("estimated_saved: %d B/%d tok", candidate.EstimatedSavedBytes, candidate.ApproxEstimatedSavedTokens),
		fmt.Sprintf("actual_saved: %d B/%d tok", candidate.ArtifactBackedActualSavedBytes, candidate.ApproxArtifactBackedActualSavedTokens),
	)
	return fields
}

func ledgerRawOutputBaseFields(sourceKind string, historyIndex int, toolName, toolCallID, refID string, ref rawoutputs.RawOutputRef) []string {
	fields := []string{
		"source_kind: " + sourceKind,
		fmt.Sprintf("history_index: %d", historyIndex),
		"tool: " + ledgerDisplayValue(toolName, "unknown"),
		"raw_output_ref: " + ledgerDisplayValue(refID, "missing"),
		"surface: " + ledgerDisplayValue(ref.Surface, "unknown"),
		"semantic_role: " + ledgerDisplayValue(ref.SemanticRole, "unknown"),
		"family: " + ledgerDisplayValue(ref.Family, "unknown"),
		"classifier: " + ledgerDisplayValue(ref.Classifier, "unknown"),
		fmt.Sprintf("bytes: %d", ref.ByteSize),
		fmt.Sprintf("approx_tokens: %d", ref.ApproxTokens),
		"sha256: " + ledgerRawOutputHashPrefix(ref.ContentHash),
	}
	if strings.TrimSpace(toolCallID) != "" {
		fields = append(fields, "tool_call_id: "+toolCallID)
	}
	return fields
}

func ledgerCommandRawOutputApplyState(candidate ProviderHistoryCommandEditDryRunCandidate) string {
	switch {
	case candidate.ReplacementApplied:
		return "applied"
	case candidate.ArtifactBackedApplyEligible:
		return "apply_eligible"
	case strings.TrimSpace(candidate.FailClosedReason) != "" || strings.TrimSpace(candidate.KeepReason) != "":
		return "kept_raw"
	default:
		return "dry_run"
	}
}

func ledgerToolRawOutputApplyState(candidate ProviderHistoryReductionCandidate) string {
	switch {
	case candidate.ReplacementApplied:
		return "applied"
	case candidate.ArtifactBackedApplyEligible:
		return "apply_eligible"
	case candidate.CandidateOnly:
		return "candidate_only"
	case strings.TrimSpace(candidate.FailClosedReason) != "" || strings.TrimSpace(candidate.KeepReason) != "":
		return "kept_raw"
	default:
		return "dry_run"
	}
}

func ledgerRawOutputHashPrefix(hash string) string {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return "unknown"
	}
	hash = strings.TrimPrefix(hash, "sha256:")
	if len(hash) > 12 {
		hash = hash[:12]
	}
	return hash
}

func ledgerFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func ledgerEvidenceFields(item ledgerEvidenceItem) []string {
	parts := []string{
		fmt.Sprintf("%s:L%d-L%d", item.Path(), item.StartLine(), item.EndLine()),
		fmt.Sprintf("source: %s", ledgerDisplayValue(item.Source(), "unknown")),
	}
	if toolCallID := strings.TrimSpace(item.ToolCallID()); toolCallID != "" {
		parts = append(parts, "tool_call_id: "+toolCallID)
	}
	if item.Stale() {
		parts = append(parts, "stale: true")
	}
	if fileHash := strings.TrimSpace(item.FileHash()); fileHash != "" {
		parts = append(parts, "file_hash: "+fileHash)
	}
	return append(parts, "excerpt: "+ledgerSingleLineExcerpt(item.Excerpt()))
}

func ledgerRecommendedReadFields(item ledgerRecommendedReadItem) []string {
	parts := []string{
		item.Path(),
		"reason: " + ledgerDisplayValue(item.Reason(), "none"),
	}
	if source := strings.TrimSpace(item.Source()); source != "" {
		parts = append(parts, "source: "+source)
	}
	if toolCallID := strings.TrimSpace(item.ToolCallID()); toolCallID != "" {
		parts = append(parts, "tool_call_id: "+toolCallID)
	}
	return parts
}

func ledgerTestResultFields(result taskstate.TestResult) []string {
	return []string{
		"command: " + result.Command(),
		"status: " + ledgerDisplayValue(result.Status(), "unknown"),
		fmt.Sprintf("exit code: %d", result.ExitCode()),
		"excerpt: " + ledgerSingleLineExcerpt(result.Excerpt()),
	}
}

func ledgerDisplayValue(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func ledgerSingleLineExcerpt(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return "(empty)"
	}
	if utf8.RuneCountInString(value) <= ledgerCommandExcerptLimit {
		return value
	}
	runes := []rune(value)
	return string(runes[:ledgerCommandExcerptLimit-3]) + "..."
}

func ledgerRehydratePlanLineRange(item taskstate.RehydratePlanItem) string {
	if item.EndLine > item.StartLine {
		return fmt.Sprintf("L%d-L%d", item.StartLine, item.EndLine)
	}
	return fmt.Sprintf("L%d", item.StartLine)
}

type ledgerEvidenceItem interface {
	Path() string
	StartLine() int
	EndLine() int
	Source() string
	ToolCallID() string
	FileHash() string
	Stale() bool
	Excerpt() string
}

type ledgerRecommendedReadItem interface {
	Path() string
	Reason() string
	Source() string
	ToolCallID() string
}
