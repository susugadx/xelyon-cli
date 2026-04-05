package file

import (
	"fmt"
	"strings"
)

func joinFailureResult(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, "\n")
}

func buildDeferredStrReplaceResult(status, mode, path, comment string) string {
	header := status + " str_replace"
	if mode != "" {
		header += " (" + mode + ")"
	}
	header += " not applied for " + path + "."
	if strings.TrimSpace(comment) == "" {
		return joinFailureResult(
			header,
			"Next: review with read_file before retrying; do not repeat the same replacement unchanged.",
		)
	}
	return joinFailureResult(
		header,
		"Comment: "+strings.TrimSpace(comment),
		"Next: review with read_file and retry only after user approval.",
	)
}

func buildCandidateSummary(lines []string, cands []lineRange, total int) string {
	if total <= 0 {
		return ""
	}
	shown := min(len(cands), maxFailureCandidatesToShow)
	var b strings.Builder
	fmt.Fprintf(&b, "Candidates: %d total", total)
	if shown > 0 && shown < total {
		fmt.Fprintf(&b, " (showing %d)", shown)
	}
	for i := 0; i < shown; i++ {
		c := cands[i]
		fmt.Fprintf(&b, "\n- lines %d-%d: %s", c.StartLine, c.EndLine, buildInlinePreview(lines, c.StartLine, c.EndLine, 1))
	}
	if total > shown {
		fmt.Fprintf(&b, "\n- ... %d more candidates", total-shown)
	}
	return b.String()
}

func buildHeadPreview(lines []string, limit int) string {
	if len(lines) == 0 || limit <= 0 {
		return ""
	}
	previewCount := min(limit, len(lines))
	parts := make([]string, 0, previewCount)
	for i := 0; i < previewCount; i++ {
		parts = append(parts, fmt.Sprintf("%d:%s", i+1, compactPreviewLine(lines[i])))
	}
	preview := "Preview: " + strings.Join(parts, " | ")
	if len(lines) > previewCount {
		preview += fmt.Sprintf(" | ... +%d more lines", len(lines)-previewCount)
	}
	return preview
}

func buildInlinePreview(lines []string, startLine, endLine, ctx int) string {
	if len(lines) == 0 {
		return ""
	}
	if ctx < 0 {
		ctx = 0
	}
	start := startLine - ctx
	if start < 1 {
		start = 1
	}
	end := endLine + ctx
	if end > len(lines) {
		end = len(lines)
	}
	parts := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		marker := ""
		if i >= startLine && i <= endLine {
			marker = "*"
		}
		parts = append(parts, fmt.Sprintf("%d%s:%s", i, marker, compactPreviewLine(lines[i-1])))
	}
	return strings.Join(parts, " | ")
}

func compactPreviewLine(line string) string {
	line = strings.ReplaceAll(line, "\t", " ")
	line = strings.TrimSpace(line)
	if line == "" {
		return "∅"
	}
	if len(line) <= failurePreviewLineWidth {
		return line
	}
	return line[:failurePreviewLineWidth-3] + "..."
}
