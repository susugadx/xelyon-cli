package common

import (
	"path/filepath"
	"sort"
	"strings"
)

// BlockRange はファイル内のブロック（関数/クラス）の範囲
type BlockRange struct {
	Name      string
	StartLine int
	EndLine   int
}

// IsCommentLine はコメント行かどうか判定する
func IsCommentLine(trimmed string) bool {
	for _, prefix := range []string{"//", "/*", "#", "--", ";", `"""`, "'''"} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

// StripQuoted は文字列リテラル内の内容を除去する（クォート外のみ残す）
// ダブルクォート、シングルクォート、バッククォート（Go struct tag 等）に対応
func StripQuoted(s string) string {
	var result strings.Builder
	inDouble, inSingle, inBacktick := false, false, false
	escaped := false
	for _, ch := range s {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && (inDouble || inSingle) {
			escaped = true
			continue
		}
		if ch == '"' && !inSingle && !inBacktick {
			inDouble = !inDouble
			continue
		}
		if ch == '\'' && !inDouble && !inBacktick {
			inSingle = !inSingle
			continue
		}
		if ch == '`' && !inDouble && !inSingle {
			inBacktick = !inBacktick
			continue
		}
		if !inDouble && !inSingle && !inBacktick {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

// IsBraceLanguage はファイル拡張子からブレース言語かどうか判定する
func IsBraceLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".py", ".pyw", ".yaml", ".yml", ".coffee":
		return false
	default:
		return true
	}
}

// ExtractBlockName は宣言行からブロック名を抽出する
func ExtractBlockName(line string) string {
	// Go method receiver: func (f *Foo) Bar(...)
	if strings.HasPrefix(line, "func (") {
		closeIdx := strings.Index(line, ") ")
		if closeIdx > 0 {
			rest := line[closeIdx+2:]
			nameEnd := strings.IndexAny(rest, "( {")
			if nameEnd < 0 {
				nameEnd = len(rest)
			}
			name := strings.TrimSpace(rest[:nameEnd])
			if name != "" {
				return "func " + name
			}
		}
		return ""
	}
	// General: keyword + name
	keywords := []string{
		"func ", "fn ", "def ", "function ", "sub ", "method ",
		"type ", "class ", "struct ", "interface ", "enum ", "trait ",
	}
	for _, kw := range keywords {
		if strings.HasPrefix(line, kw) {
			rest := line[len(kw):]
			nameEnd := strings.IndexAny(rest, "( {:\n")
			if nameEnd < 0 {
				nameEnd = len(rest)
			}
			name := strings.TrimSpace(rest[:nameEnd])
			if name != "" {
				return strings.TrimSpace(kw) + " " + name
			}
		}
	}
	return ""
}

// CountIndent は行のインデント幅を返す（タブ=4スペース換算）
func CountIndent(line string) int {
	count := 0
	for _, ch := range line {
		switch ch {
		case ' ':
			count++
		case '\t':
			count += 4
		default:
			return count
		}
	}
	return count
}

// BuildBlockMap はファイル内容からブロック範囲リストを構築する
func BuildBlockMap(content string, isBrace bool) []BlockRange {
	lines := strings.Split(content, "\n")
	if isBrace {
		return buildBlockMapBrace(lines)
	}
	return buildBlockMapIndent(lines)
}

// buildBlockMapBrace はブレース言語用のブロック検出
func buildBlockMapBrace(lines []string) []BlockRange {
	var ranges []BlockRange
	type stackEntry struct {
		name      string
		startLine int
		depth     int // ブロック開始前の depth
	}
	var stack []stackEntry
	depth := 0
	var inBlockComment bool

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// 複数行コメントスキップ
		if inBlockComment {
			if strings.Contains(trimmed, "*/") {
				inBlockComment = false
			}
			continue
		}
		// stripQuoted 後に /* が含まれれば複数行コメント開始
		stripped := StripQuoted(line)
		if strings.Contains(stripped, "/*") {
			if !strings.Contains(stripped, "*/") {
				inBlockComment = true
			}
			continue
		}

		// 単一行コメントスキップ
		if IsCommentLine(trimmed) {
			continue
		}

		// 宣言検出
		if name := ExtractBlockName(trimmed); name != "" {
			stack = append(stack, stackEntry{name: name, startLine: lineNum, depth: depth})
		}

		// 文字列リテラル内の {} を除外してカウント
		for _, ch := range stripped {
			switch ch {
			case '{':
				depth++
			case '}':
				depth--
				if depth < 0 {
					depth = 0
				}
				// スタック top の depth >= 現在 depth → ブロック終了
				for len(stack) > 0 && stack[len(stack)-1].depth >= depth {
					top := stack[len(stack)-1]
					stack = stack[:len(stack)-1]
					ranges = append(ranges, BlockRange{
						Name:      top.name,
						StartLine: top.startLine,
						EndLine:   lineNum,
					})
				}
			}
		}
	}

	// 未クローズブロック → EndLine = EOF
	totalLines := len(lines)
	for _, s := range stack {
		ranges = append(ranges, BlockRange{
			Name:      s.name,
			StartLine: s.startLine,
			EndLine:   totalLines,
		})
	}

	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].StartLine < ranges[j].StartLine
	})

	return ranges
}

// buildBlockMapIndent はインデント言語用のブロック検出
func buildBlockMapIndent(lines []string) []BlockRange {
	var ranges []BlockRange
	type stackEntry struct {
		name      string
		startLine int
		indent    int
	}
	var stack []stackEntry

	for i, line := range lines {
		lineNum := i + 1
		if strings.TrimSpace(line) == "" {
			continue // 空行スキップ
		}

		indent := CountIndent(line)

		// 現在行のインデント <= スタック top のインデント → ブロック終了
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			ranges = append(ranges, BlockRange{
				Name:      top.name,
				StartLine: top.startLine,
				EndLine:   lineNum - 1,
			})
		}

		trimmed := strings.TrimSpace(line)
		if name := ExtractBlockName(trimmed); name != "" {
			stack = append(stack, stackEntry{name: name, startLine: lineNum, indent: indent})
		}
	}

	// 未クローズブロック → EndLine = EOF
	totalLines := len(lines)
	for _, s := range stack {
		ranges = append(ranges, BlockRange{
			Name:      s.name,
			StartLine: s.startLine,
			EndLine:   totalLines,
		})
	}

	sort.Slice(ranges, func(i, j int) bool {
		return ranges[i].StartLine < ranges[j].StartLine
	})

	return ranges
}
