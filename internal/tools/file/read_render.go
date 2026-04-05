package file

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func printReadStatus(out common.Output, format string, args ...interface{}) {
	if out.SuppressStdout() {
		return
	}
	out.Green.Printf(format, args...)
}

// formatFileSize はバイト数を人間が読みやすい形式に変換
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func renderReadResult(ctx readFileContext, contentStr string, startLine, endLine int) string {
	lines := splitNormalizedReadLines(contentStr)
	totalLines := len(lines)

	if startLine > 0 || endLine > 0 {
		window, errResult := resolveReadLineRange(totalLines, startLine, endLine)
		if errResult != "" {
			return errResult
		}

		result := formatLinesWithNumbers(lines[window.startLine-1:window.endLine], window.startLine)
		if ctx.showFileInfo && ctx.fileSize > 0 {
			printReadStatus(ctx.out, "📄 Read: %s (%s, lines %d-%d of %d)\n", ctx.path, formatFileSize(ctx.fileSize), window.startLine, window.endLine, totalLines)
		} else {
			printReadStatus(ctx.out, "📄 Read: %s (lines %d-%d of %d)\n", ctx.path, window.startLine, window.endLine, totalLines)
		}
		return result
	}

	if totalLines <= ctx.outlineThreshold {
		result := formatLinesWithNumbers(lines, 1)
		if ctx.showFileInfo && ctx.fileSize > 0 {
			printReadStatus(ctx.out, "📄 Read: %s (%s, %d lines)\n", ctx.path, formatFileSize(ctx.fileSize), totalLines)
		} else {
			printReadStatus(ctx.out, "📄 Read: %s (%d lines)\n", ctx.path, totalLines)
		}
		return result
	}

	result := formatOutline(ctx.absPath, lines, totalLines)
	if ctx.showFileInfo && ctx.fileSize > 0 {
		printReadStatus(ctx.out, "📄 Read: %s (%s, outline of %d lines)\n", ctx.path, formatFileSize(ctx.fileSize), totalLines)
	} else {
		printReadStatus(ctx.out, "📄 Read: %s (outline of %d lines)\n", ctx.path, totalLines)
	}
	return result
}

// outlineHeadLines はアウトラインモードで表示する先頭行数
const outlineHeadLines = 30

// outlineTailLines はアウトラインモードで表示する末尾行数
const outlineTailLines = 10

// formatOutline はファイルのアウトラインを生成する。
// 先頭30行 + 関数/メソッドシグネチャ一覧 + 末尾10行 を返す。
func formatOutline(filePath string, lines []string, totalLines int) string {
	var sb strings.Builder

	headEnd := outlineHeadLines
	if headEnd > totalLines {
		headEnd = totalLines
	}
	sb.WriteString(formatCappedLinesWithNumbers(lines[:headEnd], 1, previewMaxLineBytes))

	content := strings.Join(lines, "\n")
	isBrace := common.IsBraceLanguage(filePath)
	blocks := common.BuildBlockMap(content, isBrace)

	var signatures []string
	for _, b := range blocks {
		if b.StartLine > headEnd && b.StartLine <= totalLines-outlineTailLines {
			signatures = append(signatures, fmt.Sprintf("  L%-4d %s", b.StartLine, b.Name))
		}
	}

	if len(signatures) > 0 {
		sb.WriteString("\n--- Signatures ---\n")
		for _, sig := range signatures {
			sb.WriteString(sig)
			sb.WriteString("\n")
		}
	}

	tailStart := totalLines - outlineTailLines
	if tailStart < headEnd {
		tailStart = headEnd
	}
	if tailStart < totalLines && tailStart < len(lines) {
		sb.WriteString("\n--- Last lines ---\n")
		sb.WriteString(formatCappedLinesWithNumbers(lines[tailStart:], tailStart+1, previewMaxLineBytes))
	}

	fmt.Fprintf(&sb, "\n(%d lines total. For specific sections: paths=[%q])\n", totalLines, filePath+":start-end")
	return sb.String()
}

func formatSampledOutline(filePath string, headLines, tailLines []string, totalLines int) string {
	var sb strings.Builder

	headEnd := outlineHeadLines
	if headEnd > len(headLines) {
		headEnd = len(headLines)
	}
	if headEnd > totalLines {
		headEnd = totalLines
	}
	if headEnd > 0 {
		sb.WriteString(formatLinesWithNumbers(headLines[:headEnd], 1))
	}

	content := strings.Join(headLines, "\n")
	isBrace := common.IsBraceLanguage(filePath)
	blocks := common.BuildBlockMap(content, isBrace)

	var signatures []string
	for _, b := range blocks {
		if b.StartLine > headEnd && b.StartLine <= len(headLines) {
			signatures = append(signatures, fmt.Sprintf("  L%-4d %s", b.StartLine, b.Name))
		}
	}

	if len(signatures) > 0 {
		sb.WriteString("\n--- Signatures ---\n")
		for _, sig := range signatures {
			sb.WriteString(sig)
			sb.WriteString("\n")
		}
	}

	if len(tailLines) > 0 {
		tailStart := totalLines - len(tailLines) + 1
		if tailStart <= headEnd {
			skip := headEnd - tailStart + 1
			if skip < len(tailLines) {
				tailLines = tailLines[skip:]
				tailStart = headEnd + 1
			} else {
				tailLines = nil
			}
		}
		if len(tailLines) > 0 {
			sb.WriteString("\n--- Last lines ---\n")
			sb.WriteString(formatLinesWithNumbers(tailLines, tailStart))
		}
	}

	fmt.Fprintf(&sb, "\n(%d lines total. For specific sections: paths=[%q])\n", totalLines, filePath+":start-end")
	return sb.String()
}

// formatLinesWithNumbers は行番号付きでフォーマット
func formatLinesWithNumbers(lines []string, startNum int) string {
	var sb strings.Builder
	for i, line := range lines {
		fmt.Fprintf(&sb, "%d: %s\n", startNum+i, line)
	}
	return sb.String()
}

func formatCappedLinesWithNumbers(lines []string, startNum, maxLineBytes int) string {
	var sb strings.Builder
	for i, line := range lines {
		if maxLineBytes > 0 && len(line) > maxLineBytes {
			line = line[:maxLineBytes] + "..."
		}
		fmt.Fprintf(&sb, "%d: %s\n", startNum+i, line)
	}
	return sb.String()
}
