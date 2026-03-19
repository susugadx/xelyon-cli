package search

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

const maxLineLength = 500

func estimateTokens(line string) int {
	return utf8.RuneCountInString(line)/2 + len(line)/8 + 3
}

func truncateLine(line string) string {
	if utf8.RuneCountInString(line) > maxLineLength {
		return string([]rune(line)[:maxLineLength]) + "..."
	}
	return line
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

func formatSearchResults(results []SearchResult, truncated bool, tokenBudget int) string {
	totalMatches := 0
	for _, r := range results {
		totalMatches += r.MatchCount
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d match(es) in %d file(s)\n", totalMatches, len(results))
	sb.WriteString(formatSearchResultsBody(results, truncated, tokenBudget))
	return sb.String()
}

// formatSearchResultsBody はファイルごとの結果部分のみをフォーマットする
// formatSearchResults と formatMultiResults の両方から呼ばれる
func formatSearchResultsBody(results []SearchResult, truncated bool, tokenBudget int) string {
	var sb strings.Builder

	for _, r := range results {
		fmt.Fprintf(&sb, "\n📄 %s (%d match(es))\n", r.FilePath, r.MatchCount)

		blocks := buildMatchBlocks(r.Matches)
		sort.SliceStable(blocks, func(i, j int) bool {
			return blocks[i].typ < blocks[j].typ
		})
		var sorted []Match
		for _, b := range blocks {
			sorted = append(sorted, b.matches...)
		}

		prevLineNum := -1
		for _, m := range sorted {
			if m.LineNum < 0 {
				sb.WriteString("      ...\n")
				fmt.Fprintf(&sb, "  %10s       │ %s\n", "", m.Line)
				sb.WriteString("      ...\n")
				prevLineNum = -1
				continue
			}

			if prevLineNum > 0 && m.LineNum != prevLineNum+1 {
				sb.WriteString("      ...\n")
			}
			prevLineNum = m.LineNum

			if m.IsMatch {
				fmt.Fprintf(&sb, "  %-10s> %4d │ %s\n", matchTypeTag[m.Type], m.LineNum, truncateLine(m.Line))
				if m.Block != nil {
					if m.Block.StartLine > 0 {
						fmt.Fprintf(&sb, "  %10s  %4s   ── in %s (L%d)\n", "", "", m.Block.Name, m.Block.StartLine)
					} else {
						fmt.Fprintf(&sb, "  %10s  %4s   ── in %s\n", "", "", m.Block.Name)
					}
				}
			} else {
				fmt.Fprintf(&sb, "  %10s  %4d │ %s\n", "", m.LineNum, truncateLine(m.Line))
			}
		}
	}

	if truncated {
		fmt.Fprintf(&sb, "\n[Results truncated to fit token budget (%d)]\n", tokenBudget)
	}

	return sb.String()
}

// formatMultiResults は複数パターンの検索結果をフォーマットする
func formatMultiResults(collected []patternResult, tokenBudget int) string {
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

		b.WriteString(formatSearchResultsBody(pr.Results, pr.Truncated, budgetPerPattern))
		b.WriteString("\n")
	}

	return b.String()
}

// formatManifestLine はファイル1つ分のマニフェスト行を生成する。
// ブロック名を最大3つまで付記する。
func formatManifestLine(r SearchResult) string {
	blockNames := collectUniqueBlockNames(r.Matches, 3)
	if len(blockNames) > 0 {
		return fmt.Sprintf("  %-40s — %d matches (%s)", r.FilePath, r.MatchCount, strings.Join(blockNames, ", "))
	}
	return fmt.Sprintf("  %-40s — %d matches", r.FilePath, r.MatchCount)
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
func formatManifestResults(results []SearchResult) string {
	totalMatches := 0
	for _, r := range results {
		totalMatches += r.MatchCount
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d matches in %d files:\n", totalMatches, len(results))
	for _, r := range results {
		sb.WriteString(formatManifestLine(r))
		sb.WriteString("\n")
	}
	return sb.String()
}

// formatManifestMultiResults は複数パターンのマニフェスト出力を生成する
func formatManifestMultiResults(collected []patternResult) string {
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
			sb.WriteString(formatManifestLine(r))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
