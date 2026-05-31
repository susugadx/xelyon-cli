package agent

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

const (
	ledgerCommandExcerptLimit       = 180
	ledgerCommandRehydratePlanLimit = 5
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
