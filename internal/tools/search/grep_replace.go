package search

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// Colors from common package
var (
	green  = common.Green
	yellow = common.Yellow
	red    = common.Red
)

// GrepReplaceResult is the result of bulk replace
type GrepReplaceResult struct {
	FilesModified int
	TotalMatches  int
	FileResults   []FileReplaceResult
}

// FileReplaceResult is the result of single file replacement
type FileReplaceResult struct {
	FilePath   string
	MatchCount int
}

// ExecuteGrepReplace executes bulk replace across multiple files
// pattern: search pattern (regex)
// replacement: replacement string
// path: target directory (default: ".")
// filePattern: file name pattern (e.g., "*.go", default: all files)
func ExecuteGrepReplace(pattern, replacement, path, filePattern string) (string, []FileReplaceResult, error) {
	if pattern == "" {
		return "", nil, fmt.Errorf("pattern is required")
	}
	// Note: replacement == "" is allowed (acts as deletion)

	if path == "" {
		path = "."
	}
	if filePattern == "" {
		filePattern = "*"
	}

	// Compile regex
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", nil, fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Collect target files
	var targetFiles []string
	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors and continue
		}

		// Skip directories
		if info.IsDir() {
			// Skip .git, node_modules, vendor, etc.
			base := filepath.Base(filePath)
			if base == ".git" || base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check file pattern match
		matched, _ := filepath.Match(filePattern, filepath.Base(filePath))
		if !matched && filePattern != "*" {
			return nil
		}

		// Only target code files
		ext := strings.ToLower(filepath.Ext(filePath))
		codeExts := map[string]bool{
			".go": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
			".py": true, ".rb": true, ".java": true, ".c": true, ".cpp": true,
			".h": true, ".hpp": true, ".rs": true, ".swift": true, ".kt": true,
			".md": true, ".json": true, ".yaml": true, ".yml": true, ".toml": true,
			".html": true, ".css": true, ".scss": true, ".sql": true, ".sh": true,
			".txt": true, ".xml": true, ".vue": true, ".svelte": true,
		}
		if !codeExts[ext] && filePattern == "*" {
			return nil
		}

		targetFiles = append(targetFiles, filePath)
		return nil
	})

	if err != nil {
		return "", nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(targetFiles) == 0 {
		return "No files found matching the criteria", nil, nil
	}

	// Process each file
	var results []FileReplaceResult
	totalMatches := 0
	skippedUnread := 0

	for _, filePath := range targetFiles {
		// Read-Before-Write guard: 未読ファイルはスキップ
		absFilePath, absErr := filepath.Abs(filePath)
		if absErr != nil {
			continue
		}
		if !tools.GlobalReadTracker.IsRead(absFilePath) {
			skippedUnread++
			continue
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			continue // Skip read errors
		}

		// Count matches
		matches := re.FindAllStringIndex(string(content), -1)
		if len(matches) == 0 {
			continue
		}

		totalMatches += len(matches)

		// Execute replacement
		newContent := re.ReplaceAllString(string(content), replacement)

		// Write to file
		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			red.Printf("Error: Failed to write %s: %v\n", filePath, err)
			continue
		}

		results = append(results, FileReplaceResult{
			FilePath:   filePath,
			MatchCount: len(matches),
		})
	}

	// Generate result summary
	var sb strings.Builder

	sb.WriteString("✅ Replacement completed:\n")

	sb.WriteString(fmt.Sprintf("  Pattern: %s\n", pattern))
	sb.WriteString(fmt.Sprintf("  Replacement: %s\n", replacement))
	sb.WriteString(fmt.Sprintf("  Files: %d\n", len(results)))
	sb.WriteString(fmt.Sprintf("  Total matches: %d\n", totalMatches))
	sb.WriteString("\n")

	// Per-file results (max 20)
	displayCount := len(results)
	if displayCount > 20 {
		displayCount = 20
	}

	for i := 0; i < displayCount; i++ {
		r := results[i]
		sb.WriteString(fmt.Sprintf("  %s (%d matches)\n", r.FilePath, r.MatchCount))
	}

	if len(results) > 20 {
		sb.WriteString(fmt.Sprintf("  ... and %d more files\n", len(results)-20))
	}

	if skippedUnread > 0 {
		sb.WriteString(fmt.Sprintf("\n  Skipped %d file(s) not previously read with read_file.\n", skippedUnread))
	}

	return sb.String(), results, nil
}
