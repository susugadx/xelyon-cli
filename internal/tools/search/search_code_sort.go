package search

import (
	"path/filepath"
	"sort"
	"strings"
)

func filterResultsByOptions(results []SearchResult, opts SearchOptions) []SearchResult {
	if opts.FileType == "" && opts.FilePattern == "" && opts.ignoreMatcher == nil {
		return results
	}

	var filtered []SearchResult
	for _, result := range results {
		if opts.ignoreMatcher != nil && opts.ignoreMatcher.Match(result.FilePath, false) {
			continue
		}
		if matchesSearchFileFilter(result.FilePath, opts) {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

func matchesSearchFileFilter(filePath string, opts SearchOptions) bool {
	glob := opts.FilePattern
	if opts.FileType != "" {
		if typeGlob, ok := fileTypeToGlob(opts.FileType); ok {
			glob = typeGlob
		}
	}
	if glob == "" {
		return true
	}

	cleanPath := filepath.ToSlash(filePath)
	base := filepath.Base(cleanPath)
	if matched, err := filepath.Match(glob, base); err == nil && matched {
		return true
	}
	if matched, err := filepath.Match(glob, cleanPath); err == nil && matched {
		return true
	}
	return false
}

func fileTypeToGlob(fileType string) (string, bool) {
	typeToGlob := map[string]string{
		"go":    "*.go",
		"py":    "*.py",
		"js":    "*.js",
		"ts":    "*.ts",
		"rust":  "*.rs",
		"java":  "*.java",
		"c":     "*.c",
		"cpp":   "*.cpp",
		"rb":    "*.rb",
		"php":   "*.php",
		"swift": "*.swift",
	}
	glob, ok := typeToGlob[fileType]
	return glob, ok
}

func mergeContextLines(results []SearchResult) []SearchResult {
	merged := make([]SearchResult, 0, len(results))
	for _, r := range results {
		if len(r.Matches) == 0 {
			continue
		}

		seen := make(map[int]int)
		var deduped []Match
		for _, m := range r.Matches {
			if idx, exists := seen[m.LineNum]; exists {
				if m.IsMatch {
					deduped[idx] = m
				}
				continue
			}
			seen[m.LineNum] = len(deduped)
			deduped = append(deduped, m)
		}

		merged = append(merged, SearchResult{
			FilePath:   r.FilePath,
			Matches:    deduped,
			MatchCount: r.MatchCount,
		})
	}
	return merged
}

// adaptiveContextTrim はファイルごとのマッチ数に応じてコンテキスト行を削る
// 5マッチ以下: そのまま、6-15マッチ: 前後1行のみ、16マッチ以上: コンテキスト行なし
func adaptiveContextTrim(results []SearchResult) []SearchResult {
	for i, r := range results {
		matchCount := 0
		for _, m := range r.Matches {
			if m.IsMatch {
				matchCount++
			}
		}
		if matchCount <= 5 {
			continue
		}

		maxCtx := 1
		if matchCount > 15 {
			maxCtx = 0
		}

		var trimmed []Match
		for _, m := range r.Matches {
			if m.IsMatch {
				trimmed = append(trimmed, m)
				continue
			}
			if maxCtx == 0 {
				continue
			}
			nearMatch := false
			for k := range r.Matches {
				if r.Matches[k].IsMatch && absInt(m.LineNum-r.Matches[k].LineNum) <= maxCtx {
					nearMatch = true
					break
				}
			}
			if nearMatch {
				trimmed = append(trimmed, m)
			}
		}
		results[i].Matches = trimmed
	}
	return results
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// sortResultsByPriority はファイルを重要度順にソート（非テスト→テスト、定義あり→なし）
func sortResultsByPriority(results []SearchResult) {
	sort.SliceStable(results, func(i, j int) bool {
		iTest := isTestFile(results[i].FilePath)
		jTest := isTestFile(results[j].FilePath)
		if iTest != jTest {
			return !iTest
		}
		iHasDef := hasDefinitionMatch(results[i])
		jHasDef := hasDefinitionMatch(results[j])
		if iHasDef != jHasDef {
			return iHasDef
		}
		return false
	})
}

// hasDefinitionMatch はファイルに定義マッチが含まれるか判定する
func hasDefinitionMatch(r SearchResult) bool {
	for _, m := range r.Matches {
		if m.IsMatch && m.Type == MatchTypeDefinition {
			return true
		}
	}
	return false
}

// isTestFile はファイルパスがテストファイルかどうか判定する
func isTestFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".test.js") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".spec.js") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.Contains(base, "test_")
}
