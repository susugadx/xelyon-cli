package ui

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

// ToolDisplayInfo はツール実行結果の表示情報を保持する。
type ToolDisplayInfo struct {
	ToolName string
	Args     map[string]string
	Result   string
	Error    bool
}

var (
	searchSummaryPattern            = regexp.MustCompile(`Found (\d+) match(?:\(es\)|es?) in (\d+) file(?:\(s\)|s?)`)
	searchFileHeaderPattern         = regexp.MustCompile(`^📄 (.+?) \(\d+ match(?:\(es\)|es?)\)(?: .+)?$`)
	searchSymbolHeaderPattern       = regexp.MustCompile(`^── .+ \(L(\d+)(?:-L?\d+)?\) in (.+?)(?: @\S+)? ──$`)
	searchBundleItemPattern         = regexp.MustCompile(`^- (.+?):(\d+)(?:\s|\||$)`)
	searchFormattedMatchLinePattern = regexp.MustCompile(`^(?:\[[^]]+\]\s+)?(?:>\s*)?(\d+)\s*│`)
	outlineSummaryPattern           = regexp.MustCompile(`\((\d+) lines total(?:\.[^)]*)?\)`)
	strReplaceRangePattern          = regexp.MustCompile(`lines (\d+)-(\d+)`)
	strReplaceEditsPattern          = regexp.MustCompile(`Successfully applied (\d+) edits`)
	writeFileLinesPattern           = regexp.MustCompile(`Successfully wrote \d+ bytes \((\d+) lines\) to `)
	leadingLineNumPattern           = regexp.MustCompile(`^(\d+): `)
)

// FormatToolLine はツール実行の1行サマリーを返す。
func FormatToolLine(info ToolDisplayInfo) string {
	trimmed := strings.TrimSpace(info.Result)
	blueName := Blue.Sprint(info.ToolName)
	if isToolDisplayError(info, trimmed) {
		target := toolTarget(info)
		if target == "" {
			return fmt.Sprintf("❌ %s → %s", blueName, firstLine(trimmed))
		}
		return fmt.Sprintf("❌ %s: %s → %s", blueName, target, firstLine(trimmed))
	}

	summary := formatToolSummary(info, trimmed)
	if summary == "" {
		return fmt.Sprintf("%s %s", toolIcon(info.ToolName), blueName)
	}
	return fmt.Sprintf("%s %s: %s", toolIcon(info.ToolName), blueName, summary)
}

// PrintParallelGroupStartToWriter は並列実行グループの開始行を指定 writer に表示する。
func PrintParallelGroupStartToWriter(w io.Writer, count int) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, "┌ Parallel (%d calls)\n", count)
}

// PrintParallelGroupLineToWriter は並列実行グループ内の1行を指定 writer に表示する。
func PrintParallelGroupLineToWriter(w io.Writer, line string) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, "│ %s\n", line)
}

// PrintParallelGroupEndToWriter は並列実行グループの終了行を指定 writer に表示する。
func PrintParallelGroupEndToWriter(w io.Writer, summary string) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, "└ %s\n", summary)
}

// FormatParallelElapsed は並列実行の経過時間を短い文字列で返す。
func FormatParallelElapsed(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < 10*time.Second:
		return fmt.Sprintf("%.1fs", d.Seconds())
	default:
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
}
