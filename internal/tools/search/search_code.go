package search

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/repomap"
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
	Pattern          string
	Intent           string
	Mode             string
	Path             string
	FilePattern      string // file_filter から自動判定。glob 文字を含む場合に設定。
	FileType         string // file_filter から自動判定。glob 文字を含まない場合に設定。
	CtxLines         int    // ツール経路では内部固定値（3）。外部パラメータは廃止。
	TokenBudget      int    // 内部固定値（15000）。外部パラメータは廃止。
	IsRegex          bool
	Multiline        bool   // ツール経路では内部固定値（false）。外部パラメータは廃止。
	IncludeHidden    bool   // ツール経路では内部固定値（false）。外部パラメータは廃止。
	IncludeIgnored   bool   // ツール経路では内部固定値（false）。外部パラメータは廃止。
	OutputMode       string // 内部専用。外部パラメータは廃止。
	LegacyIsRegexSet bool

	LocatorRegistry    *locator.Registry // Locator ID レジストリ（nilの場合はID付与しない）
	LSPClient          navigation.LSPClient
	ProjectMap         *repomap.ProjectMap
	ProjectMapRootPath string
	ProjectMapStateKey string
	InvocationCWD      string

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
	var ok bool
	opts, ok = normalizeSearchOptions(opts)
	if !ok {
		return "Error: invalid mode (expected auto, symbol, literal, or regex)"
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

	if shouldExecuteImpactSearch(opts) {
		return executeImpactSearch(cache, opts)
	}
	patterns := effectiveSearchPatterns(opts)
	return executeSearchPatterns(cache, patterns, opts)
}

// impactPatternExpansionCap keeps intent=impact conservative and aligned with
// the existing search_code batch-merge size used by the agent optimizer.
const impactPatternExpansionCap = 5

// executeSinglePattern は単一パターンの検索処理（キャッシュ・検索・パース・マージ・トランケート・ブロック認識・フォーマット・キャッシュ保存）
func executeSinglePattern(cache tools.ToolCacheInterface, pattern string, opts SearchOptions) string {
	return executeSinglePatternDetailed(cache, pattern, opts).Output
}

func executeSinglePatternWithTrace(cache tools.ToolCacheInterface, pattern string, opts SearchOptions) (string, searchRouteTrace) {
	result := executeSinglePatternDetailed(cache, pattern, opts)
	return result.Output, result.Route
}

type singlePatternExecution struct {
	Pattern       string
	Output        string
	Route         searchRouteTrace
	Bundle        *SymbolBundle
	AffectedFiles []string
}

var singlePatternBundleCache sync.Map
var singlePatternAffectedFilesCache sync.Map

func init() {
	tools.RegisterSearchCacheLifecycleHooks(clearSinglePatternBundleCache, invalidateSinglePatternBundleCacheKeys, invalidateSinglePatternBundleCacheKeys)
}

func singlePatternBundleCacheKey(pattern, cacheKey string) string {
	return pattern + "::" + cacheKey
}

func clearSinglePatternBundleCache() {
	singlePatternBundleCache.Range(func(key, value any) bool {
		singlePatternBundleCache.Delete(key)
		return true
	})
	singlePatternAffectedFilesCache.Range(func(key, value any) bool {
		singlePatternAffectedFilesCache.Delete(key)
		return true
	})
}

func invalidateSinglePatternBundleCacheKeys(keys []string) {
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			continue
		}
		singlePatternBundleCache.Delete(key)
		singlePatternAffectedFilesCache.Delete(key)
	}
}

func loadSinglePatternBundle(pattern, cacheKey string) *SymbolBundle {
	value, ok := singlePatternBundleCache.Load(singlePatternBundleCacheKey(pattern, cacheKey))
	if !ok {
		return nil
	}
	bundle, _ := value.(*SymbolBundle)
	return cloneSymbolBundle(bundle)
}

func storeSinglePatternBundle(pattern, cacheKey string, bundle *SymbolBundle) {
	if bundle == nil {
		return
	}
	singlePatternBundleCache.Store(singlePatternBundleCacheKey(pattern, cacheKey), cloneSymbolBundle(bundle))
}

func loadSinglePatternAffectedFiles(pattern, cacheKey string) []string {
	value, ok := singlePatternAffectedFilesCache.Load(singlePatternBundleCacheKey(pattern, cacheKey))
	if !ok {
		return nil
	}
	paths, _ := value.([]string)
	return append([]string(nil), paths...)
}

func storeSinglePatternAffectedFiles(pattern, cacheKey string, affectedFiles []string) {
	if len(affectedFiles) == 0 {
		return
	}
	singlePatternAffectedFilesCache.Store(singlePatternBundleCacheKey(pattern, cacheKey), append([]string(nil), affectedFiles...))
}

func cloneSymbolBundle(bundle *SymbolBundle) *SymbolBundle {
	if bundle == nil {
		return nil
	}
	cloned := *bundle
	if bundle.Definition.Body != nil {
		cloned.Definition.Body = append([]string(nil), bundle.Definition.Body...)
	}
	if bundle.Sections != nil {
		cloned.Sections = make([]SymbolBundleSection, len(bundle.Sections))
		for i := range bundle.Sections {
			cloned.Sections[i] = bundle.Sections[i]
			if bundle.Sections[i].Items != nil {
				cloned.Sections[i].Items = append([]SymbolBundleItem(nil), bundle.Sections[i].Items...)
			}
		}
	}
	if bundle.Debug.MatchedPatterns != nil {
		cloned.Debug.MatchedPatterns = append([]string(nil), bundle.Debug.MatchedPatterns...)
	}
	return &cloned
}

func executeSinglePatternDetailed(cache tools.ToolCacheInterface, pattern string, opts SearchOptions) singlePatternExecution {
	route := planSearchRoute(pattern, opts)
	cacheKey := buildSearchCacheKeyWithRoute(opts, route.cacheSignature())
	if cache != nil {
		if cached, ok := cache.GetSearch(pattern, cacheKey); ok {
			bundle := loadSinglePatternBundle(pattern, cacheKey)
			affectedFiles := loadSinglePatternAffectedFiles(pattern, cacheKey)
			if len(affectedFiles) == 0 {
				affectedFiles = deriveAffectedFilesFromCachedResult(bundle, cached, opts)
			}
			return singlePatternExecution{
				Pattern:       pattern,
				Output:        cached,
				Route:         route,
				Bundle:        bundle,
				AffectedFiles: affectedFiles,
			}
		}
	}

	if route.InitialLane == searchLaneSymbol {
		route.SymbolAttempted = true
		resolver := resolverForLanguage(route.Language)
		if resolver != nil {
			resolved := resolver.Resolve(route.SymbolQuery, opts)
			switch resolved.Status {
			case symbolResolveSingle:
				route.FinalLane = searchLaneSymbol
				route.SymbolResolved = true
				resolved.Bundle = attachBundleRoute(resolved.Bundle, route)
				affectedFiles := collectSymbolBundleAffectedFiles(resolved.Bundle, opts)
				if cache != nil {
					cache.SetSearch(pattern, cacheKey, resolved.Output, affectedFiles)
					storeSinglePatternBundle(pattern, cacheKey, resolved.Bundle)
					storeSinglePatternAffectedFiles(pattern, cacheKey, affectedFiles)
				}
				return singlePatternExecution{
					Pattern:       pattern,
					Output:        resolved.Output,
					Route:         route,
					Bundle:        resolved.Bundle,
					AffectedFiles: affectedFiles,
				}
			case symbolResolveMultiple:
				route.FinalLane = searchLaneSymbol
				route.SymbolResolved = true
				affectedFiles := resolved.AffectedFiles
				if len(affectedFiles) == 0 {
					affectedFiles = deriveAffectedFilesFromCachedResult(nil, resolved.Output, opts)
				}
				if cache != nil {
					cache.SetSearch(pattern, cacheKey, resolved.Output, affectedFiles)
					storeSinglePatternAffectedFiles(pattern, cacheKey, affectedFiles)
				}
				return singlePatternExecution{
					Pattern:       pattern,
					Output:        resolved.Output,
					Route:         route,
					AffectedFiles: affectedFiles,
				}
			case symbolResolveNone:
				route.SymbolResolved = false
			}
		}
		if route.FallbackLane != "" {
			route.FallbackUsed = true
			route.FinalLane = route.FallbackLane
		}
	}
	if route.FinalLane == "" {
		route.FinalLane = route.InitialLane
	}

	textOpts := opts
	textOpts.IsRegex = route.textIsRegex()
	output, useRipgrep, warnings, err := executeSearch(pattern, textOpts)
	if err != nil {
		return singlePatternExecution{Pattern: pattern, Output: fmt.Sprintf("Error: %v", err), Route: route}
	}

	var results []SearchResult
	if useRipgrep {
		results = parseRipgrepJSON(output, 0)
	} else {
		results = parseGrepOutput(output, 0)
	}
	results = filterResultsByOptions(results, textOpts)
	reclassifyWithAST(results, pattern, textOpts.IsRegex)

	if len(results) == 0 {
		if len(warnings) > 0 {
			return singlePatternExecution{Pattern: pattern, Output: strings.Join(warnings, "\n") + "\nNo matches found", Route: route}
		}
		return singlePatternExecution{Pattern: pattern, Output: "No matches found", Route: route}
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
			affectedFiles := collectFilePaths(results, opts)
			cache.SetSearch(pattern, cacheKey, finalOutput, affectedFiles)
			storeSinglePatternAffectedFiles(pattern, cacheKey, affectedFiles)
			return singlePatternExecution{Pattern: pattern, Output: finalOutput, Route: route, AffectedFiles: affectedFiles}
		}
		return singlePatternExecution{Pattern: pattern, Output: finalOutput, Route: route, AffectedFiles: collectFilePaths(results, opts)}
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

	affectedFiles := collectFilePaths(results, opts)
	if cache != nil {
		cache.SetSearch(pattern, cacheKey, finalOutput, affectedFiles)
		storeSinglePatternAffectedFiles(pattern, cacheKey, affectedFiles)
	}

	return singlePatternExecution{Pattern: pattern, Output: finalOutput, Route: route, AffectedFiles: affectedFiles}
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

type formattedPatternExecution struct {
	Index int
	singlePatternExecution
}

// executeMultiplePatterns は複数パターンを goroutine 並列で検索する。
// 各パターンは executeSinglePattern に委譲し、symbol fast path + キャッシュが自動で効く。
func executeMultiplePatterns(cache tools.ToolCacheInterface, patterns []string, opts SearchOptions) string {
	ch := make(chan formattedPatternExecution, len(patterns))
	for i, p := range patterns {
		go func(idx int, pat string) {
			result := executeSinglePatternDetailed(cache, pat, opts)
			// lineRangeHint は最後に1回だけ付与するため個別結果からは除去
			result.Output = strings.TrimSuffix(result.Output, lineRangeHint)
			ch <- formattedPatternExecution{Index: idx, singlePatternExecution: result}
		}(i, p)
	}

	collected := make([]formattedPatternExecution, len(patterns))
	for range patterns {
		r := <-ch
		collected[r.Index] = r
	}

	var sb strings.Builder
	grouped := groupPatternSymbolBundles(collected)
	for i, pr := range collected {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		if pr.Bundle != nil {
			if group, ok := grouped[pr.Bundle.Identity.Canonical]; ok && len(group.Patterns) > 1 {
				if group.Emitted {
					continue
				}
				group.Emitted = true
				grouped[pr.Bundle.Identity.Canonical] = group
				fmt.Fprintf(&sb, "━━ Symbol Bundle: %q ━━\n", group.Bundle.Identity.DisplayName)
				sb.WriteString(formatSymbolBundle(group.Bundle, opts.LocatorRegistry, group.Patterns))
				if !strings.HasSuffix(sb.String(), "\n") {
					sb.WriteString("\n")
				}
				continue
			}
		}
		fmt.Fprintf(&sb, "━━ Pattern %d/%d: %q ━━\n", i+1, len(patterns), pr.Pattern)
		sb.WriteString(pr.Output)
		if !strings.HasSuffix(pr.Output, "\n") {
			sb.WriteString("\n")
		}
	}

	// Cross-pattern file index
	pats := make([]string, len(collected))
	outs := make([]string, len(collected))
	for i, c := range collected {
		pats[i] = c.Pattern
		outs[i] = c.Output
	}
	if idx := buildCrossPatternIndex(pats, outs, opts.LocatorRegistry); idx != "" {
		sb.WriteString(idx)
	}

	output := sb.String() + lineRangeHint

	if cache != nil {
		multiKey := buildMultiCacheKey(patterns)
		cacheKey := buildMultiSearchCacheKey(opts, patterns)
		affectedFiles := collectAffectedFilesFromExecutions(collected, opts)
		cache.SetSearch(multiKey, cacheKey, output, affectedFiles)
	}

	return output
}

type patternSymbolBundleGroup struct {
	Bundle   *SymbolBundle
	Patterns []string
	Emitted  bool
}

func groupPatternSymbolBundles(collected []formattedPatternExecution) map[string]patternSymbolBundleGroup {
	groups := make(map[string]patternSymbolBundleGroup)
	for _, item := range collected {
		if item.Bundle == nil {
			continue
		}
		key := item.Bundle.Identity.Canonical
		group := groups[key]
		if group.Bundle == nil {
			group.Bundle = item.Bundle
		}
		group.Patterns = appendPatternIfMissing(group.Patterns, item.Pattern)
		for _, candidate := range item.Route.SymbolCandidates {
			group.Patterns = appendPatternIfMissing(group.Patterns, candidate)
		}
		groups[key] = group
	}
	return groups
}

func appendPatternIfMissing(patterns []string, pattern string) []string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return patterns
	}
	for _, existing := range patterns {
		if existing == pattern {
			return patterns
		}
	}
	return append(patterns, pattern)
}

func effectiveSearchPatterns(opts SearchOptions) []string {
	patterns := splitPatterns(opts.Pattern)
	if len(patterns) != 1 {
		return patterns
	}
	if !strings.EqualFold(strings.TrimSpace(opts.Intent), "impact") {
		return patterns
	}
	return expandImpactPatterns(patterns[0], opts)
}

// expandImpactPatterns conservatively widens a single-symbol search into a
// small related set so shared-change discovery can reuse multi-pattern search.
func expandImpactPatterns(pattern string, opts SearchOptions) []string {
	_ = opts
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil
	}

	variants := []string{
		pattern,
		pattern + "Impl",
	}

	seen := make(map[string]struct{}, len(variants))
	expanded := make([]string, 0, minInt(len(variants), impactPatternExpansionCap))
	for _, candidate := range variants {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		expanded = append(expanded, candidate)
		if len(expanded) >= impactPatternExpansionCap {
			break
		}
	}

	return expanded
}

func shouldExecuteImpactSearch(opts SearchOptions) bool {
	return strings.EqualFold(strings.TrimSpace(opts.Intent), "impact") && len(splitPatterns(opts.Pattern)) == 1
}

func executeImpactSearch(cache tools.ToolCacheInterface, opts SearchOptions) string {
	basePatterns := expandImpactPatterns(strings.TrimSpace(opts.Pattern), opts)
	if len(basePatterns) == 0 {
		return "Error: pattern is required"
	}

	baseOutput := executeSearchPatterns(cache, basePatterns, opts)
	if impactOutputHasTestCoverage(baseOutput) || len(basePatterns) >= impactPatternExpansionCap {
		return baseOutput
	}

	testProbe := impactTestProbePattern(opts.Pattern)
	if testProbe == "" {
		return baseOutput
	}
	for _, existing := range basePatterns {
		if existing == testProbe {
			return baseOutput
		}
	}

	finalPatterns := append(append([]string(nil), basePatterns...), testProbe)
	return executeSearchPatterns(cache, finalPatterns, opts)
}

func executeSearchPatterns(cache tools.ToolCacheInterface, patterns []string, opts SearchOptions) string {
	if len(patterns) > 1 {
		multiKey := buildMultiCacheKey(patterns)
		cacheKey := buildMultiSearchCacheKey(opts, patterns)
		if cache != nil {
			if cached, ok := cache.GetSearch(multiKey, cacheKey); ok {
				return cached
			}
		}
		return executeMultiplePatterns(cache, patterns, opts)
	}
	return executeSinglePattern(cache, patterns[0], opts)
}

func impactOutputHasTestCoverage(output string) bool {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "_test.go") || strings.Contains(trimmed, ".test.") || strings.Contains(trimmed, "Tests (") {
			return true
		}
	}
	return false
}

func impactTestProbePattern(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return ""
	}
	runes := []rune(pattern)
	runes[0] = unicode.ToUpper(runes[0])
	return "Test" + string(runes)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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
			continue
		}
		// Multiple generic definitions: "1. kind Name (L10) in path/to/file"
		if hasNumericListPrefix(trimmed) {
			if idx := strings.LastIndex(trimmed, " in "); idx > 0 {
				add(strings.TrimSpace(trimmed[idx+4:]))
				continue
			}
		}
		// Multiple Go symbol candidates: "1. path/to/file kind Symbol (L10-L12)"
		if numbered, ok := parseNumberedCandidateFilePath(trimmed); ok {
			add(numbered)
		}
	}
	return paths
}

func parseNumberedCandidateFilePath(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}
	dotIdx := strings.Index(line, ".")
	if dotIdx <= 0 {
		return "", false
	}
	for _, r := range line[:dotIdx] {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	rest := strings.TrimSpace(line[dotIdx+1:])
	if rest == "" {
		return "", false
	}
	if idx := strings.Index(rest, " function "); idx > 0 {
		return strings.TrimSpace(rest[:idx]), true
	}
	if idx := strings.Index(rest, " method "); idx > 0 {
		return strings.TrimSpace(rest[:idx]), true
	}
	if idx := strings.Index(rest, " type "); idx > 0 {
		return strings.TrimSpace(rest[:idx]), true
	}
	if idx := strings.Index(rest, " interface "); idx > 0 {
		return strings.TrimSpace(rest[:idx]), true
	}
	if idx := strings.Index(rest, " const "); idx > 0 {
		return strings.TrimSpace(rest[:idx]), true
	}
	if idx := strings.Index(rest, " var "); idx > 0 {
		return strings.TrimSpace(rest[:idx]), true
	}
	return "", false
}

func hasNumericListPrefix(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	dotIdx := strings.Index(line, ".")
	if dotIdx <= 0 {
		return false
	}
	for _, r := range line[:dotIdx] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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

func buildSearchCacheKeyWithRoute(opts SearchOptions, routeSignature string) string {
	return fmt.Sprintf("%s|%s|%s|%d|%d|mode=%s|regex=%t|multiline=%t|hidden=%t|ignored=%t|output=%s|ignore=%s|route=%s",
		opts.Path, opts.FilePattern, opts.FileType, opts.CtxLines, opts.TokenBudget, opts.Mode, opts.IsRegex, opts.Multiline, opts.IncludeHidden, opts.IncludeIgnored, opts.OutputMode, opts.ignoreKey, routeSignature)
}

func buildMultiSearchCacheKey(opts SearchOptions, patterns []string) string {
	signatures := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		signatures = append(signatures, planSearchRoute(pattern, opts).cacheSignature())
	}
	sort.Strings(signatures)
	return buildSearchCacheKeyWithRoute(opts, strings.Join(signatures, ";"))
}

func collectFilePaths(results []SearchResult, opts SearchOptions) []string {
	paths := make([]string, 0, len(results))
	for _, r := range results {
		if r.FilePath == "" {
			continue
		}
		if absPath := absoluteAffectedFilePath(r.FilePath, opts, affectedFileSourceText); absPath != "" {
			paths = append(paths, absPath)
		}
	}
	return dedupePaths(paths)
}

func collectAffectedFilesFromExecutions(collected []formattedPatternExecution, opts SearchOptions) []string {
	paths := make([]string, 0, len(collected)*2)
	var outputs []string
	for _, execution := range collected {
		paths = append(paths, execution.AffectedFiles...)
		outputs = append(outputs, execution.Output)
	}
	paths = append(paths, collectPrimaryFilePathsFromOutputs(outputs, opts)...)
	return dedupePaths(paths)
}

func deriveAffectedFilesFromCachedResult(bundle *SymbolBundle, output string, opts SearchOptions) []string {
	if affected := collectSymbolBundleAffectedFiles(bundle, opts); len(affected) > 0 {
		return affected
	}
	return collectPrimaryFilePathsFromOutputs([]string{output}, opts)
}

func collectSymbolBundleAffectedFiles(bundle *SymbolBundle, opts SearchOptions) []string {
	if bundle == nil {
		return nil
	}

	paths := make([]string, 0, 1+len(bundle.Sections))
	rootPath := strings.TrimSpace(bundle.Debug.FileRootPath)
	add := func(file string) {
		if absPath := absoluteAffectedFilePathForSymbol(file, opts, rootPath); absPath != "" {
			paths = append(paths, absPath)
		}
	}

	add(bundle.Definition.File)
	for _, section := range bundle.Sections {
		for _, item := range section.Items {
			add(item.File)
		}
	}
	return dedupePaths(paths)
}

func collectPrimaryFilePathsFromOutputs(outputs []string, opts SearchOptions) []string {
	var paths []string
	for _, output := range outputs {
		for _, file := range extractPrimaryFilePaths(output) {
			if absPath := absoluteAffectedFilePath(file, opts, affectedFileSourceText); absPath != "" {
				paths = append(paths, absPath)
			}
		}
	}
	return dedupePaths(paths)
}

type affectedFileSource int

const (
	affectedFileSourceText affectedFileSource = iota
	affectedFileSourceSymbol
)

func absoluteAffectedFilePath(file string, opts SearchOptions, source affectedFileSource) string {
	return absoluteAffectedFilePathWithBase(file, affectedFileBasePath(opts, source))
}

func absoluteAffectedFilePathForSymbol(file string, opts SearchOptions, rootPath string) string {
	rootPath = strings.TrimSpace(rootPath)
	if rootPath != "" {
		return absoluteAffectedFilePathWithBase(file, rootPath)
	}
	return absoluteAffectedFilePath(file, opts, affectedFileSourceSymbol)
}

func absoluteAffectedFilePathWithBase(file, basePath string) string {
	file = strings.TrimSpace(file)
	if file == "" {
		return ""
	}
	if filepath.IsAbs(file) {
		return filepath.Clean(file)
	}

	basePath = strings.TrimSpace(basePath)
	if basePath != "" {
		return filepath.Clean(filepath.Join(basePath, filepath.FromSlash(file)))
	}

	if absPath, err := filepath.Abs(file); err == nil {
		return filepath.Clean(absPath)
	}
	return filepath.Clean(file)
}

func affectedFileBasePath(opts SearchOptions, source affectedFileSource) string {
	switch source {
	case affectedFileSourceSymbol:
		if root := strings.TrimSpace(opts.ProjectMapRootPath); root != "" {
			if abs, err := filepath.Abs(root); err == nil {
				return abs
			}
			return filepath.Clean(root)
		}
	}
	return invocationCWDOrGetwd(opts)
}

func invocationCWDOrGetwd(opts SearchOptions) string {
	if cwd := strings.TrimSpace(opts.InvocationCWD); cwd != "" {
		if abs, err := filepath.Abs(cwd); err == nil {
			return abs
		}
		return filepath.Clean(cwd)
	}
	if cwd, err := os.Getwd(); err == nil {
		if abs, err := filepath.Abs(cwd); err == nil {
			return abs
		}
		return filepath.Clean(cwd)
	}
	return ""
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
