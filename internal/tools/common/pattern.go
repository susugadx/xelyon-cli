package common

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
			if result.MatchIdx == -1 {
				result.MatchIdx = i
			}
			result.MatchCount++
			result.MatchIndices = append(result.MatchIndices, i)
		}
	}

	// Tier 2: Normalized whitespace match (only if Tier 1 found nothing)
	if result.MatchIdx == -1 {
		normalizedPattern := NormalizeLeadingWhitespace(pattern)
		for i, line := range lines {
			if NormalizeLeadingWhitespace(line) == normalizedPattern {
				if result.MatchIdx == -1 {
					result.MatchIdx = i
				}
				result.MatchCount++
				result.MatchIndices = append(result.MatchIndices, i)
			}
		}
	}

	return result
}

// DisplayPatternNotFound はパターンが見つからない場合のエラー表示
func DisplayPatternNotFound(pattern string, lines []string, maxPreview int) {
	DisplayPatternNotFoundWithOutput(DefaultOutput(), pattern, lines, maxPreview)
}

// DisplayPatternNotFoundWithOutput は出力先を指定して未検出メッセージを表示する。
func DisplayPatternNotFoundWithOutput(out Output, pattern string, lines []string, maxPreview int) {
	if out.SuppressStdout() {
		return
	}
	out.Yellow.Printf("Pattern not found: %s\n\n", pattern)
	out.Yellow.Println("File preview (first 50 lines):")
	for i := 0; i < Min(len(lines), maxPreview); i++ {
		out.Printf("%4d: %s\n", i+1, lines[i])
	}
	if len(lines) > maxPreview {
		out.Yellow.Printf("... (%d more lines)\n", len(lines)-maxPreview)
	}
}

// DisplayMultipleMatches は複数マッチ時のエラー表示
func DisplayMultipleMatches(matchIndices []int, lines []string) {
	DisplayMultipleMatchesWithOutput(DefaultOutput(), matchIndices, lines)
}

// DisplayMultipleMatchesWithOutput は出力先を指定して複数マッチを表示する。
func DisplayMultipleMatchesWithOutput(out Output, matchIndices []int, lines []string) {
	if out.SuppressStdout() {
		return
	}
	out.Red.Printf("Error: Pattern matches %d locations (must be unique)\n", len(matchIndices))
	out.Yellow.Println("All match locations:")
	for _, idx := range matchIndices {
		start := Max(0, idx-2)
		end := Min(len(lines), idx+3)
		for i := start; i < end; i++ {
			prefix := "  "
			if i == idx {
				prefix = "> "
			}
			out.Printf("%s%4d: %s\n", prefix, i+1, lines[i])
		}
		out.Println()
	}
}

// DisplayContextAround はマッチ行の周囲コンテキストを表示
func DisplayContextAround(matchIdx int, lines []string, contextLines int) {
	DisplayContextAroundWithOutput(DefaultOutput(), matchIdx, lines, contextLines)
}

// DisplayContextAroundWithOutput は出力先を指定して周辺コンテキストを表示する。
func DisplayContextAroundWithOutput(out Output, matchIdx int, lines []string, contextLines int) {
	if out.SuppressStdout() {
		return
	}
	out.Green.Printf("Pattern found at line %d\n\n", matchIdx+1)
	out.Yellow.Printf("Context (%d lines before/after):\n", contextLines)
	start := Max(0, matchIdx-contextLines)
	end := Min(len(lines), matchIdx+contextLines+1)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == matchIdx {
			prefix = "> "
		}
		out.Printf("%s%4d: %s\n", prefix, i+1, lines[i])
	}
}

// DisplayContentToInsert は挿入する内容を表示
func DisplayContentToInsert(content string) {
	DisplayContentToInsertWithOutput(DefaultOutput(), content)
}

// DisplayContentToInsertWithOutput は出力先を指定して挿入内容を表示する。
func DisplayContentToInsertWithOutput(out Output, content string) {
	if out.SuppressStdout() {
		return
	}
	out.Cyan.Println("\n---- Content to insert ----")
	out.Println(content)
	out.Cyan.Println("---------------------------")
}
