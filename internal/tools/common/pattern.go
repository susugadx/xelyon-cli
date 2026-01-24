package common

import (
	"fmt"
)

// PatternMatchResult はパターンマッチの結果を保持
type PatternMatchResult struct {
	MatchIdx     int   // マッチした行インデックス（-1 = not found）
	MatchCount   int   // マッチ数
	MatchIndices []int // 全マッチ位置
}

// FindPatternInLines は行配列からパターンを検索
// Tier 1: Exact match, Tier 2: Normalized whitespace match
func FindPatternInLines(lines []string, pattern string) PatternMatchResult {
	result := PatternMatchResult{MatchIdx: -1}

	// Tier 1: Exact match
	for i, line := range lines {
		if line == pattern {
			result.MatchIdx = i
			result.MatchCount++
			result.MatchIndices = append(result.MatchIndices, i)
		}
	}

	// Tier 2: Normalized whitespace match (only if Tier 1 found nothing)
	if result.MatchIdx == -1 {
		normalizedPattern := NormalizeLeadingWhitespace(pattern)
		for i, line := range lines {
			if NormalizeLeadingWhitespace(line) == normalizedPattern {
				result.MatchIdx = i
				result.MatchCount++
				result.MatchIndices = append(result.MatchIndices, i)
			}
		}
	}

	return result
}

// DisplayPatternNotFound はパターンが見つからない場合のエラー表示
func DisplayPatternNotFound(pattern string, lines []string, maxPreview int) {
	Yellow.Printf("Pattern not found: %s\n\n", pattern)
	Yellow.Println("File preview (first 50 lines):")
	for i := 0; i < Min(len(lines), maxPreview); i++ {
		fmt.Printf("%4d: %s\n", i+1, lines[i])
	}
	if len(lines) > maxPreview {
		Yellow.Printf("... (%d more lines)\n", len(lines)-maxPreview)
	}
}

// DisplayMultipleMatches は複数マッチ時のエラー表示
func DisplayMultipleMatches(matchIndices []int, lines []string) {
	Red.Printf("Error: Pattern matches %d locations (must be unique)\n", len(matchIndices))
	Yellow.Println("All match locations:")
	for _, idx := range matchIndices {
		start := Max(0, idx-2)
		end := Min(len(lines), idx+3)
		for i := start; i < end; i++ {
			prefix := "  "
			if i == idx {
				prefix = "> "
			}
			fmt.Printf("%s%4d: %s\n", prefix, i+1, lines[i])
		}
		fmt.Println()
	}
}

// DisplayContextAround はマッチ行の周囲コンテキストを表示
func DisplayContextAround(matchIdx int, lines []string, contextLines int) {
	Green.Printf("Pattern found at line %d\n\n", matchIdx+1)
	Yellow.Printf("Context (%d lines before/after):\n", contextLines)
	start := Max(0, matchIdx-contextLines)
	end := Min(len(lines), matchIdx+contextLines+1)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == matchIdx {
			prefix = "> "
		}
		fmt.Printf("%s%4d: %s\n", prefix, i+1, lines[i])
	}
}

// DisplayContentToInsert は挿入する内容を表示
func DisplayContentToInsert(content string) {
	Cyan.Println("\n---- Content to insert ----")
	fmt.Println(content)
	Cyan.Println("---------------------------")
}
