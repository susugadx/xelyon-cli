package navigation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// readDefinitionBody は定義本文を読み出す。
func readDefinitionBody(cand SymbolCandidate, maxLines int) []string {
	absPath := candidateAbsPath(cand)
	if absPath == "" {
		return nil
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	start := cand.Line - 1 // 0-indexed
	end := cand.EndLine    // exclusive

	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}

	totalLines := end - start
	truncated := totalLines > maxLines
	if truncated {
		end = start + maxLines
	}

	var result []string
	for i := start; i < end; i++ {
		result = append(result, fmt.Sprintf("%d: %s", i+1, lines[i]))
	}
	if truncated {
		result = append(result, fmt.Sprintf("... (%d more lines, L%d-L%d)", totalLines-maxLines, start+maxLines+1, cand.EndLine))
	}
	return result
}

func candidateAbsPath(cand SymbolCandidate) string {
	if strings.TrimSpace(cand.File) == "" {
		return ""
	}
	if filepath.IsAbs(cand.File) {
		return filepath.Clean(cand.File)
	}
	if strings.TrimSpace(cand.RootPath) != "" {
		return filepath.Join(cand.RootPath, filepath.FromSlash(cand.File))
	}
	absPath, err := filepath.Abs(cand.File)
	if err != nil {
		return ""
	}
	return absPath
}
