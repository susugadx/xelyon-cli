package search

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// MatchType はマッチ行の種別（ソート順序を定義）
type MatchType int

const (
	MatchTypeDefinition MatchType = iota // 0: func/type/class 等の定義
	MatchTypeImport                      // 1: import/require/use 等
	MatchTypeCall                        // 2: 関数/メソッド呼び出し
	MatchTypeAssignment                  // 3: := や = による代入
	MatchTypeRef                         // 4: その他の参照
	MatchTypeComment                     // 5: コメント行
	MatchTypeString                      // 6: 文字列リテラル
)

const MatchTypeUsage MatchType = MatchTypeRef // 後方互換: 旧名称

// lineRangeHint は search_code 結果末尾に付与する str_replace line-range モード推奨ヒント。
// UI には表示されず、LLM の tool result にのみ含まれる。
const lineRangeHint = "\n\nTip: Use str_replace line-range mode (old_str empty + start_line/end_line) to edit matched lines directly."

// matchTypeTag はマッチ種別の表示タグ
var matchTypeTag = [7]string{"[def]", "[import]", "[call]", "[assign]", "[ref]", "[comment]", "[string]"}

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
	TokenBudget    int // 内部固定値（15000）。外部パラメータは廃止。
	IsRegex        bool
	Multiline      bool
	IncludeHidden  bool
	IncludeIgnored bool
	OutputMode     string // ""（default）, "full", "manifest"

	ignoreMatcher *pathmatch.Matcher
	ignoreGlobs   []string
	ignoreKey     string
}

// ExecuteSearchCode はコード検索を実行し、フォーマット済み結果を返す
func ExecuteSearchCode(opts SearchOptions) string {
	return ExecuteSearchCodeWithConfig(nil, nil, opts)
}

// ExecuteSearchCodeWithCache はキャッシュを指定してコード検索を実行する。
func ExecuteSearchCodeWithCache(cache tools.ToolCacheInterface, opts SearchOptions) string {
	return ExecuteSearchCodeWithConfig(nil, cache, opts)
}

// ExecuteSearchCodeWithConfig は設定とキャッシュを指定してコード検索を実行する。
func ExecuteSearchCodeWithConfig(cfg *config.Config, cache tools.ToolCacheInterface, opts SearchOptions) string {
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

	// token_budget is fixed internally.
	// 15000 tokens cover almost all normal searches without truncation.
	// Only abnormal wide searches should hit this safety valve.
	opts.TokenBudget = 15000

	if !opts.IncludeIgnored {
		projectCfg := config.LoadProjectConfig()
		ignorePatterns := config.ResolveSharedIgnorePatterns(cfg, projectCfg)
		opts.ignoreMatcher = pathmatch.NewMatcher(ignorePatterns)
		opts.ignoreGlobs = pathmatch.BuildRGIgnoreGlobs(ignorePatterns)
		opts.ignoreKey = strings.Join(ignorePatterns, ",")
	}

	patterns := splitPatterns(opts.Pattern)
	if len(patterns) > 1 {
		multiKey := buildMultiCacheKey(patterns)
		cacheKey := buildSearchCacheKey(opts)
		if cache != nil {
			if cached, ok := cache.GetSearch(multiKey, cacheKey); ok {
				return cached
			}
		}
		return executeMultiplePatterns(cache, patterns, opts)
	}
	return executeSinglePattern(cache, patterns[0], opts)
}

// executeSinglePattern は単一パターンの検索処理（キャッシュ・検索・パース・マージ・トランケート・ブロック認識・フォーマット・キャッシュ保存）
func executeSinglePattern(cache tools.ToolCacheInterface, pattern string, opts SearchOptions) string {
	cacheKey := buildSearchCacheKey(opts)
	if cache != nil {
		if cached, ok := cache.GetSearch(pattern, cacheKey); ok {
			return cached
		}
	}

	maxCountPerFile := calcMaxCountPerFile(opts.TokenBudget)

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
	results = filterResultsByOptions(results, opts)
	reclassifyWithAST(results, pattern, opts.IsRegex)

	if len(results) == 0 {
		if len(warnings) > 0 {
			return strings.Join(warnings, "\n") + "\nNo matches found"
		}
		return "No matches found"
	}

	if opts.OutputMode == "manifest" {
		sortResultsByPriority(results)
		detectBlocksWithCache(cache, results)
		formatted := formatManifestResults(results)
		finalOutput := formatted
		if len(warnings) > 0 {
			finalOutput = strings.Join(warnings, "\n") + "\n" + formatted
		}

		if cache != nil {
			cache.SetSearch(pattern, cacheKey, finalOutput, collectFilePaths(results))
		}
		return finalOutput
	}

	results = mergeContextLines(results)
	results = adaptiveContextTrim(results)
	sortResultsByPriority(results)

	results, truncated := truncateToTokenBudget(results, opts.TokenBudget, false)

	detectBlocksWithCache(cache, results)
	collapseBlockMatches(results)

	formatted := formatSearchResults(results, truncated, opts.TokenBudget)
	finalOutput := formatted
	if len(warnings) > 0 {
		finalOutput = strings.Join(warnings, "\n") + "\n" + formatted
	}
	finalOutput += lineRangeHint

	if cache != nil {
		cache.SetSearch(pattern, cacheKey, finalOutput, collectFilePaths(results))
	}

	return finalOutput
}

const escapedCommaPlaceholder = "\x00COMMA\x00"

// splitPatterns はカンマ区切りのパターン文字列を分割する。
// \, はリテラルカンマとして扱う。空文字除外、TrimSpace、最大 5 パターン。
func splitPatterns(pattern string) []string {
	s := strings.ReplaceAll(pattern, `\,`, escapedCommaPlaceholder)
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
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
func executeMultiplePatterns(cache tools.ToolCacheInterface, patterns []string, opts SearchOptions) string {
	budgetPerPattern := opts.TokenBudget / len(patterns)
	if budgetPerPattern < 500 {
		budgetPerPattern = 500
	}

	maxCountPerFile := calcMaxCountPerFile(budgetPerPattern)

	ch := make(chan patternResult, len(patterns))
	for i, p := range patterns {
		go func(idx int, pat string) {
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
			results = filterResultsByOptions(results, opts)
			reclassifyWithAST(results, pat, opts.IsRegex)
			results = mergeContextLines(results)
			results = adaptiveContextTrim(results)
			sortResultsByPriority(results)

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

	totalAllMatches := 0
	for _, c := range collected {
		totalAllMatches += c.TotalMatches
	}
	if opts.OutputMode == "manifest" {
		for i := range collected {
			detectBlocksWithCache(cache, collected[i].Results)
		}
		formatted := formatManifestMultiResults(collected)

		if cache != nil {
			multiKey := buildMultiCacheKey(patterns)
			cacheKey := buildSearchCacheKey(opts)
			allFiles := make([]string, 0)
			for _, c := range collected {
				allFiles = append(allFiles, collectFilePaths(c.Results)...)
			}
			cache.SetSearch(multiKey, cacheKey, formatted, dedupePaths(allFiles))
		}
		return formatted
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
		collected[i].Results, collected[i].Truncated = truncateToTokenBudget(c.Results, allocatedBudget, false)
		detectBlocksWithCache(cache, collected[i].Results)
		collapseBlockMatches(collected[i].Results)
	}

	formatted := formatMultiResults(collected, opts.TokenBudget)
	formattedWithHint := formatted + lineRangeHint

	if cache != nil {
		multiKey := buildMultiCacheKey(patterns)
		cacheKey := buildSearchCacheKey(opts)
		allFiles := make([]string, 0)
		for _, c := range collected {
			allFiles = append(allFiles, collectFilePaths(c.Results)...)
		}
		cache.SetSearch(multiKey, cacheKey, formattedWithHint, dedupePaths(allFiles))
	}

	return formattedWithHint
}

// buildMultiCacheKey は複数パターンからソート済みキャッシュキーを構築する
func buildMultiCacheKey(patterns []string) string {
	sorted := make([]string, len(patterns))
	copy(sorted, patterns)
	sort.Strings(sorted)
	return strings.Join(sorted, "|")
}

func buildSearchCacheKey(opts SearchOptions) string {
	return fmt.Sprintf("%s|%s|%s|%d|%d|regex=%t|multiline=%t|hidden=%t|ignored=%t|mode=%s|ignore=%s",
		opts.Path, opts.FilePattern, opts.FileType, opts.CtxLines, opts.TokenBudget, opts.IsRegex, opts.Multiline, opts.IncludeHidden, opts.IncludeIgnored, opts.OutputMode, opts.ignoreKey)
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
	if common.IsRipgrepAvailable() {
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
		for _, glob := range opts.ignoreGlobs {
			args = append(args, "--glob", glob)
		}
		args = append(args, pattern, opts.Path)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, common.RipgrepPath(), args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		_ = cmd.Run()
		if stdout.Len() == 0 && stderr.Len() > 0 {
			return "", true, nil, fmt.Errorf("regex error: %s", strings.TrimSpace(stderr.String()))
		}
		return stdout.String(), true, nil, nil
	}

	var warnings []string
	args := []string{
		"-rn",
		"-I",
		"-m", strconv.Itoa(maxCountPerFile),
		"--exclude-dir=.git",
		"--exclude-dir=node_modules",
		"--exclude-dir=vendor",
		"--exclude-dir=.next",
	}
	if opts.IsRegex {
		args = append(args, "-E")
	} else {
		args = append(args, "-F")
	}
	if !opts.IncludeHidden {
		if isGNUGrep() {
			args = append(args,
				"--exclude=.[!.]*",
				"--exclude=..?*",
				"--exclude-dir=.[!.]*",
				"--exclude-dir=..?*",
			)
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
	_ = cmd.Run()
	if stdout.Len() == 0 && stderr.Len() > 0 {
		return "", false, warnings, fmt.Errorf("regex error: %s", strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), false, warnings, nil
}
