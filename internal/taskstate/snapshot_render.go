package taskstate

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	CurrentTaskStateStartMarker = "<current_task_state>"
	CurrentTaskStateEndMarker   = "</current_task_state>"

	defaultSnapshotChangedFilesLimit     = 20
	defaultSnapshotTouchedFilesLimit     = 30
	defaultSnapshotEvidenceLimit         = 20
	defaultSnapshotRecommendedReadsLimit = 10
	defaultSnapshotFailedTestsLimit      = 3
	defaultSnapshotPassedTestsLimit      = 3
	defaultSnapshotExcerptRuneLimit      = 140
)

// SnapshotRenderOptions は CurrentTaskState snapshot の出力量を制御する。
type SnapshotRenderOptions struct {
	ChangedFilesLimit     int
	TouchedFilesLimit     int
	EvidenceLimit         int
	RecommendedReadsLimit int
	FailedTestsLimit      int
	PassedTestsLimit      int
	ExcerptRuneLimit      int
}

// DefaultSnapshotRenderOptions は model-facing snapshot renderer の既定値を返す。
func DefaultSnapshotRenderOptions() SnapshotRenderOptions {
	return SnapshotRenderOptions{
		ChangedFilesLimit:     defaultSnapshotChangedFilesLimit,
		TouchedFilesLimit:     defaultSnapshotTouchedFilesLimit,
		EvidenceLimit:         defaultSnapshotEvidenceLimit,
		RecommendedReadsLimit: defaultSnapshotRecommendedReadsLimit,
		FailedTestsLimit:      defaultSnapshotFailedTestsLimit,
		PassedTestsLimit:      defaultSnapshotPassedTestsLimit,
		ExcerptRuneLimit:      defaultSnapshotExcerptRuneLimit,
	}
}

// RenderCurrentTaskStateSnapshot は RuntimeTaskState を prompt-ready text に整形する。
func RenderCurrentTaskStateSnapshot(state RuntimeTaskState, opts SnapshotRenderOptions) string {
	opts = normalizeSnapshotRenderOptions(opts)

	var b strings.Builder
	b.WriteString(CurrentTaskStateStartMarker)
	b.WriteByte('\n')
	b.WriteString("CurrentTaskState:\n")

	if state.IsEmpty() {
		b.WriteString("- No runtime task facts recorded yet.\n")
		b.WriteString(CurrentTaskStateEndMarker)
		return b.String()
	}

	renderSnapshotPathSection(&b, "Changed files:", state.ChangedFiles.Paths(), opts.ChangedFilesLimit)
	renderSnapshotPathSection(&b, "Recently touched files:", state.TouchedFiles.Paths(), opts.TouchedFilesLimit)
	renderSnapshotEvidenceSection(&b, state.Evidence.Items(), opts.EvidenceLimit, opts.ExcerptRuneLimit)
	renderSnapshotRecommendedReadsSection(&b, state.RecommendedReads.Items(), opts.RecommendedReadsLimit, opts.ExcerptRuneLimit)
	renderSnapshotFailedTestsSection(&b, state.LastFailedTests.Results(), opts.FailedTestsLimit, opts.ExcerptRuneLimit)
	renderSnapshotPassedTestsSection(&b, state.LastPassedTests.Results(), opts.PassedTestsLimit, opts.ExcerptRuneLimit)

	b.WriteString(CurrentTaskStateEndMarker)
	return b.String()
}

func normalizeSnapshotRenderOptions(opts SnapshotRenderOptions) SnapshotRenderOptions {
	defaults := DefaultSnapshotRenderOptions()
	opts.ChangedFilesLimit = snapshotDefaultLimit(opts.ChangedFilesLimit, defaults.ChangedFilesLimit)
	opts.TouchedFilesLimit = snapshotDefaultLimit(opts.TouchedFilesLimit, defaults.TouchedFilesLimit)
	opts.EvidenceLimit = snapshotDefaultLimit(opts.EvidenceLimit, defaults.EvidenceLimit)
	opts.RecommendedReadsLimit = snapshotDefaultLimit(opts.RecommendedReadsLimit, defaults.RecommendedReadsLimit)
	opts.FailedTestsLimit = snapshotDefaultLimit(opts.FailedTestsLimit, defaults.FailedTestsLimit)
	opts.PassedTestsLimit = snapshotDefaultLimit(opts.PassedTestsLimit, defaults.PassedTestsLimit)
	opts.ExcerptRuneLimit = snapshotDefaultLimit(opts.ExcerptRuneLimit, defaults.ExcerptRuneLimit)
	return opts
}

func snapshotDefaultLimit(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func renderSnapshotPathSection(b *strings.Builder, title string, paths []string, limit int) {
	renderSnapshotSection(b, title, paths, limit, func(b *strings.Builder, path string) {
		fmt.Fprintf(b, "- %s\n", path)
	})
}

func renderSnapshotEvidenceSection(b *strings.Builder, items []evidenceFact, limit, excerptLimit int) {
	renderSnapshotSection(b, "Evidence pointers:", items, limit, func(b *strings.Builder, item evidenceFact) {
		fields := []string{fmt.Sprintf("%s:L%d-L%d", item.Path(), item.StartLine(), item.EndLine())}
		fields = appendSnapshotAttribute(fields, "source", item.Source())
		fields = appendSnapshotAttribute(fields, "id", item.ToolCallID())
		fields = appendSnapshotAttribute(fields, "file_hash", item.FileHash())
		if item.Stale() {
			fields = append(fields, "stale=true")
		}
		fields = append(fields, "excerpt="+quoteSnapshotText(item.Excerpt(), excerptLimit))
		fmt.Fprintf(b, "- %s\n", strings.Join(fields, " "))
	})
}

func renderSnapshotRecommendedReadsSection(b *strings.Builder, items []recommendedReadFact, limit, excerptLimit int) {
	renderSnapshotSection(b, "Recommended reads:", items, limit, func(b *strings.Builder, item recommendedReadFact) {
		fields := []string{item.Path()}
		fields = appendQuotedSnapshotAttribute(fields, "reason", item.Reason(), excerptLimit)
		fields = appendSnapshotAttribute(fields, "source", item.Source())
		fields = appendSnapshotAttribute(fields, "id", item.ToolCallID())
		fmt.Fprintf(b, "- %s\n", strings.Join(fields, " "))
	})
}

func renderSnapshotFailedTestsSection(b *strings.Builder, results []TestResult, limit, excerptLimit int) {
	renderSnapshotSection(b, "Last failed tests:", results, limit, func(b *strings.Builder, result TestResult) {
		fmt.Fprintf(b, "- failed: %s exit=%d excerpt=%s\n",
			snapshotSingleLine(result.Command()),
			result.ExitCode(),
			quoteSnapshotText(result.Excerpt(), excerptLimit),
		)
	})
}

func renderSnapshotPassedTestsSection(b *strings.Builder, results []TestResult, limit, excerptLimit int) {
	renderSnapshotSection(b, "Last passed tests:", results, limit, func(b *strings.Builder, result TestResult) {
		line := "- passed: " + snapshotSingleLine(result.Command())
		if excerpt := snapshotSingleLine(result.Excerpt()); excerpt != "" {
			line += " excerpt=" + quoteSnapshotText(excerpt, excerptLimit)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	})
}

func renderSnapshotSection[T any](b *strings.Builder, title string, items []T, limit int, renderItem func(*strings.Builder, T)) {
	if len(items) == 0 {
		return
	}
	b.WriteString(title)
	b.WriteByte('\n')
	visible := snapshotVisibleCount(len(items), limit)
	for _, item := range items[:visible] {
		renderItem(b, item)
	}
	renderSnapshotOmittedLine(b, len(items)-visible)
}

func appendSnapshotAttribute(fields []string, name, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fields
	}
	return append(fields, name+"="+value)
}

func appendQuotedSnapshotAttribute(fields []string, name, value string, runeLimit int) []string {
	value = snapshotSingleLine(value)
	if value == "" {
		return fields
	}
	return append(fields, name+"="+quoteSnapshotText(value, runeLimit))
}

func snapshotVisibleCount(total, limit int) int {
	if total < limit {
		return total
	}
	return limit
}

func renderSnapshotOmittedLine(b *strings.Builder, omitted int) {
	if omitted > 0 {
		fmt.Fprintf(b, "- ... %d more omitted\n", omitted)
	}
}

func quoteSnapshotText(value string, runeLimit int) string {
	return strconv.Quote(truncateSnapshotRunes(snapshotSingleLine(value), runeLimit))
}

func snapshotSingleLine(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func truncateSnapshotRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	if limit <= 3 {
		return "..."[:limit]
	}
	runes := []rune(value)
	return string(runes[:limit-3]) + "..."
}
