package applypatch

import (
	"fmt"
	"strings"
)

// LocateResult はチャンク検索の結果を保持する。
type LocateResult struct {
	StartIdx  int      // マッチした開始行インデックス
	Pattern   []string // 実際にマッチしたパターン（末尾空行トリム後の場合あり）
	NewLines  []string // 対応するNewLines（末尾空行トリム後の場合あり）
	NextIndex int      // 次のチャンクの検索開始位置
}

// LocateChunk は1つのチャンクの適用位置を特定する。
// prevChunkEnd は前のチャンクの終了位置（オーバーラップ検出用、最初のチャンクは0）。
func LocateChunk(originalLines []string, path string, chunk UpdateFileChunk, lineIndex int, prevChunkEnd int) (LocateResult, error) {
	searchIndex, err := resolveChunkSearchIndex(originalLines, path, chunk.ChangeContext, lineIndex)
	if err != nil {
		return LocateResult{}, err
	}

	pattern := chunk.OldLines
	newSlice := chunk.NewLines
	if len(pattern) == 0 {
		return buildInsertionLocateResult(originalLines, pattern, newSlice, searchIndex), nil
	}

	startIdx, matchedPattern, matchedNewLines, ok := locateChunkPattern(
		originalLines,
		pattern,
		newSlice,
		searchIndex,
		prevChunkEnd,
		chunk.IsEndOfFile,
	)
	if !ok {
		return LocateResult{}, fmt.Errorf("failed to find expected lines in %s:\n%s", path, strings.Join(chunk.OldLines, "\n"))
	}

	return buildLocateResult(startIdx, matchedPattern, matchedNewLines), nil
}

func resolveChunkSearchIndex(originalLines []string, path string, changeContext string, lineIndex int) (int, error) {
	if changeContext == "" {
		return lineIndex, nil
	}

	idx, ok := SeekSequence(originalLines, []string{changeContext}, lineIndex, false)
	if ok {
		return idx + 1, nil
	}
	if looksLikeUnifiedDiffHeader(changeContext) {
		return 0, fmt.Errorf(
			"@@ header must be a code fragment (e.g. @@ func name), not unified diff line numbers — got: @@ %s",
			changeContext,
		)
	}
	return 0, fmt.Errorf("failed to find context '%s' in %s", changeContext, path)
}

func buildInsertionLocateResult(originalLines []string, pattern []string, newSlice []string, searchIndex int) LocateResult {
	insertionIdx := len(originalLines)
	if len(originalLines) > 0 && originalLines[len(originalLines)-1] == "" {
		insertionIdx = len(originalLines) - 1
	}
	return LocateResult{
		StartIdx:  insertionIdx,
		Pattern:   pattern,
		NewLines:  newSlice,
		NextIndex: searchIndex,
	}
}

func locateChunkPattern(
	originalLines []string,
	pattern []string,
	newSlice []string,
	searchIndex int,
	prevChunkEnd int,
	isEndOfFile bool,
) (startIdx int, matchedPattern []string, matchedNewLines []string, ok bool) {
	startIdx, ok = SeekSequence(originalLines, pattern, searchIndex, isEndOfFile)
	if ok {
		return startIdx, pattern, newSlice, true
	}

	trimmedPattern, trimmedNewLines, trimmed := trimTrailingEmptyLine(pattern, newSlice)
	if trimmed {
		startIdx, ok = SeekSequence(originalLines, trimmedPattern, searchIndex, isEndOfFile)
		if ok {
			return startIdx, trimmedPattern, trimmedNewLines, true
		}
	}

	startIdx, ok = fallbackSeekChunkPattern(originalLines, pattern, searchIndex, prevChunkEnd, isEndOfFile)
	if ok {
		return startIdx, pattern, newSlice, true
	}
	if trimmed {
		startIdx, ok = fallbackSeekChunkPattern(originalLines, trimmedPattern, searchIndex, prevChunkEnd, isEndOfFile)
		if ok {
			return startIdx, trimmedPattern, trimmedNewLines, true
		}
	}

	return 0, nil, nil, false
}

func trimTrailingEmptyLine(pattern []string, newSlice []string) ([]string, []string, bool) {
	if len(pattern) == 0 || pattern[len(pattern)-1] != "" {
		return pattern, newSlice, false
	}

	trimmedPattern := pattern[:len(pattern)-1]
	trimmedNewLines := newSlice
	if len(trimmedNewLines) > 0 && trimmedNewLines[len(trimmedNewLines)-1] == "" {
		trimmedNewLines = trimmedNewLines[:len(trimmedNewLines)-1]
	}
	return trimmedPattern, trimmedNewLines, true
}

func fallbackSeekChunkPattern(lines []string, pattern []string, searchIndex int, prevChunkEnd int, eof bool) (int, bool) {
	// フォールバック: @@コンテキストが変更対象より後ろにある場合（先頭行変更など）、
	// 先頭から再検索する。ただし前のチャンクとの重複は禁止する。
	if searchIndex <= 0 {
		return 0, false
	}

	startIdx, ok := SeekSequence(lines, pattern, 0, eof)
	if ok && startIdx < prevChunkEnd {
		return 0, false
	}
	return startIdx, ok
}

func buildLocateResult(startIdx int, pattern []string, newSlice []string) LocateResult {
	return LocateResult{
		StartIdx:  startIdx,
		Pattern:   pattern,
		NewLines:  newSlice,
		NextIndex: startIdx + len(pattern),
	}
}

// FindLineNumber は fileContent から changeContext の開始行番号を 1-based で返す。
// 見つからない場合は -1 を返す。
func FindLineNumber(fileContent string, changeContext string) int {
	lines := strings.Split(fileContent, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	idx, ok := SeekSequence(lines, []string{changeContext}, 0, false)
	if !ok {
		return -1
	}
	return idx + 1
}

// SeekSequence はファイル内の lines から pattern を検索する。
// start 以降で最初にマッチした位置のインデックスを返す。
// eof が true の場合、ファイル末尾からの検索を優先する。
func SeekSequence(lines []string, pattern []string, start int, eof bool) (int, bool) {
	if start < 0 {
		start = 0
	}
	if len(pattern) == 0 {
		return start, true
	}
	if len(pattern) > len(lines) {
		return 0, false
	}

	searchStart := start
	if eof && len(lines) >= len(pattern) {
		searchStart = len(lines) - len(pattern)
	}

	for _, matcher := range seekComparators {
		if idx, ok := seekWith(lines, pattern, searchStart, matcher); ok {
			return idx, true
		}
	}

	return 0, false
}

var seekComparators = []func(string, string) bool{
	func(a, b string) bool { return a == b },
	func(a, b string) bool { return strings.TrimRight(a, " \t") == strings.TrimRight(b, " \t") },
	func(a, b string) bool { return strings.TrimSpace(a) == strings.TrimSpace(b) },
	func(a, b string) bool { return normalizeUnicode(a) == normalizeUnicode(b) },
}

func seekWith(lines, pattern []string, start int, eq func(string, string) bool) (int, bool) {
	for i := start; i <= len(lines)-len(pattern); i++ {
		match := true
		for j, p := range pattern {
			if !eq(lines[i+j], p) {
				match = false
				break
			}
		}
		if match {
			return i, true
		}
	}
	return 0, false
}

func normalizeUnicode(s string) string {
	s = strings.TrimSpace(s)

	var b strings.Builder
	for _, c := range s {
		switch {
		case c == '\u2010' || c == '\u2011' || c == '\u2012' ||
			c == '\u2013' || c == '\u2014' || c == '\u2015' || c == '\u2212':
			b.WriteByte('-')
		case c == '\u2018' || c == '\u2019' || c == '\u201A' || c == '\u201B':
			b.WriteByte('\'')
		case c == '\u201C' || c == '\u201D' || c == '\u201E' || c == '\u201F':
			b.WriteByte('"')
		case c == '\u00A0' || (c >= '\u2002' && c <= '\u200A') ||
			c == '\u202F' || c == '\u205F' || c == '\u3000':
			b.WriteByte(' ')
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}
