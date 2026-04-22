package search

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

func estimateTokens(line string) int {
	return utf8.RuneCountInString(line)/2 + len(line)/8 + 3
}

func truncateToTokenBudget(results []SearchResult, budget int, _ bool) ([]SearchResult, bool) {
	headerTokens := 10
	usedTokens := headerTokens
	truncated := false

	var kept []SearchResult
	for _, r := range results {
		fileHeader := len(r.FilePath)/4 + 5
		if usedTokens+fileHeader > budget {
			truncated = true
			break
		}
		usedTokens += fileHeader

		var keptMatches []Match
		matchCount := 0

		for _, m := range r.Matches {
			lineTokens := estimateTokens(m.Line)
			if m.IsMatch {
				lineTokens += 10
			}
			if usedTokens+lineTokens > budget {
				truncated = true
				break
			}
			usedTokens += lineTokens
			keptMatches = append(keptMatches, m)
			if m.IsMatch {
				matchCount++
			}
		}

		if len(keptMatches) > 0 {
			kept = append(kept, SearchResult{
				FilePath:   r.FilePath,
				Matches:    keptMatches,
				MatchCount: matchCount,
			})
		}

		if truncated {
			break
		}
	}

	return kept, truncated
}

func formatSearchResults(results []SearchResult, truncated bool, tokenBudget int, reg *locator.Registry) string {
	return formatSearchResultsWithOptions(results, truncated, tokenBudget, reg, SearchOptions{})
}

func formatSearchResultsWithOptions(results []SearchResult, truncated bool, tokenBudget int, reg *locator.Registry, opts SearchOptions) string {
	totalMatches := 0
	for _, r := range results {
		totalMatches += r.MatchCount
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d match(es) in %d file(s)\n", totalMatches, len(results))
	sb.WriteString(formatSearchResultsBody(results, truncated, tokenBudget, reg, opts))
	return sb.String()
}

// formatSearchResultsBody はファイルごとの結果部分のみをフォーマットする
// formatSearchResults と formatMultiResults の両方から呼ばれる
func formatSearchResultsBody(results []SearchResult, truncated bool, tokenBudget int, reg *locator.Registry, opts SearchOptions) string {
	var sb strings.Builder

	for _, r := range results {
		renderSearchResultBodyFile(&sb, r, reg, opts)
	}

	if truncated {
		fmt.Fprintf(&sb, "\n[Results truncated to fit token budget (%d)]\n", tokenBudget)
	}

	return sb.String()
}

func renderSearchResultBodyFile(sb *strings.Builder, result SearchResult, reg *locator.Registry, opts SearchOptions) {
	fmt.Fprintf(sb, "%s\n", formatSearchResultFileHeader(result, reg, opts))

	prevLineNum := -1
	for _, match := range sortMatchesForSearchResultBody(result.Matches) {
		if match.LineNum < 0 {
			appendSearchResultBodyDetachedLine(sb, match.Line)
			prevLineNum = -1
			continue
		}

		appendSearchResultBodyGapMarkerIfNeeded(sb, prevLineNum, match.LineNum)
		prevLineNum = match.LineNum
		appendSearchResultBodyMatchLine(sb, result.FilePath, match, reg, opts)
	}
}

func formatSearchResultFileHeader(result SearchResult, reg *locator.Registry, opts SearchOptions) string {
	fileHeader := fmt.Sprintf("\n📄 %s (%d match(es))", result.FilePath, result.MatchCount)
	if reg != nil {
		id := reg.Register(newTextSearchLocator(result.FilePath, 0, 0, "", opts))
		fileHeader += " " + id
	}
	return fileHeader
}

func sortMatchesForSearchResultBody(matches []Match) []Match {
	blocks := buildMatchBlocks(matches)
	sort.SliceStable(blocks, func(i, j int) bool {
		return blocks[i].typ < blocks[j].typ
	})

	sorted := make([]Match, 0, len(matches))
	for _, block := range blocks {
		sorted = append(sorted, block.matches...)
	}
	return sorted
}

func appendSearchResultBodyDetachedLine(sb *strings.Builder, line string) {
	sb.WriteString("      ...\n")
	fmt.Fprintf(sb, "  %10s       │ %s\n", "", line)
	sb.WriteString("      ...\n")
}

func appendSearchResultBodyGapMarkerIfNeeded(sb *strings.Builder, prevLineNum, currentLineNum int) {
	if prevLineNum > 0 && currentLineNum != prevLineNum+1 {
		sb.WriteString("      ...\n")
	}
}

func appendSearchResultBodyMatchLine(sb *strings.Builder, filePath string, match Match, reg *locator.Registry, opts SearchOptions) {
	if match.IsMatch {
		fmt.Fprintf(sb, "  %-10s> %4d │ %s\n", matchTypeTag[match.Type], match.LineNum, match.Line)
		appendSearchResultBodyBlockLine(sb, filePath, match, reg, opts)
		return
	}
	fmt.Fprintf(sb, "  %10s  %4d │ %s\n", "", match.LineNum, match.Line)
}

func appendSearchResultBodyBlockLine(sb *strings.Builder, filePath string, match Match, reg *locator.Registry, opts SearchOptions) {
	if match.Block == nil {
		return
	}

	blockLine := ""
	if match.Block.StartLine > 0 {
		blockLine = fmt.Sprintf("  %10s  %4s   ── in %s (L%d)", "", "", match.Block.Name, match.Block.StartLine)
	} else {
		blockLine = fmt.Sprintf("  %10s  %4s   ── in %s", "", "", match.Block.Name)
	}
	if reg != nil && match.Block.StartLine > 0 {
		id := reg.Register(newTextSearchLocator(filePath, match.Block.StartLine, 0, match.Block.Name, opts))
		blockLine += " " + id
	}
	fmt.Fprintf(sb, "%s\n", blockLine)
}

// formatMultiResults は複数パターンの検索結果をフォーマットする
func formatMultiResults(collected []patternResult, tokenBudget int) string {
	return formatMultiResultsWithOptions(collected, tokenBudget, SearchOptions{})
}

func formatMultiResultsWithOptions(collected []patternResult, tokenBudget int, opts SearchOptions) string {
	var b strings.Builder

	totalMatches := 0
	matchedPatterns := 0
	for _, pr := range collected {
		for _, r := range pr.Results {
			totalMatches += r.MatchCount
		}
		if len(pr.Results) > 0 {
			matchedPatterns++
		}
	}

	if totalMatches == 0 {
		return "No matches found\n"
	}

	fmt.Fprintf(&b, "Found %d match(es) across %d/%d patterns\n\n", totalMatches, matchedPatterns, len(collected))

	budgetPerPattern := tokenBudget / len(collected)
	for i, pr := range collected {
		fmt.Fprintf(&b, "━━ Pattern %d/%d: %q ━━\n", i+1, len(collected), pr.Pattern)
		for _, w := range pr.Warnings {
			fmt.Fprintf(&b, "%s\n", w)
		}
		if len(pr.Warnings) > 0 {
			b.WriteString("\n")
		}
		if pr.Error != "" {
			fmt.Fprintf(&b, "⚠️ Error: %s\n\n", pr.Error)
			continue
		}

		if len(pr.Results) == 0 {
			b.WriteString("No matches found\n\n")
			continue
		}

		b.WriteString(formatSearchResultsBody(pr.Results, pr.Truncated, budgetPerPattern, nil, opts))
		b.WriteString("\n")
	}

	return b.String()
}

func formatManifestLineWithOptions(r SearchResult, reg *locator.Registry, opts SearchOptions) string {
	blockNames := collectUniqueBlockNames(r.Matches, 3)
	var line string
	if len(blockNames) > 0 {
		line = fmt.Sprintf("  %-40s — %d matches (%s)", r.FilePath, r.MatchCount, strings.Join(blockNames, ", "))
	} else {
		line = fmt.Sprintf("  %-40s — %d matches", r.FilePath, r.MatchCount)
	}
	if reg != nil {
		id := reg.Register(newTextSearchLocator(r.FilePath, 0, 0, "", opts))
		line += " " + id
	}
	return line
}

// collectUniqueBlockNames はマッチ群からユニークなブロック名を出現順で最大 limit 個返す
func collectUniqueBlockNames(matches []Match, limit int) []string {
	seen := make(map[string]bool)
	var names []string
	for _, m := range matches {
		if !m.IsMatch || m.Block == nil {
			continue
		}
		if !seen[m.Block.Name] {
			seen[m.Block.Name] = true
			names = append(names, m.Block.Name)
			if len(names) >= limit {
				break
			}
		}
	}
	return names
}

// formatManifestResults は単一パターンのマニフェスト出力を生成する
func formatManifestResults(results []SearchResult, reg *locator.Registry) string {
	return formatManifestResultsWithOptions(results, reg, SearchOptions{})
}

func formatManifestResultsWithOptions(results []SearchResult, reg *locator.Registry, opts SearchOptions) string {
	totalMatches := 0
	for _, r := range results {
		totalMatches += r.MatchCount
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d matches in %d files:\n", totalMatches, len(results))
	for _, r := range results {
		sb.WriteString(formatManifestLineWithOptions(r, reg, opts))
		sb.WriteString("\n")
	}
	return sb.String()
}

// formatManifestMultiResults は複数パターンのマニフェスト出力を生成する
func formatManifestMultiResults(collected []patternResult, reg *locator.Registry) string {
	return formatManifestMultiResultsWithOptions(collected, reg, SearchOptions{})
}

func formatManifestMultiResultsWithOptions(collected []patternResult, reg *locator.Registry, opts SearchOptions) string {
	var sb strings.Builder
	totalMatches := 0
	for _, pr := range collected {
		for _, r := range pr.Results {
			totalMatches += r.MatchCount
		}
	}
	if totalMatches == 0 {
		return "No matches found\n"
	}

	fmt.Fprintf(&sb, "Found %d matches across %d patterns:\n\n", totalMatches, len(collected))
	for i, pr := range collected {
		fmt.Fprintf(&sb, "━━ Pattern %d/%d: %q ━━\n", i+1, len(collected), pr.Pattern)
		if pr.Error != "" {
			fmt.Fprintf(&sb, "⚠️ Error: %s\n\n", pr.Error)
			continue
		}
		if len(pr.Results) == 0 {
			sb.WriteString("No matches found\n\n")
			continue
		}
		for _, r := range pr.Results {
			sb.WriteString(formatManifestLineWithOptions(r, reg, opts))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
