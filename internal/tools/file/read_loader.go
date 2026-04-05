package file

import (
	"fmt"
	"io"
	"os"
	"strings"
)

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

	lines, totalRead, hasMore, err := readFirstNLines(f, MaxReadLines)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err), true
	}

	totalLines := len(lines)
	if hasMore && totalRead > 0 {
		avgLineLen := ctx.fileInfo.Size() / int64(totalRead)
		if avgLineLen > 0 {
			totalLines = int(ctx.fileInfo.Size() / avgLineLen)
		}
	}

	if !hasMore && len(lines) <= ctx.outlineThreshold {
		result := formatLinesWithNumbers(lines, 1)
		if ctx.showFileInfo && ctx.fileSize > 0 {
			printReadStatus(ctx.out, "📄 Read: %s (%s, %d lines)\n", ctx.path, formatFileSize(ctx.fileSize), len(lines))
		} else {
			printReadStatus(ctx.out, "📄 Read: %s (%d lines)\n", ctx.path, len(lines))
		}
		return result, true
	}

	result := formatOutline(ctx.absPath, lines, totalLines)
	if ctx.showFileInfo && ctx.fileSize > 0 {
		printReadStatus(ctx.out, "📄 Read: %s (%s, outline of ~%d lines)\n", ctx.path, formatFileSize(ctx.fileSize), totalLines)
	} else {
		printReadStatus(ctx.out, "📄 Read: %s (outline of ~%d lines)\n", ctx.path, totalLines)
	}
	return result, true
}

func loadReadContent(ctx readFileContext, startLine, endLine int) (string, string) {
	if startLine == 0 && endLine == 0 && ctx.cache != nil {
		if cached, hit := ctx.cache.GetFile(ctx.absPath); hit {
			return cached, ""
		}
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
