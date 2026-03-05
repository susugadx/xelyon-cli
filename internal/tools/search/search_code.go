package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
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

// SearchOptions はコード検索のオプション
type SearchOptions struct {
	Pattern        string
	Path           string
	FilePattern    string
	FileType       string
	CtxLines       int
	TokenBudget    int
	IsRegex        bool
	Multiline      bool
	IncludeHidden  bool
	IncludeIgnored bool
}

// ExecuteSearchCode はコード検索を実行し、フォーマット済み結果を返す
func ExecuteSearchCode(opts SearchOptions) string {
	// 引数バリデーション
	if opts.Pattern == "" {
		return "Error: pattern is required"
	}
	if opts.Path == "" {
		opts.Path = "."
	}

	if opts.CtxLines < 0 {
		opts.CtxLines = 3
	}
	if opts.CtxLines > 10 {
		opts.CtxLines = 10
	}

	if opts.TokenBudget < 0 {
		opts.TokenBudget = 3000
	}
	if opts.TokenBudget < 500 {
		opts.TokenBudget = 500
	}
	if opts.TokenBudget > 6000 {
		opts.TokenBudget = 6000
	}

	patterns := splitPatterns(opts.Pattern)
	if len(patterns) > 1 {
		// 複数パターン: マルチキャッシュチェック → 並列検索
		multiKey := buildMultiCacheKey(patterns)
		cacheKey := buildSearchCacheKey(opts)
		if tools.GlobalToolCache != nil {
			if cached, ok := tools.GlobalToolCache.GetSearch(multiKey, cacheKey); ok {
				return cached
			}
		}
		return executeMultiplePatterns(patterns, opts)
	}
	return executeSinglePattern(patterns[0], opts)
}

// executeSinglePattern は単一パターンの検索処理（キャッシュ・検索・パース・マージ・トランケート・ブロック認識・フォーマット・キャッシュ保存）
func executeSinglePattern(pattern string, opts SearchOptions) string {
	// キャッシュチェック
	cacheKey := buildSearchCacheKey(opts)
	if tools.GlobalToolCache != nil {
		if cached, ok := tools.GlobalToolCache.GetSearch(pattern, cacheKey); ok {
			return cached
		}
	}

	maxCountPerFile := calcMaxCountPerFile(opts.TokenBudget)

	// ripgrep 検索
	output, useRipgrep, warnings, err := executeSearch(pattern, opts, maxCountPerFile)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	var results []SearchResult
	if useRipgrep {
		results = parseRipgrepJSON(output, 200)
	} else {
		results = parseGrepOutput(output, 200)
	}

	if len(results) == 0 {
		if len(warnings) > 0 {
			return strings.Join(warnings, "\n") + "\nNo matches found"
		}
		return "No matches found"
	}

	// コンテキスト行マージ
	results = mergeContextLines(results)
	// マッチ数に応じてコンテキスト行を適応的に縮小
	results = adaptiveContextTrim(results)
	// ファイル優先度ソート（非テスト→テスト、定義あり→なし）
	sortResultsByPriority(results)

	// トークンバジェット制御
	results, truncated := truncateToTokenBudget(results, opts.TokenBudget)

	// ブロック認識（関数/クラス境界検出）
	detectBlocks(results)

	// 出力フォーマット
	formatted := formatSearchResults(results, truncated, opts.TokenBudget)

	// キャッシュ保存
	if tools.GlobalToolCache != nil {
		tools.GlobalToolCache.SetSearch(pattern, cacheKey, formatted, collectFilePaths(results))
	}

	if len(warnings) > 0 {
		return strings.Join(warnings, "\n") + "\n" + formatted
	}

	return formatted
}

const escapedCommaPlaceholder = "\x00COMMA\x00"

// splitPatterns はカンマ区切りのパターン文字列を分割する。
// \, はリテラルカンマとして扱う。空文字除外、TrimSpace、最大 5 パターン。
func splitPatterns(pattern string) []string {
	// エスケープカンマを一時退避
	s := strings.ReplaceAll(pattern, `\,`, escapedCommaPlaceholder)
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// プレースホルダをリテラルカンマに復元
		p = strings.ReplaceAll(p, escapedCommaPlaceholder, ",")
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) > 5 {
		result = result[:5]
	}
	return result
}

// patternResult は複数パターン検索の各パターンの結果
type patternResult struct {
	Pattern      string
	Results      []SearchResult
	Truncated    bool
	Index        int
	TotalMatches int // truncate前の全マッチ数（バジェット比例配分に使用）
	Error        string
	Warnings     []string
}

// executeMultiplePatterns は複数パターンを goroutine 並列で検索する
func executeMultiplePatterns(patterns []string, opts SearchOptions) string {
	budgetPerPattern := opts.TokenBudget / len(patterns)
	if budgetPerPattern < 500 {
		budgetPerPattern = 500
	}

	maxCountPerFile := calcMaxCountPerFile(budgetPerPattern)

	ch := make(chan patternResult, len(patterns))
	for i, p := range patterns {
		go func(idx int, pat string) {
			// ripgrep 検索
			output, useRg, searchWarnings, searchErr := executeSearch(pat, opts, maxCountPerFile)
			if searchErr != nil {
				ch <- patternResult{Pattern: pat, Index: idx, Error: searchErr.Error(), Warnings: searchWarnings}
				return
			}
			maxMatches := 200 / len(patterns)
			if maxMatches < 50 {
				maxMatches = 50
			}
			var results []SearchResult
			if useRg {
				results = parseRipgrepJSON(output, maxMatches)
			} else {
				results = parseGrepOutput(output, maxMatches)
			}
			results = mergeContextLines(results)
			results = adaptiveContextTrim(results)
			sortResultsByPriority(results)

			// truncate前のマッチ数を記録（比例配分に使用）
			totalMatches := 0
			for _, r := range results {
				totalMatches += r.MatchCount
			}

			ch <- patternResult{Pattern: pat, Results: results, Index: idx, TotalMatches: totalMatches, Warnings: searchWarnings}
		}(i, p)
	}

	collected := make([]patternResult, len(patterns))
	for range patterns {
		r := <-ch
		collected[r.Index] = r
	}

	// パス2: マッチ数に比例してバジェット再配分
	totalAllMatches := 0
	for _, c := range collected {
		totalAllMatches += c.TotalMatches
	}
	for i, c := range collected {
		var allocatedBudget int
		if totalAllMatches == 0 {
			allocatedBudget = opts.TokenBudget / len(patterns)
		} else {
			allocatedBudget = opts.TokenBudget * c.TotalMatches / totalAllMatches
		}
		if allocatedBudget < 300 {
			allocatedBudget = 300
		}
		collected[i].Results, collected[i].Truncated = truncateToTokenBudget(c.Results, allocatedBudget)
		detectBlocks(collected[i].Results)
	}

	formatted := formatMultiResults(collected, opts.TokenBudget)

	// キャッシュ保存（multi 全体を1エントリとして保存）
	if tools.GlobalToolCache != nil {
		multiKey := buildMultiCacheKey(patterns)
		cacheKey := buildSearchCacheKey(opts)
		allFiles := make([]string, 0)
		for _, c := range collected {
			allFiles = append(allFiles, collectFilePaths(c.Results)...)
		}
		tools.GlobalToolCache.SetSearch(multiKey, cacheKey, formatted, dedupePaths(allFiles))
	}

	return formatted
}

// buildMultiCacheKey は複数パターンからソート済みキャッシュキーを構築する
func buildMultiCacheKey(patterns []string) string {
	sorted := make([]string, len(patterns))
	copy(sorted, patterns)
	sort.Strings(sorted)
	return strings.Join(sorted, "|")
}

func buildSearchCacheKey(opts SearchOptions) string {
	return fmt.Sprintf("%s|%s|%s|%d|%d|regex=%t|multiline=%t|hidden=%t|ignored=%t",
		opts.Path, opts.FilePattern, opts.FileType, opts.CtxLines, opts.TokenBudget, opts.IsRegex, opts.Multiline, opts.IncludeHidden, opts.IncludeIgnored)
}

func collectFilePaths(results []SearchResult) []string {
	paths := make([]string, 0, len(results))
	for _, r := range results {
		if r.FilePath == "" {
			continue
		}
		if absPath, err := filepath.Abs(r.FilePath); err == nil {
			paths = append(paths, absPath)
		} else {
			paths = append(paths, r.FilePath)
		}
	}
	return dedupePaths(paths)
}

func dedupePaths(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func fileTypeToGlob(fileType string) (string, bool) {
	typeToGlob := map[string]string{
		"go":    "*.go",
		"py":    "*.py",
		"js":    "*.js",
		"ts":    "*.ts",
		"rust":  "*.rs",
		"java":  "*.java",
		"c":     "*.c",
		"cpp":   "*.cpp",
		"rb":    "*.rb",
		"php":   "*.php",
		"swift": "*.swift",
	}
	glob, ok := typeToGlob[fileType]
	return glob, ok
}

var (
	gnuGrepCheckOnce sync.Once
	gnuGrepAvailable bool
)

func isGNUGrep() bool {
	gnuGrepCheckOnce.Do(func() {
		out, err := exec.Command("grep", "--version").CombinedOutput()
		if err != nil {
			gnuGrepAvailable = false
			return
		}
		gnuGrepAvailable = strings.Contains(strings.ToLower(string(out)), "gnu grep")
	})
	return gnuGrepAvailable
}

// calcMaxCountPerFile はトークンバジェットからファイルあたりのマッチ上限を計算する
// 1マッチあたり平均 ~30 トークン（行 + コンテキスト + ブロック注釈）と見積もり
func calcMaxCountPerFile(budget int) int {
	n := budget / 30
	if n < 10 {
		n = 10
	}
	if n > 50 {
		n = 50
	}
	return n
}

// executeSearch は rg（優先）または grep を実行し、出力と使用ツールを返す
// maxCountPerFile はファイルあたりのマッチ上限（ripgrep --max-count に対応）
func executeSearch(pattern string, opts SearchOptions, maxCountPerFile int) (string, bool, []string, error) {
	// ripgrep を試行
	if rgPath, err := exec.LookPath("rg"); err == nil {
		args := []string{
			"--json",
			"-n",
			"--max-count", strconv.Itoa(maxCountPerFile),
		}
		if opts.CtxLines > 0 {
			args = append(args, "--context", strconv.Itoa(opts.CtxLines))
		}
		if opts.FileType != "" {
			args = append(args, "--type", opts.FileType)
		} else if opts.FilePattern != "" {
			args = append(args, "--glob", opts.FilePattern)
		}
		if !opts.IsRegex {
			args = append(args, "--fixed-strings")
		}
		if opts.Multiline {
			args = append(args, "--multiline")
		}
		if opts.IncludeHidden {
			args = append(args, "--hidden")
		}
		if opts.IncludeIgnored {
			args = append(args, "--no-ignore")
		}
		args = append(args, pattern, opts.Path)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, rgPath, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		_ = cmd.Run() // rg はマッチなしで exit 1 を返すのでエラーは無視
		if stdout.Len() == 0 && stderr.Len() > 0 {
			return "", true, nil, fmt.Errorf("regex error: %s", strings.TrimSpace(stderr.String()))
		}
		return stdout.String(), true, nil, nil
	}

	// grep フォールバック（rg がない環境用）
	// Note: grep は .gitignore を参照しない。主要ディレクトリは --exclude-dir で除外。rg 推奨。
	warnings := []string{
		"Warning: ripgrep (rg) not found; using grep fallback mode.",
	}
	args := []string{
		"-rn",
		"-I",
		"-m", strconv.Itoa(maxCountPerFile), // NOTE: grep -m はファイルあたりN行（rg --max-count と同じ）
		"--exclude-dir=.git",
		"--exclude-dir=node_modules",
		"--exclude-dir=vendor",
		"--exclude-dir=.next",
	}
	if opts.IsRegex {
		args = append(args, "-E") // 拡張正規表現（rg と同等の regex 解釈 + 不正 regex のエラー検出）
	} else {
		args = append(args, "-F")
	}
	if !opts.IncludeHidden {
		// rg のデフォルト挙動に寄せる。GNU grep でのみ --exclude 系を使用。
		if isGNUGrep() {
			args = append(args, "--exclude=.*", "--exclude-dir=.*")
		} else {
			warnings = append(warnings, "Warning: hidden-file exclusion is not fully supported in grep fallback mode on non-GNU grep")
		}
	} else {
		warnings = append(warnings, "Warning: include_hidden is partially supported in grep fallback mode")
	}

	if opts.FileType != "" {
		if glob, ok := fileTypeToGlob(opts.FileType); ok {
			args = append(args, "--include="+glob)
		} else {
			warnings = append(warnings, fmt.Sprintf("Warning: file_type=%q is not supported in grep fallback mode (rg not found)", opts.FileType))
			if opts.FilePattern != "" {
				args = append(args, "--include="+opts.FilePattern)
			}
		}
	} else if opts.FilePattern != "" {
		args = append(args, "--include="+opts.FilePattern)
	}

	if opts.Multiline {
		warnings = append(warnings, "Warning: multiline search is not supported in grep fallback mode (rg not found)")
	}
	if opts.CtxLines > 0 {
		args = append(args, "-C", strconv.Itoa(opts.CtxLines))
	}
	args = append(args, pattern, opts.Path)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "grep", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run() // grep もマッチなしで exit 1
	if stdout.Len() == 0 && stderr.Len() > 0 {
		return "", false, warnings, fmt.Errorf("regex error: %s", strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), false, warnings, nil
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

func parseRipgrepJSON(output string, maxTotalMatches int) []SearchResult {
	if output == "" {
		return nil
	}

	fileMap := make(map[string]*SearchResult)
	var fileOrder []string
	var currentFile string
	totalMatches := 0

	lines := strings.Split(strings.TrimSpace(output), "\n")
outer:
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
				totalMatches++
				if totalMatches >= maxTotalMatches {
					break outer
				}
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

func parseGrepOutput(output string, maxTotalMatches int) []SearchResult {
	if output == "" {
		return nil
	}

	fileMap := make(map[string]*SearchResult)
	var fileOrder []string
	totalMatches := 0

	lines := strings.Split(strings.TrimSpace(output), "\n")
outer:
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
			totalMatches++
			if totalMatches >= maxTotalMatches {
				break outer
			}
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

// --- adaptive context trim ---

// adaptiveContextTrim はファイルごとのマッチ数に応じてコンテキスト行を削る
// 5マッチ以下: そのまま、6-15マッチ: 前後1行のみ、16マッチ以上: コンテキスト行なし
func adaptiveContextTrim(results []SearchResult) []SearchResult {
	for i, r := range results {
		matchCount := 0
		for _, m := range r.Matches {
			if m.IsMatch {
				matchCount++
			}
		}
		if matchCount <= 5 {
			continue // 5マッチ以下はコンテキスト維持
		}

		// マッチが多いファイル: コンテキスト行をマッチ行の直前後 maxCtx 行のみに縮小
		maxCtx := 1
		if matchCount > 15 {
			maxCtx = 0 // 15マッチ超はコンテキスト行なし
		}

		var trimmed []Match
		for _, m := range r.Matches {
			if m.IsMatch {
				trimmed = append(trimmed, m)
				continue
			}
			if maxCtx == 0 {
				continue
			}
			// コンテキスト行: 行番号の差でマッチ行との距離を判定
			nearMatch := false
			for k := range r.Matches {
				if r.Matches[k].IsMatch && absInt(m.LineNum-r.Matches[k].LineNum) <= maxCtx {
					nearMatch = true
					break
				}
			}
			if nearMatch {
				trimmed = append(trimmed, m)
			}
		}
		results[i].Matches = trimmed
	}
	return results
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// --- ファイル優先度ソート ---

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

// isTestFile はファイルパスがテストファイルかどうか判定する
func isTestFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".test.js") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".spec.js") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.Contains(base, "test_")
}

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
			lineTokens := estimateTokens(m.Line) // 行番号 + セパレータ + 内容
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

var controlFlowKeywords = map[string]bool{
	"if": true, "else": true, "for": true, "while": true,
	"switch": true, "case": true, "return": true, "break": true,
	"continue": true, "throw": true, "try": true, "catch": true,
	"finally": true, "select": true, "range": true, "yield": true, "await": true,
}

// classifyMatch はマッチ行の内容から種別を判定する（言語非依存の汎用パターン）
func classifyMatch(line string) MatchType {
	trimmed := strings.TrimSpace(line)

	// 1. コメント
	if common.IsCommentLine(trimmed) {
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

	strippedTrimmed := common.StripModifiers(trimmed)

	// 3. 定義（インデント問わず defKeyword で始まる行）
	defKeywords := []string{
		"func ", "fn ", "def ", "function ", "sub ", "method ",
		"type ", "class ", "struct ", "interface ", "enum ", "trait ",
		"const ", "var ", "let ", "static ", "pub ", "export ",
		"module ", "namespace ", "package ",
	}
	for _, kw := range defKeywords {
		if strings.HasPrefix(strippedTrimmed, kw) {
			return MatchTypeDefinition
		}
	}

	// 戻り値型スキップ判定
	parts := strings.Fields(strippedTrimmed)
	if len(parts) > 0 {
		firstWord := parts[0]
		// 制御構文の場合は即座に Usage を返す（代入と誤判定させない）
		if controlFlowKeywords[firstWord] {
			return MatchTypeUsage
		}

		if len(parts) >= 2 {
			rest := strings.TrimSpace(strippedTrimmed[len(firstWord):])
			// 識別子 + '(' であれば定義とみなす
			parenIdx := strings.Index(rest, "(")
			if parenIdx > 0 {
				idCandidate := strings.TrimSpace(rest[:parenIdx])
				if isValidIdentifier(idCandidate) {
					return MatchTypeDefinition
				}
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

// isValidIdentifier は関数名などの識別子として妥当か判定する（ジェネリクス <> [] 対応）
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	// ジェネリクスの括弧を取り除く
	idx := strings.IndexAny(s, "<[")
	if idx >= 0 {
		s = s[:idx]
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	for i, r := range s {
		if i == 0 {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && r != '_' && r < 0x80 {
				return false
			}
		} else {
			if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' && r < 0x80 {
				return false
			}
		}
	}
	return true
}

// hasAssignment は行に代入演算子が含まれるか判定する（比較演算子・文字列内を除外）
func hasAssignment(line string) bool {
	s := common.StripQuoted(line)
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

// findBlockForLine は指定行番号を含む最内ブロックを返す
func findBlockForLine(ranges []common.BlockRange, lineNum int) *BlockInfo {
	var best *common.BlockRange
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
		isBrace := common.IsBraceLanguage(r.FilePath)
		blocks := common.BuildBlockMap(content, isBrace)
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
	fmt.Fprintf(&sb, "Found %d match(es) in %d file(s)\n", totalMatches, len(results))
	sb.WriteString(formatSearchResultsBody(results, truncated, tokenBudget))
	return sb.String()
}

// formatSearchResultsBody はファイルごとの結果部分のみをフォーマットする
// formatSearchResults と formatMultiResults の両方から呼ばれる
func formatSearchResultsBody(results []SearchResult, truncated bool, tokenBudget int) string {
	var sb strings.Builder

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
				sb.WriteString(fmt.Sprintf("  %-10s> %4d │ %s\n", matchTypeTag[m.Type], m.LineNum, truncateLine(m.Line)))
				if m.Block != nil {
					sb.WriteString(fmt.Sprintf("  %10s  %4s   ── in %s (L%d)\n", "", "", m.Block.Name, m.Block.StartLine))
				}
			} else {
				sb.WriteString(fmt.Sprintf("  %10s  %4d │ %s\n", "", m.LineNum, truncateLine(m.Line)))
			}
		}
	}

	if truncated {
		sb.WriteString(fmt.Sprintf("\n[Results truncated to fit token budget (%d)]\n", tokenBudget))
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
