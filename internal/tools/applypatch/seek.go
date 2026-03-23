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
	searchIndex := lineIndex
	if chunk.ChangeContext != "" {
		idx, ok := SeekSequence(originalLines, []string{chunk.ChangeContext}, lineIndex, false)
		if !ok {
			if looksLikeUnifiedDiffHeader(chunk.ChangeContext) {
				return LocateResult{}, fmt.Errorf(
					"@@ header must be a code fragment (e.g. @@ func name), not unified diff line numbers — got: @@ %s",
					chunk.ChangeContext,
				)
			}
			return LocateResult{}, fmt.Errorf("failed to find context '%s' in %s", chunk.ChangeContext, path)
		}
		searchIndex = idx + 1
	}

	pattern := chunk.OldLines
	newSlice := chunk.NewLines
	if len(pattern) == 0 {
		insertionIdx := len(originalLines)
		if len(originalLines) > 0 && originalLines[len(originalLines)-1] == "" {
			insertionIdx = len(originalLines) - 1
		}
		return LocateResult{
			StartIdx:  insertionIdx,
			Pattern:   pattern,
			NewLines:  newSlice,
			NextIndex: searchIndex,
		}, nil
	}

	startIdx, ok := SeekSequence(originalLines, pattern, searchIndex, chunk.IsEndOfFile)

	if !ok && len(pattern) > 0 && pattern[len(pattern)-1] == "" {
		pattern = pattern[:len(pattern)-1]
		if len(newSlice) > 0 && newSlice[len(newSlice)-1] == "" {
			newSlice = newSlice[:len(newSlice)-1]
		}
		startIdx, ok = SeekSequence(originalLines, pattern, searchIndex, chunk.IsEndOfFile)
	}

	// フォールバック: @@コンテキストが変更対象より後ろにある場合（先頭行変更など）、
	// 先頭から再検索する。ただし前のチャンクとの重複は禁止する。
	if !ok && searchIndex > 0 {
		startIdx, ok = SeekSequence(originalLines, pattern, 0, chunk.IsEndOfFile)
		if ok && startIdx < prevChunkEnd {
			ok = false
		}
	}

	if !ok {
		return LocateResult{}, fmt.Errorf("failed to find expected lines in %s:\n%s", path, strings.Join(chunk.OldLines, "\n"))
	}

	return LocateResult{
		StartIdx:  startIdx,
		Pattern:   pattern,
		NewLines:  newSlice,
		NextIndex: startIdx + len(pattern),
	}, nil
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

	if idx, ok := seekWith(lines, pattern, searchStart, func(a, b string) bool { return a == b }); ok {
		return idx, true
	}
	if idx, ok := seekWith(lines, pattern, searchStart, func(a, b string) bool {
		return strings.TrimRight(a, " \t") == strings.TrimRight(b, " \t")
	}); ok {
		return idx, true
	}
	if idx, ok := seekWith(lines, pattern, searchStart, func(a, b string) bool {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}); ok {
		return idx, true
	}
	if idx, ok := seekWith(lines, pattern, searchStart, func(a, b string) bool {
		return normalizeUnicode(a) == normalizeUnicode(b)
	}); ok {
		return idx, true
	}

	return 0, false
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
