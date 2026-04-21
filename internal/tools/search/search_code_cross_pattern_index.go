package search

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

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
