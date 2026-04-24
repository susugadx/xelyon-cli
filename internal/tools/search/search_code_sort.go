package search

import (
	"sort"

	"github.com/susugadx/xelyon-cli/internal/filefilter"
	"github.com/susugadx/xelyon-cli/internal/repomap"
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
	return filefilter.MatchesParts(
		filefilter.MatchPathWithWorkspace(filePath, opts.Path, resolveSearchWorkspaceRoot(opts)),
		opts.FileType,
		opts.FilePattern,
	)
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

// isTestFile はファイルパスがテストファイルかどうか判定する。
// repomap.IsTestFile に委譲して全言語のテスト検出を一元化する。
func isTestFile(path string) bool {
	return repomap.IsTestFile(path)
}
