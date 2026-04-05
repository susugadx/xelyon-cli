package file

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const previewMaxLineBytes = 4096

func statReadFile(absPath string, showFileInfo bool) (os.FileInfo, int64, string) {
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, 0, fmt.Sprintf("Error reading file: %v", err)
	}
	if !showFileInfo {
		return info, 0, ""
	}
	return info, info.Size(), ""
}

func maybeReadLargeFile(ctx readFileContext) (string, bool) {
	if ctx.fileInfo == nil || ctx.fileInfo.Size() <= LargeFileThreshold {
		return "", false
	}

	f, err := os.Open(ctx.absPath)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err), true
	}
	defer f.Close()

	header := make([]byte, 512)
	n, _ := f.Read(header)
	if strings.Contains(string(header[:n]), "\x00") {
		return binaryFileError(ctx.path), true
	}
	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Sprintf("Error reading file: %v", err), true
	}

	headLines, tailLines, totalLines, _, truncated, err := readOutlineSample(f, MaxReadLines, outlineTailLines, previewMaxLineBytes)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err), true
	}

	if !truncated && totalLines <= len(headLines) && len(headLines) <= ctx.outlineThreshold {
		result := formatLinesWithNumbers(headLines, 1)
		if ctx.showFileInfo && ctx.fileSize > 0 {
			printReadStatus(ctx.out, "📄 Read: %s (%s, %d lines)\n", ctx.path, formatFileSize(ctx.fileSize), len(headLines))
		} else {
			printReadStatus(ctx.out, "📄 Read: %s (%d lines)\n", ctx.path, len(headLines))
		}
		return result, true
	}

	result := formatSampledOutline(ctx.absPath, headLines, tailLines, totalLines)
	if ctx.showFileInfo && ctx.fileSize > 0 {
		printReadLargeFileOutlineStatus(ctx, totalLines, false)
	} else {
		printReadLargeFileOutlineStatus(ctx, totalLines, false)
	}
	return result, true
}

func loadReadContent(ctx readFileContext, startLine, endLine int) (string, string) {
	if cached, hit := getCachedReadContent(ctx); hit {
		return cached, ""
	}

	f, err := os.Open(ctx.absPath)
	if err != nil {
		return "", fmt.Sprintf("Error reading file: %v", err)
	}
	defer f.Close()

	header := make([]byte, 512)
	n, _ := f.Read(header)
	if strings.Contains(string(header[:n]), "\x00") {
		return "", binaryFileError(ctx.path)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return "", fmt.Sprintf("Error reading file: %v", err)
	}

	content, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Sprintf("Error reading file: %v", err)
	}
	contentStr := string(content)

	if startLine == 0 && endLine == 0 && ctx.cache != nil {
		ctx.cache.SetFile(ctx.absPath, contentStr)
	}
	return contentStr, ""
}

func getCachedReadContent(ctx readFileContext) (string, bool) {
	if ctx.cache == nil {
		return "", false
	}
	return ctx.cache.GetFile(ctx.absPath)
}

func loadReadRangeLines(ctx readFileContext, startLine, endLine int) ([]string, readLineRange, string) {
	if errResult := validateRequestedReadLineRange(startLine, endLine); errResult != "" {
		return nil, readLineRange{}, errResult
	}

	window := normalizeRequestedReadLineRange(startLine, endLine)

	f, err := os.Open(ctx.absPath)
	if err != nil {
		return nil, readLineRange{}, fmt.Sprintf("Error reading file: %v", err)
	}
	defer f.Close()

	header := make([]byte, 512)
	n, _ := f.Read(header)
	if strings.Contains(string(header[:n]), "\x00") {
		return nil, readLineRange{}, binaryFileError(ctx.path)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return nil, readLineRange{}, fmt.Sprintf("Error reading file: %v", err)
	}

	lines, totalRead, err := readWindowLines(f, window.startLine, window.endLine, previewMaxLineBytes)
	if err != nil {
		return nil, readLineRange{}, fmt.Sprintf("Error reading file: %v", err)
	}
	if totalRead < window.startLine {
		return nil, readLineRange{}, fmt.Sprintf("Error: start_line %d exceeds total lines %d", window.startLine, totalRead)
	}
	if totalRead < window.endLine {
		window.endLine = totalRead
	}

	return lines, window, ""
}

func executeStreamedReadRange(ctx readFileContext, startLine, endLine int) string {
	lines, window, errResult := loadReadRangeLines(ctx, startLine, endLine)
	if errResult != "" {
		return errResult
	}

	result := formatLinesWithNumbers(lines, window.startLine)
	if ctx.showFileInfo && ctx.fileSize > 0 {
		printReadStatus(ctx.out, "📄 Read: %s (%s, lines %d-%d)\n", ctx.path, formatFileSize(ctx.fileSize), window.startLine, window.endLine)
	} else {
		printReadStatus(ctx.out, "📄 Read: %s (lines %d-%d)\n", ctx.path, window.startLine, window.endLine)
	}
	return result
}

func executeCompactRangeFromContent(ctx readFileContext, contentStr string, startLine, endLine int) string {
	lines := splitNormalizedReadLines(contentStr)
	window, errResult := resolveReadLineRange(len(lines), startLine, endLine)
	if errResult != "" {
		return errResult
	}

	result := formatCappedLinesWithNumbers(lines[window.startLine-1:window.endLine], window.startLine, previewMaxLineBytes)
	if ctx.showFileInfo && ctx.fileSize > 0 {
		printReadStatus(ctx.out, "📄 Read: %s (%s, lines %d-%d of %d)\n", ctx.path, formatFileSize(ctx.fileSize), window.startLine, window.endLine, len(lines))
	} else {
		printReadStatus(ctx.out, "📄 Read: %s (lines %d-%d of %d)\n", ctx.path, window.startLine, window.endLine, len(lines))
	}
	return result
}

func splitNormalizedReadLines(contentStr string) []string {
	lines := strings.Split(contentStr, "\n")
	if len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(line, "\r")
	}
	return lines
}

func executeLargeFileOutline(ctx readFileContext) string {
	f, err := os.Open(ctx.absPath)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}
	defer f.Close()

	header := make([]byte, 512)
	n, _ := f.Read(header)
	if strings.Contains(string(header[:n]), "\x00") {
		return binaryFileError(ctx.path)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}

	headLines, tailLines, totalLines, _, _, err := readOutlineSample(f, MaxReadLines, outlineTailLines, previewMaxLineBytes)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}

	result := formatSampledOutline(ctx.absPath, headLines, tailLines, totalLines)
	if ctx.showFileInfo && ctx.fileSize > 0 {
		printReadLargeFileOutlineStatus(ctx, totalLines, false)
	} else {
		printReadLargeFileOutlineStatus(ctx, totalLines, false)
	}
	return result
}

func printReadLargeFileOutlineStatus(ctx readFileContext, totalLines int, approximate bool) {
	lineCount := fmt.Sprintf("%d", totalLines)
	if approximate {
		lineCount = "~" + lineCount
	}
	if ctx.showFileInfo && ctx.fileSize > 0 {
		printReadStatus(ctx.out, "📄 Read: %s (%s, outline of %s lines)\n", ctx.path, formatFileSize(ctx.fileSize), lineCount)
		return
	}
	printReadStatus(ctx.out, "📄 Read: %s (outline of %s lines)\n", ctx.path, lineCount)
}

func binaryFileError(path string) string {
	return fmt.Sprintf("Error: %s appears to be a binary file (contains null bytes). Use 'file %s' or 'xxd %s | head' for binary inspection.", path, path, path)
}

func isBinaryContent(content string) bool {
	checkLen := len(content)
	if checkLen > 512 {
		checkLen = 512
	}
	return strings.Contains(content[:checkLen], "\x00")
}
