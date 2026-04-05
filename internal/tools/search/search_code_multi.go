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
		if strings.HasPrefix(trimmed, "📄 ") {
			rest := strings.TrimPrefix(trimmed, "📄 ")
			if idx := strings.Index(rest, " ("); idx > 0 {
				add(rest[:idx])
			}
			continue
		}
		if strings.HasPrefix(trimmed, "── ") && strings.Contains(trimmed, " in ") && strings.HasSuffix(trimmed, "──") {
			inIdx := strings.LastIndex(trimmed, " in ")
			rest := trimmed[inIdx+4:]
			rest = strings.TrimSuffix(rest, "──")
			rest = strings.TrimSpace(rest)
			if atIdx := strings.LastIndex(rest, " @"); atIdx > 0 {
				rest = rest[:atIdx]
			}
			add(rest)
			continue
		}
		if hasNumericListPrefix(trimmed) {
			if idx := strings.LastIndex(trimmed, " in "); idx > 0 {
				add(strings.TrimSpace(trimmed[idx+4:]))
				continue
			}
		}
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
