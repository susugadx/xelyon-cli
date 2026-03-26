package search

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/navigation"
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

// lineRangeHint は search_code 結果末尾に付与する編集ヒント。
// UI には表示されず、LLM の tool result にのみ含まれる。
const lineRangeHint = "\n\nTip: Use the active edit tool with the matched lines plus surrounding context to make exact edits."

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
	FilePattern    string // file_filter から自動判定。glob 文字を含む場合に設定。
	FileType       string // file_filter から自動判定。glob 文字を含まない場合に設定。
	CtxLines       int    // ツール経路では内部固定値（3）。外部パラメータは廃止。
	TokenBudget    int    // 内部固定値（15000）。外部パラメータは廃止。
	IsRegex        bool
	Multiline      bool   // ツール経路では内部固定値（false）。外部パラメータは廃止。
	IncludeHidden  bool   // ツール経路では内部固定値（false）。外部パラメータは廃止。
	IncludeIgnored bool   // ツール経路では内部固定値（false）。外部パラメータは廃止。
	OutputMode     string // 内部専用。外部パラメータは廃止。

	LocatorRegistry *locator.Registry // Locator ID レジストリ（nilの場合はID付与しない）

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

	// context_lines is fixed internally.
	opts.CtxLines = 3

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

	// ── Symbol fast path ──
	if shouldTrySymbolResolve(pattern, opts) {
		lang := resolveLanguage(opts)

		if lang == "go" {
			// Go: AST ベースの正確な解決
			symbolOutput, status := navigation.InspectSymbolAuto(pattern, opts.Path, opts.LocatorRegistry)
			switch status {
			case navigation.SymbolAutoSingle:
				if cache != nil {
					cache.SetSearch(pattern, cacheKey, symbolOutput, nil)
				}
				return symbolOutput
			case navigation.SymbolAutoMultiple:
				return symbolOutput
			case navigation.SymbolAutoNone:
				// フォールバック
			}
		} else {
			// 他言語: パターンベースのシンボル解決（言語別参照分類付き）
			var symbolOutput string
			var status genericSymbolStatus
			switch lang {
			case "js":
				symbolOutput, status = resolveJSSymbol(pattern, opts)
			case "python":
				symbolOutput, status = resolvePythonSymbol(pattern, opts)
			case "rust":
				symbolOutput, status = resolveRustSymbol(pattern, opts)
			case "java":
				symbolOutput, status = resolveJavaSymbol(pattern, opts)
			case "csharp":
				symbolOutput, status = resolveCSharpSymbol(pattern, opts)
			case "php":
				symbolOutput, status = resolvePHPSymbol(pattern, opts)
			case "ruby":
				symbolOutput, status = resolveRubySymbol(pattern, opts)
			case "swift":
				symbolOutput, status = resolveSwiftSymbol(pattern, opts)
			case "scala":
				symbolOutput, status = resolveScalaSymbol(pattern, opts)
			case "elixir":
				symbolOutput, status = resolveElixirSymbol(pattern, opts)
			case "lua":
				symbolOutput, status = resolveLuaSymbol(pattern, opts)
			case "cpp":
				symbolOutput, status = resolveCppSymbol(pattern, opts)
			default:
				symbolOutput, status = resolveGenericSymbol(pattern, opts)
			}
			switch status {
			case genericSymbolSingle:
				if cache != nil {
					cache.SetSearch(pattern, cacheKey, symbolOutput, nil)
				}
				return symbolOutput
			case genericSymbolMultiple:
				return symbolOutput
			case genericSymbolNone:
				// フォールバック
			}
		}
	}

	output, useRipgrep, warnings, err := executeSearch(pattern, opts)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	var results []SearchResult
	if useRipgrep {
		results = parseRipgrepJSON(output, 0)
	} else {
		results = parseGrepOutput(output, 0)
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
		formatted := formatManifestResults(results, opts.LocatorRegistry)
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
	sortResultsByPriority(results)

	results, truncated := truncateToTokenBudget(results, opts.TokenBudget, false)

	detectBlocksWithCache(cache, results)

	formatted := formatSearchResults(results, truncated, opts.TokenBudget, opts.LocatorRegistry)
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
// \, はリテラルカンマとして扱う。空文字除外、TrimSpace、最大 10 パターン。
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
	if len(result) > 10 {
		result = result[:10]
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

// executeMultiplePatterns は複数パターンを goroutine 並列で検索する。
// 各パターンは executeSinglePattern に委譲し、symbol fast path + キャッシュが自動で効く。
func executeMultiplePatterns(cache tools.ToolCacheInterface, patterns []string, opts SearchOptions) string {
	type formattedResult struct {
		Pattern   string
		Index     int
		Formatted string
	}

	ch := make(chan formattedResult, len(patterns))
	for i, p := range patterns {
		go func(idx int, pat string) {
			result := executeSinglePattern(cache, pat, opts)
			// lineRangeHint は最後に1回だけ付与するため個別結果からは除去
			result = strings.TrimSuffix(result, lineRangeHint)
			ch <- formattedResult{Pattern: pat, Index: idx, Formatted: result}
		}(i, p)
	}

	collected := make([]formattedResult, len(patterns))
	for range patterns {
		r := <-ch
		collected[r.Index] = r
	}

	var sb strings.Builder
	for i, pr := range collected {
		if i > 0 {
			sb.WriteString("\n")
		}
		fmt.Fprintf(&sb, "━━ Pattern %d/%d: %q ━━\n", i+1, len(patterns), pr.Pattern)
		sb.WriteString(pr.Formatted)
		if !strings.HasSuffix(pr.Formatted, "\n") {
			sb.WriteString("\n")
		}
	}

	// Cross-pattern file index
	pats := make([]string, len(collected))
	outs := make([]string, len(collected))
	for i, c := range collected {
		pats[i] = c.Pattern
		outs[i] = c.Formatted
	}
	if idx := buildCrossPatternIndex(pats, outs, opts.LocatorRegistry); idx != "" {
		sb.WriteString(idx)
	}

	output := sb.String() + lineRangeHint

	if cache != nil {
		multiKey := buildMultiCacheKey(patterns)
		cacheKey := buildSearchCacheKey(opts)
		cache.SetSearch(multiKey, cacheKey, output, nil)
	}

	return output
}

// extractPrimaryFilePaths は整形済み検索結果から主要ファイルパスを抽出する。
// 📄 ヘッダー（通常検索）と ── ... in filepath ──（シンボル解決）の両方を認識する。
func extractPrimaryFilePaths(output string) []string {
	var paths []string
	seen := make(map[string]bool)
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p != "" && !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		// Regular search: "📄 path/to/file (N match(es))"
		if strings.HasPrefix(trimmed, "📄 ") {
			rest := strings.TrimPrefix(trimmed, "📄 ")
			if idx := strings.Index(rest, " ("); idx > 0 {
				add(rest[:idx])
			}
			continue
		}
		// Symbol definition: "── kind Name (LN) in filepath ──" or "── kind Name (LN-LN) in filepath @locN ──"
		if strings.HasPrefix(trimmed, "── ") && strings.Contains(trimmed, " in ") && strings.HasSuffix(trimmed, "──") {
			inIdx := strings.LastIndex(trimmed, " in ")
			rest := trimmed[inIdx+4:]
			rest = strings.TrimSuffix(rest, "──")
			rest = strings.TrimSpace(rest)
			// Remove trailing "@locN"
			if atIdx := strings.LastIndex(rest, " @"); atIdx > 0 {
				rest = rest[:atIdx]
			}
			add(rest)
		}
	}
	return paths
}

// classifyFilePath はファイルパスをカテゴリ（impl/test/config）に分類する。
func classifyFilePath(path string) string {
	base := filepath.Base(path)
	if strings.HasSuffix(base, "_test.go") || strings.Contains(base, ".test.") ||
		strings.Contains(base, ".spec.") || strings.HasPrefix(base, "test_") {
		return "test"
	}
	switch filepath.Ext(base) {
	case ".yaml", ".yml", ".toml", ".env", ".ini", ".cfg", ".conf":
		return "config"
	}
	return "impl"
}

// buildCrossPatternIndex は複数パターン検索結果のファイル横断インデックスを生成する。
// パターンごとの出力からファイルパスを抽出し、カテゴリ別に分類して一覧にする。
// 複数パターンに出現するファイルには ★N マークを付与する。
// reg が non-nil の場合、各ファイルに locator ID を付与する。
//
// 出力条件: hotspot（複数パターンにマッチしたファイル）があるか、
// 複数カテゴリに分散しているか、unique files が 3 以上の場合のみ出力する。
func buildCrossPatternIndex(patterns, outputs []string, reg *locator.Registry) string {
	type fileEntry struct {
		patternCount int
		category     string
	}

	fileMap := make(map[string]*fileEntry)
	var order []string

	for i, output := range outputs {
		if i >= len(patterns) {
			break
		}
		for _, p := range extractPrimaryFilePaths(output) {
			if entry, ok := fileMap[p]; ok {
				entry.patternCount++
			} else {
				fileMap[p] = &fileEntry{
					patternCount: 1,
					category:     classifyFilePath(p),
				}
				order = append(order, p)
			}
		}
	}

	if len(order) == 0 {
		return ""
	}

	var impl, test, cfg []string
	for _, p := range order {
		switch fileMap[p].category {
		case "test":
			test = append(test, p)
		case "config":
			cfg = append(cfg, p)
		default:
			impl = append(impl, p)
		}
	}

	// 出力条件: hotspot / 複数カテゴリ / unique files ≥ 3
	hasHotspot := false
	for _, e := range fileMap {
		if e.patternCount > 1 {
			hasHotspot = true
			break
		}
	}
	categoryCount := 0
	if len(impl) > 0 {
		categoryCount++
	}
	if len(test) > 0 {
		categoryCount++
	}
	if len(cfg) > 0 {
		categoryCount++
	}
	if !hasHotspot && categoryCount < 2 && len(order) < 3 {
		return ""
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "\n━━ File Index (%d unique files) ━━\n", len(order))

	writeGroup := func(label string, paths []string) {
		if len(paths) == 0 {
			return
		}
		fmt.Fprintf(&sb, "%s:\n", label)
		for _, p := range paths {
			e := fileMap[p]
			var line string
			if e.patternCount > 1 {
				line = fmt.Sprintf("  %s (★%d patterns)", p, e.patternCount)
			} else {
				line = fmt.Sprintf("  %s", p)
			}
			if reg != nil {
				id := reg.Register(locator.Location{FilePath: p})
				line += " " + id
			}
			fmt.Fprintf(&sb, "%s\n", line)
		}
	}

	writeGroup("Impl", impl)
	writeGroup("Test", test)
	writeGroup("Config", cfg)

	return sb.String()
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

// executeSearch は rg（優先）または grep を実行し、出力と使用ツールを返す
func executeSearch(pattern string, opts SearchOptions) (string, bool, []string, error) {
	if common.IsRipgrepAvailable() {
		args := []string{
			"--json",
			"-n",
		}
		if opts.CtxLines > 0 {
			args = append(args, "--context", fmt.Sprintf("%d", opts.CtxLines))
		}
		if opts.FileType != "" {
			args = append(args, "--type", normalizeRgType(opts.FileType))
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
			warnings = append(warnings, fmt.Sprintf("Warning: file_filter=%q is not supported in grep fallback mode as a language type (rg not found)", opts.FileType))
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
		args = append(args, "-C", fmt.Sprintf("%d", opts.CtxLines))
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
