package search

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

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
	IsMatch bool // true=マッチ行, false=コンテキスト行
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

	// ReadTracker 連携: 結果ファイルを既読マーク
	for _, r := range results {
		if absPath, err := filepath.Abs(r.FilePath); err == nil {
			tools.GlobalReadTracker.MarkRead(absPath)
		}
	}

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
				sr.Matches = append(sr.Matches, Match{
					LineNum: data.LineNumber,
					Line:    strings.TrimRight(data.Lines.Text, "\n"),
					IsMatch: true,
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
		sr.Matches = append(sr.Matches, Match{
			LineNum: lineNum,
			Line:    content,
			IsMatch: isMatch,
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

		prevLineNum := -1
		for _, m := range r.Matches {
			// 非連続行間に "..." を表示
			if prevLineNum > 0 && m.LineNum > prevLineNum+1 {
				sb.WriteString("      ...\n")
			}
			prevLineNum = m.LineNum

			if m.IsMatch {
				sb.WriteString(fmt.Sprintf("  > %4d │ %s\n", m.LineNum, m.Line))
			} else {
				sb.WriteString(fmt.Sprintf("    %4d │ %s\n", m.LineNum, m.Line))
			}
		}
	}

	if truncated {
		sb.WriteString(fmt.Sprintf("\n[Results truncated to fit token budget (%d)]\n", tokenBudget))
	}

	return sb.String()
}
