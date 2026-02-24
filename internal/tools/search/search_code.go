package search

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// MatchType はマッチ行の種別（ソート順序を定義）
type MatchType int

const (
	MatchTypeDefinition MatchType = iota // 0: func/type/class 等の定義
	MatchTypeImport                      // 1: import/require/use 等
	MatchTypeAssignment                  // 2: := や = による代入
	MatchTypeUsage                       // 3: その他の参照・使用
	MatchTypeComment                     // 4: コメント行
)

// matchTypeTag はマッチ種別の表示タグ
var matchTypeTag = [5]string{"[def]", "[import]", "[assign]", "[use]", "[comment]"}

// BlockInfo はマッチが所属するブロック（関数/クラス）の情報
type BlockInfo struct {
	Name      string // "func handleSSEResponse", "class MyClass" 等
	StartLine int
}

// SearchResult はファイルごとの検索結果
type SearchResult struct {
	FilePath   string
	Matches    []Match
	MatchCount int // マッチ行のみのカウント
}

// Match はマッチ行またはコンテキスト行
type Match struct {
	LineNum int
	Line    string
	IsMatch bool       // true=マッチ行, false=コンテキスト行
	Type    MatchType  // マッチ種別（ソート用）
	Block   *BlockInfo // マッチが所属するブロック（nil=トップレベル）
}

// ExecuteSearchCode はコード検索を実行し、フォーマット済み結果を返す
func ExecuteSearchCode(pattern, path, filePattern, contextLinesStr, tokenBudgetStr string) string {
	// 引数バリデーション
	if pattern == "" {
		return "Error: pattern is required"
	}
	if path == "" {
		path = "."
	}

	ctxLines := 3
	if contextLinesStr != "" {
		if n, err := strconv.Atoi(contextLinesStr); err == nil {
			if n < 0 {
				n = 0
			}
			if n > 10 {
				n = 10
			}
			ctxLines = n
		}
	}

	tokenBudget := 3000
	if tokenBudgetStr != "" {
		if n, err := strconv.Atoi(tokenBudgetStr); err == nil {
			if n < 500 {
				n = 500
			}
			if n > 6000 {
				n = 6000
			}
			tokenBudget = n
		}
	}

	// キャッシュチェック
	cacheKey := path + "|" + filePattern
	if tools.GlobalToolCache != nil {
		if cached, ok := tools.GlobalToolCache.GetSearch(pattern, cacheKey); ok {
			return cached
		}
	}

	// ripgrep or grep 実行
	output, useRipgrep := executeSearch(pattern, path, filePattern, ctxLines)

	// 結果パース
	var results []SearchResult
	if useRipgrep {
		results = parseRipgrepJSON(output)
	} else {
		results = parseGrepOutput(output)
	}

	if len(results) == 0 {
		return "No matches found"
	}

	// コンテキスト行マージ
	results = mergeContextLines(results)

	// トークンバジェット制御
	results, truncated := truncateToTokenBudget(results, tokenBudget)

	// ReadTracker 連携: 結果ファイルの行範囲を既読マーク（str_replace line-range モードで read_file なし編集を許可）
	for _, r := range results {
		if absPath, err := filepath.Abs(r.FilePath); err == nil {
			if len(r.Matches) > 0 {
				// Matches はソート済みとは限らないため min/max で算出
				startLine := r.Matches[0].LineNum
				endLine := r.Matches[0].LineNum
				for _, m := range r.Matches {
					if m.LineNum < startLine {
						startLine = m.LineNum
					}
					if m.LineNum > endLine {
						endLine = m.LineNum
					}
				}
				tools.GlobalReadTracker.MarkReadRange(absPath, startLine, endLine)
			}
		}
	}

	// ブロック認識（関数/クラス境界検出）
	detectBlocks(results)

	// 出力フォーマット
	formatted := formatSearchResults(results, truncated, tokenBudget)

	// キャッシュ保存
	if tools.GlobalToolCache != nil {
		tools.GlobalToolCache.SetSearch(pattern, cacheKey, formatted)
	}

	return formatted
}

// executeSearch は rg（優先）または grep を実行し、出力と使用ツールを返す
func executeSearch(pattern, path, filePattern string, ctxLines int) (string, bool) {
	// ripgrep を試行
	if rgPath, err := exec.LookPath("rg"); err == nil {
		args := []string{
			"--json",
			"-n",
			"--max-count", "30",
		}
		if ctxLines > 0 {
			args = append(args, "--context", strconv.Itoa(ctxLines))
		}
		if filePattern != "" {
			args = append(args, "--glob", filePattern)
		}
		args = append(args, pattern, path)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, rgPath, args...)
		out, _ := cmd.Output() // rg はマッチなしで exit 1 を返すのでエラーは無視
		return string(out), true
	}

	// grep フォールバック
	args := []string{
		"-rn",
		"-I",
		"-m", "30",
		"--exclude-dir=.git",
		"--exclude-dir=node_modules",
		"--exclude-dir=vendor",
		"--exclude-dir=.next",
	}
	if filePattern != "" {
		args = append(args, "--include="+filePattern)
	}
	if ctxLines > 0 {
		args = append(args, "-C", strconv.Itoa(ctxLines))
	}
	args = append(args, pattern, path)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "grep", args...)
	out, _ := cmd.Output() // grep もマッチなしで exit 1
	return string(out), false
}

// --- rg --json パーサー ---

// rgJSONLine は rg --json 出力の1行
type rgJSONLine struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type rgMatchData struct {
	Path       rgPath `json:"path"`
	LineNumber int    `json:"line_number"`
	Lines      rgText `json:"lines"`
}

type rgPath struct {
	Text string `json:"text"`
}

type rgText struct {
	Text string `json:"text"`
}

type rgBeginData struct {
	Path rgPath `json:"path"`
}

func parseRipgrepJSON(output string) []SearchResult {
	if output == "" {
		return nil
	}

	fileMap := make(map[string]*SearchResult)
	var fileOrder []string
	var currentFile string

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var entry rgJSONLine
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		switch entry.Type {
		case "begin":
			var data rgBeginData
			if err := json.Unmarshal(entry.Data, &data); err == nil {
				currentFile = data.Path.Text
				if _, exists := fileMap[currentFile]; !exists {
					fileMap[currentFile] = &SearchResult{FilePath: currentFile}
					fileOrder = append(fileOrder, currentFile)
				}
			}

		case "match":
			var data rgMatchData
			if err := json.Unmarshal(entry.Data, &data); err == nil {
				filePath := currentFile
				if data.Path.Text != "" {
					filePath = data.Path.Text
				}
				if _, exists := fileMap[filePath]; !exists {
					fileMap[filePath] = &SearchResult{FilePath: filePath}
					fileOrder = append(fileOrder, filePath)
				}
				sr := fileMap[filePath]
				lineText := strings.TrimRight(data.Lines.Text, "\n")
				sr.Matches = append(sr.Matches, Match{
					LineNum: data.LineNumber,
					Line:    lineText,
					IsMatch: true,
					Type:    classifyMatch(lineText),
				})
				sr.MatchCount++
			}

		case "context":
			var data rgMatchData
			if err := json.Unmarshal(entry.Data, &data); err == nil {
				filePath := currentFile
				if data.Path.Text != "" {
					filePath = data.Path.Text
				}
				if _, exists := fileMap[filePath]; !exists {
					fileMap[filePath] = &SearchResult{FilePath: filePath}
					fileOrder = append(fileOrder, filePath)
				}
				sr := fileMap[filePath]
				sr.Matches = append(sr.Matches, Match{
					LineNum: data.LineNumber,
					Line:    strings.TrimRight(data.Lines.Text, "\n"),
					IsMatch: false,
					Type:    MatchTypeUsage,
				})
			}

			// "end", "summary" → 無視
		}
	}

	// ファイル順序を保持して結果配列を構築
	var results []SearchResult
	for _, fp := range fileOrder {
		results = append(results, *fileMap[fp])
	}
	return results
}

// --- grep 出力パーサー ---

func parseGrepOutput(output string) []SearchResult {
	if output == "" {
		return nil
	}

	fileMap := make(map[string]*SearchResult)
	var fileOrder []string

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		// ブロック境界セパレータ
		if line == "--" {
			continue
		}

		filePath, lineNum, content, isMatch := parseGrepLine(line)
		if filePath == "" {
			continue
		}

		if _, exists := fileMap[filePath]; !exists {
			fileMap[filePath] = &SearchResult{FilePath: filePath}
			fileOrder = append(fileOrder, filePath)
		}

		sr := fileMap[filePath]
		matchType := MatchTypeUsage
		if isMatch {
			matchType = classifyMatch(content)
		}
		sr.Matches = append(sr.Matches, Match{
			LineNum: lineNum,
			Line:    content,
			IsMatch: isMatch,
			Type:    matchType,
		})
		if isMatch {
			sr.MatchCount++
		}
	}

	var results []SearchResult
	for _, fp := range fileOrder {
		results = append(results, *fileMap[fp])
	}
	return results
}

// parseGrepLine は grep の1行をパースする
// マッチ行: file.go:13:  content   (区切り :)
// コンテキスト行: file.go-12-content  (区切り -)
func parseGrepLine(line string) (filePath string, lineNum int, content string, isMatch bool) {
	// マッチ行パターン: file:linenum:content
	// コンテキスト行パターン: file-linenum-content
	// ファイルパスに : を含む可能性があるため、行番号部分を基準にパース

	// まずマッチ行（:区切り）を試行
	if fp, ln, c, ok := tryParseGrepMatch(line, ":"); ok {
		return fp, ln, c, true
	}
	// コンテキスト行（-区切り）を試行
	if fp, ln, c, ok := tryParseGrepMatch(line, "-"); ok {
		return fp, ln, c, false
	}
	return "", 0, "", false
}

// tryParseGrepMatch は指定セパレータで grep 行をパースする
func tryParseGrepMatch(line, sep string) (filePath string, lineNum int, content string, ok bool) {
	// file<sep>linenum<sep>content のパターンを探す
	// 最初の sep+数字+sep のパターンを見つける
	idx := 0
	for {
		pos := strings.Index(line[idx:], sep)
		if pos < 0 {
			return "", 0, "", false
		}
		pos += idx

		// sep の後に数字が続くか確認
		numStart := pos + len(sep)
		numEnd := numStart
		for numEnd < len(line) && line[numEnd] >= '0' && line[numEnd] <= '9' {
			numEnd++
		}
		if numEnd == numStart {
			idx = pos + len(sep)
			continue
		}

		// 数字の後に同じ sep が続くか確認
		if numEnd+len(sep) > len(line) || line[numEnd:numEnd+len(sep)] != sep {
			idx = pos + len(sep)
			continue
		}

		n, err := strconv.Atoi(line[numStart:numEnd])
		if err != nil || n <= 0 {
			idx = pos + len(sep)
			continue
		}

		filePath = line[:pos]
		lineNum = n
		content = line[numEnd+len(sep):]
		return filePath, lineNum, content, true
	}
}

// --- コンテキスト行マージ ---

func mergeContextLines(results []SearchResult) []SearchResult {
	merged := make([]SearchResult, 0, len(results))
	for _, r := range results {
		if len(r.Matches) == 0 {
			continue
		}

		// 行番号でソート済みと仮定（rg/grep は行番号順に出力）
		// 重複行番号を除去（O(1) ルックアップ）
		seen := make(map[int]int) // lineNum → index in deduped
		var deduped []Match
		for _, m := range r.Matches {
			if idx, exists := seen[m.LineNum]; exists {
				// 重複時はマッチ行を優先
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

// --- トークンバジェット制御 ---

func truncateToTokenBudget(results []SearchResult, budget int) ([]SearchResult, bool) {
	// ヘッダー行のオーバーヘッド推定
	headerTokens := 10 // "Found N matches in M files\n\n"
	usedTokens := headerTokens
	truncated := false

	var kept []SearchResult
	for _, r := range results {
		// ファイルヘッダーのトークン推定
		fileHeader := len(r.FilePath)/4 + 5 // "📄 path (N matches)\n"
		if usedTokens+fileHeader > budget {
			truncated = true
			break
		}
		usedTokens += fileHeader

		var keptMatches []Match
		matchCount := 0
		for _, m := range r.Matches {
			lineTokens := len(m.Line)/4 + 3 // 行番号 + セパレータ + 内容
			if m.IsMatch {
				lineTokens += 10 // ブロック注釈分
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

// --- マッチ種別分類 ---

// classifyMatch はマッチ行の内容から種別を判定する（言語非依存の汎用パターン）
func classifyMatch(line string) MatchType {
	trimmed := strings.TrimSpace(line)

	// 1. コメント
	if isCommentLine(trimmed) {
		return MatchTypeComment
	}

	// 2. Import（定義より先に判定: require() を含む行の誤分類防止）
	importKeywords := []string{"import ", "from ", "use ", "include ", "using "}
	for _, kw := range importKeywords {
		if strings.HasPrefix(trimmed, kw) {
			return MatchTypeImport
		}
	}
	if strings.Contains(trimmed, "require(") {
		return MatchTypeImport
	}

	// 3. 定義（インデントなし = 行頭が空白でない）
	if line == trimmed {
		defKeywords := []string{
			"func ", "fn ", "def ", "function ", "sub ", "method ",
			"type ", "class ", "struct ", "interface ", "enum ", "trait ",
			"const ", "var ", "let ", "static ", "pub ", "export ",
			"module ", "namespace ", "package ",
		}
		for _, kw := range defKeywords {
			if strings.HasPrefix(trimmed, kw) {
				return MatchTypeDefinition
			}
		}
	}

	// 4. 代入
	if hasAssignment(line) {
		return MatchTypeAssignment
	}

	// 5. 使用（デフォルト）
	return MatchTypeUsage
}

// isCommentLine はコメント行かどうか判定する
func isCommentLine(trimmed string) bool {
	for _, prefix := range []string{"//", "/*", "#", "--", ";", `"""`, "'''"} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

// stripQuoted は文字列リテラル内の内容を除去する（クォート外のみ残す）
func stripQuoted(s string) string {
	var result strings.Builder
	inDouble, inSingle := false, false
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
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if !inDouble && !inSingle {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

// hasAssignment は行に代入演算子が含まれるか判定する（比較演算子・文字列内を除外）
func hasAssignment(line string) bool {
	s := stripQuoted(line)
	for _, op := range []string{"===", "!==", "==", "!=", ">=", "<=", "=>"} {
		s = strings.ReplaceAll(s, op, "")
	}
	return strings.Contains(s, ":=") || strings.Contains(s, "=")
}

// --- マッチブロック ---

// matchBlock はマッチ行とその前後のコンテキスト行をまとめたブロック
type matchBlock struct {
	matches []Match
	typ     MatchType
}

// buildMatchBlocks は deduped な Match リストをブロックに分割する
// コンテキスト行はマッチ行に到達するまで pending に蓄積し、次のマッチブロックの先頭に付与
func buildMatchBlocks(matches []Match) []matchBlock {
	var blocks []matchBlock
	var pending []Match

	for _, m := range matches {
		if m.IsMatch {
			var blockMatches []Match
			blockMatches = append(blockMatches, pending...)
			blockMatches = append(blockMatches, m)
			blocks = append(blocks, matchBlock{typ: m.Type, matches: blockMatches})
			pending = nil
		} else {
			pending = append(pending, m)
		}
	}
	// 末尾コンテキスト → 最後のブロック
	if len(pending) > 0 && len(blocks) > 0 {
		last := &blocks[len(blocks)-1]
		last.matches = append(last.matches, pending...)
	}
	return blocks
}

// --- ブロック認識（関数/クラス境界検出） ---

// blockRange はファイル内のブロック（関数/クラス）の範囲
type blockRange struct {
	Name      string
	StartLine int
	EndLine   int
}

// isBraceLanguage はファイル拡張子からブレース言語かどうか判定する
func isBraceLanguage(filePath string) bool {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".py", ".pyw", ".yaml", ".yml", ".coffee":
		return false
	default:
		return true
	}
}

// extractBlockName は宣言行からブロック名を抽出する
func extractBlockName(line string) string {
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

// countIndent は行のインデント幅を返す（タブ=4スペース換算）
func countIndent(line string) int {
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

// buildBlockMap はファイル内容からブロック範囲リストを構築する
func buildBlockMap(content string, isBrace bool) []blockRange {
	lines := strings.Split(content, "\n")
	if isBrace {
		return buildBlockMapBrace(lines)
	}
	return buildBlockMapIndent(lines)
}

// buildBlockMapBrace はブレース言語用のブロック検出
func buildBlockMapBrace(lines []string) []blockRange {
	var ranges []blockRange
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
		stripped := stripQuoted(line)
		if strings.Contains(stripped, "/*") {
			if !strings.Contains(stripped, "*/") {
				inBlockComment = true
			}
			continue
		}

		// 単一行コメントスキップ
		if isCommentLine(trimmed) {
			continue
		}

		// 宣言検出
		if name := extractBlockName(trimmed); name != "" {
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
					ranges = append(ranges, blockRange{
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
		ranges = append(ranges, blockRange{
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
func buildBlockMapIndent(lines []string) []blockRange {
	var ranges []blockRange
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

		indent := countIndent(line)

		// 現在行のインデント <= スタック top のインデント → ブロック終了
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			ranges = append(ranges, blockRange{
				Name:      top.name,
				StartLine: top.startLine,
				EndLine:   lineNum - 1,
			})
		}

		trimmed := strings.TrimSpace(line)
		if name := extractBlockName(trimmed); name != "" {
			stack = append(stack, stackEntry{name: name, startLine: lineNum, indent: indent})
		}
	}

	// 未クローズブロック → EndLine = EOF
	totalLines := len(lines)
	for _, s := range stack {
		ranges = append(ranges, blockRange{
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

// findBlockForLine は指定行番号を含む最内ブロックを返す
func findBlockForLine(ranges []blockRange, lineNum int) *BlockInfo {
	var best *blockRange
	for i := range ranges {
		r := &ranges[i]
		if lineNum >= r.StartLine && lineNum <= r.EndLine {
			if best == nil || r.StartLine > best.StartLine {
				best = r // 最内ブロック優先
			}
		}
	}
	if best == nil {
		return nil
	}
	return &BlockInfo{Name: best.Name, StartLine: best.StartLine}
}

// getFileContent はファイル内容を取得する（キャッシュ連携）
func getFileContent(filePath string) string {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return ""
	}
	if tools.GlobalToolCache != nil {
		if cached, ok := tools.GlobalToolCache.GetFile(absPath); ok {
			return cached
		}
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return ""
	}
	content := string(data)
	if tools.GlobalToolCache != nil {
		tools.GlobalToolCache.SetFile(absPath, content)
	}
	return content
}

// detectBlocks は検索結果の各マッチに所属ブロック情報を付与する
func detectBlocks(results []SearchResult) {
	for i := range results {
		r := &results[i]
		content := getFileContent(r.FilePath)
		if content == "" {
			continue
		}
		isBrace := isBraceLanguage(r.FilePath)
		blocks := buildBlockMap(content, isBrace)
		for j := range r.Matches {
			m := &r.Matches[j]
			if m.IsMatch {
				m.Block = findBlockForLine(blocks, m.LineNum)
				// マッチ自身がブロック開始行なら注釈不要
				if m.Block != nil && m.Block.StartLine == m.LineNum {
					m.Block = nil
				}
			}
		}
	}
}

// --- 出力フォーマット ---

func formatSearchResults(results []SearchResult, truncated bool, tokenBudget int) string {
	totalMatches := 0
	for _, r := range results {
		totalMatches += r.MatchCount
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d match(es) in %d file(s)\n", totalMatches, len(results)))

	for _, r := range results {
		sb.WriteString(fmt.Sprintf("\n📄 %s (%d match(es))\n", r.FilePath, r.MatchCount))

		// ブロック分割 → MatchType でソート → 展開
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
			// 非連続行間に "..." を表示（ソートによる逆方向ジャンプにも対応）
			if prevLineNum > 0 && m.LineNum != prevLineNum+1 {
				sb.WriteString("      ...\n")
			}
			prevLineNum = m.LineNum

			if m.IsMatch {
				sb.WriteString(fmt.Sprintf("  %-10s> %4d │ %s\n", matchTypeTag[m.Type], m.LineNum, m.Line))
				if m.Block != nil {
					sb.WriteString(fmt.Sprintf("  %10s  %4s   ── in %s (L%d)\n", "", "", m.Block.Name, m.Block.StartLine))
				}
			} else {
				sb.WriteString(fmt.Sprintf("  %10s  %4d │ %s\n", "", m.LineNum, m.Line))
			}
		}
	}

	if truncated {
		sb.WriteString(fmt.Sprintf("\n[Results truncated to fit token budget (%d)]\n", tokenBudget))
	}

	return sb.String()
}
