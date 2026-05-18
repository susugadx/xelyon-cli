package search

import (
	"path/filepath"
	"strings"
)

type structuredJavaScriptImpactTarget struct {
	suffix   string
	fileType string
}

var structuredJavaScriptImpactTargets = []structuredJavaScriptImpactTarget{
	{suffix: ".jsx", fileType: "jsx"},
	{suffix: ".js", fileType: "js"},
}

func structuredJavaScriptImpactTargetForPath(path string) (structuredJavaScriptImpactTarget, bool) {
	clean := strings.ToLower(cleanStructuredJavaScriptDisplayPath(path))
	return structuredJavaScriptImpactTargetForCleanPath(clean)
}

func structuredJavaScriptImpactTargetForFileType(fileType string) (structuredJavaScriptImpactTarget, bool) {
	fileType = strings.ToLower(strings.TrimSpace(fileType))
	for _, target := range structuredJavaScriptImpactTargets {
		if target.fileType == fileType {
			return target, true
		}
	}
	return structuredJavaScriptImpactTarget{}, false
}

func structuredJavaScriptImpactTargetForFilePattern(pattern string) (structuredJavaScriptImpactTarget, bool) {
	pattern = strings.ToLower(cleanStructuredJavaScriptFilePattern(pattern))
	if pattern == "" {
		return structuredJavaScriptImpactTarget{}, false
	}
	return structuredJavaScriptImpactTargetForCleanPath(pattern)
}

func structuredJavaScriptImpactTargetForCleanPath(path string) (structuredJavaScriptImpactTarget, bool) {
	if path == "" {
		return structuredJavaScriptImpactTarget{}, false
	}
	for _, target := range structuredJavaScriptImpactTargets {
		if strings.HasSuffix(path, target.suffix) {
			return target, true
		}
	}
	return structuredJavaScriptImpactTarget{}, false
}

func structuredJavaScriptImpactAllowsFilePattern(pattern string) bool {
	_, ok := structuredJavaScriptImpactTargetForFilePattern(pattern)
	return ok
}

func (target structuredJavaScriptImpactTarget) nearbyTestCandidatePaths(defFile string) []string {
	if target.suffix == "" {
		return nil
	}
	cleanFile := filepath.ToSlash(filepath.Clean(defFile))
	dir := filepath.ToSlash(filepath.Dir(cleanFile))
	base := javaScriptImpactBaseNameForTarget(cleanFile, target)

	candidates := []string{
		filepath.ToSlash(filepath.Join(dir, base+".test"+target.suffix)),
		filepath.ToSlash(filepath.Join(dir, base+".spec"+target.suffix)),
		filepath.ToSlash(filepath.Join(dir, "__tests__", base+".test"+target.suffix)),
		filepath.ToSlash(filepath.Join(dir, "__tests__", base+".spec"+target.suffix)),
		filepath.ToSlash(filepath.Join("tests", base+".test"+target.suffix)),
		filepath.ToSlash(filepath.Join("tests", base+".spec"+target.suffix)),
	}
	return dedupeStringList(candidates)
}

func (target structuredJavaScriptImpactTarget) matchesNearbyTestPath(path string) bool {
	if target.suffix == "" {
		return false
	}
	lower := strings.ToLower(filepath.ToSlash(filepath.Clean(path)))
	return strings.HasSuffix(lower, ".test"+target.suffix) || strings.HasSuffix(lower, ".spec"+target.suffix)
}

func javaScriptImpactBaseNameForTarget(path string, target structuredJavaScriptImpactTarget) string {
	base := filepath.Base(path)
	lowerBase := strings.ToLower(base)
	if target.suffix != "" && strings.HasSuffix(lowerBase, target.suffix) {
		return base[:len(base)-len(target.suffix)]
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
}
