package search

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

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

	if idx := buildCrossPatternIndexFromExecutions(collected, opts.LocatorRegistry, opts); idx != "" {
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

func buildCrossPatternIndex(patterns, outputs []string, reg *locator.Registry) string {
	return buildCrossPatternIndexWithOptions(patterns, outputs, reg, SearchOptions{})
}

func buildCrossPatternIndexWithOptions(patterns, outputs []string, reg *locator.Registry, opts SearchOptions) string {
	collected := make([]formattedPatternExecution, 0, min(len(patterns), len(outputs)))
	for i, output := range outputs {
		if i >= len(patterns) {
			break
		}
		collected = append(collected, formattedPatternExecution{
			Index: i,
			singlePatternExecution: singlePatternExecution{
				Pattern: patterns[i],
				Output:  output,
			},
		})
	}
	return buildCrossPatternIndexFromExecutions(collected, reg, opts)
}

func buildCrossPatternIndexFromExecutions(collected []formattedPatternExecution, reg *locator.Registry, opts SearchOptions) string {
	type fileEntry struct {
		ref          primaryFileRef
		patternCount int
		category     string
	}

	fileMap := make(map[string]*fileEntry)
	var order []string

	for _, execution := range collected {
		for _, ref := range primaryFileRefsFromExecution(execution, opts) {
			key := ref.DisplayPath + "\x00" + ref.ResolvedPath
			if entry, ok := fileMap[key]; ok {
				entry.patternCount++
			} else {
				fileMap[key] = &fileEntry{
					ref:          ref,
					patternCount: 1,
					category:     classifyFilePath(ref.DisplayPath),
				}
				order = append(order, key)
			}
		}
	}

	if len(order) == 0 {
		return ""
	}

	var impl, test, cfg []string
	for _, key := range order {
		switch fileMap[key].category {
		case "test":
			test = append(test, key)
		case "config":
			cfg = append(cfg, key)
		default:
			impl = append(impl, key)
		}
	}

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

	writeGroup := func(label string, keys []string) {
		if len(keys) == 0 {
			return
		}
		fmt.Fprintf(&sb, "%s:\n", label)
		for _, key := range keys {
			e := fileMap[key]
			p := e.ref.DisplayPath
			var line string
			if e.patternCount > 1 {
				line = fmt.Sprintf("  %s (★%d patterns)", p, e.patternCount)
			} else {
				line = fmt.Sprintf("  %s", p)
			}
			if reg != nil {
				id := reg.Register(newSearchLocator(p, e.ref.ResolvedPath, 0, 0, ""))
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

func primaryFileRefsFromExecution(execution formattedPatternExecution, opts SearchOptions) []primaryFileRef {
	return primaryFileRefsFromBundleOrOutput(execution.Bundle, execution.Output, opts)
}

func primaryFileRefsFromBundleOrOutput(bundle *SymbolBundle, output string, opts SearchOptions) []primaryFileRef {
	if ref, ok := primaryFileRefFromBundle(bundle); ok {
		return []primaryFileRef{ref}
	}
	return extractPrimaryFileRefs(output, opts)
}

func primaryFileRefFromBundle(bundle *SymbolBundle) (primaryFileRef, bool) {
	if bundle == nil {
		return primaryFileRef{}, false
	}
	displayPath := strings.TrimSpace(bundle.Identity.File)
	if displayPath == "" {
		return primaryFileRef{}, false
	}
	return primaryFileRef{
		DisplayPath:  displayPath,
		ResolvedPath: cleanResolvedLocatorPath(absoluteAffectedFilePathWithBase(displayPath, bundle.Debug.FileRootPath)),
	}, true
}

func buildMultiCacheKey(patterns []string) string {
	sorted := make([]string, len(patterns))
	copy(sorted, patterns)
	sort.Strings(sorted)
	return strings.Join(sorted, "|")
}

func buildSearchCacheKeyWithRoute(opts SearchOptions, routeSignature string) string {
	return fmt.Sprintf("%s|%s|%s|%d|%d|intent=%s|mode=%s|regex=%t|multiline=%t|hidden=%t|ignored=%t|output=%s|ignore=%s|route=%s",
		opts.Path, opts.FilePattern, opts.FileType, opts.CtxLines, opts.TokenBudget, strings.TrimSpace(opts.Intent), opts.Mode, opts.IsRegex, opts.Multiline, opts.IncludeHidden, opts.IncludeIgnored, opts.OutputMode, opts.ignoreKey, routeSignature)
}

func buildMultiSearchCacheKey(opts SearchOptions, patterns []string) string {
	signatures := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		signatures = append(signatures, planSearchRoute(pattern, opts).cacheSignature())
	}
	sort.Strings(signatures)
	return buildSearchCacheKeyWithRoute(opts, strings.Join(signatures, ";"))
}
