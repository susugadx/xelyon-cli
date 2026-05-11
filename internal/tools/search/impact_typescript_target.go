package search

import (
	"path/filepath"
	"strings"
)

type structuredTypeScriptNearbyTestLocation int

const (
	structuredTypeScriptNearbyTestSibling structuredTypeScriptNearbyTestLocation = iota
	structuredTypeScriptNearbyTestNestedTestsDir
	structuredTypeScriptNearbyTestWorkspaceTestsDir
)

type structuredTypeScriptNearbyTestCandidate struct {
	location structuredTypeScriptNearbyTestLocation
	suffix   string
}

type structuredTypeScriptImpactTarget struct {
	suffix                         string
	fileType                       string
	structuredImpact               bool
	implementation                 bool
	declaration                    bool
	pairedImplementationSuffixes   []string
	nearbyTestCandidateDefinitions []structuredTypeScriptNearbyTestCandidate
}

var (
	structuredTypeScriptDeclarationImpactTarget = structuredTypeScriptImpactTarget{
		suffix:                       ".d.ts",
		structuredImpact:             true,
		declaration:                  true,
		pairedImplementationSuffixes: []string{".ts"},
	}
	structuredTypeScriptTSXImpactTarget = structuredTypeScriptImpactTarget{
		suffix:           ".tsx",
		fileType:         "tsx",
		implementation:   true,
		structuredImpact: false,
		nearbyTestCandidateDefinitions: []structuredTypeScriptNearbyTestCandidate{
			{location: structuredTypeScriptNearbyTestSibling, suffix: ".test.tsx"},
			{location: structuredTypeScriptNearbyTestSibling, suffix: ".spec.tsx"},
			{location: structuredTypeScriptNearbyTestNestedTestsDir, suffix: ".test.tsx"},
			{location: structuredTypeScriptNearbyTestWorkspaceTestsDir, suffix: ".test.tsx"},
		},
	}
	structuredTypeScriptImplementationImpactTarget = structuredTypeScriptImpactTarget{
		suffix:           ".ts",
		fileType:         "ts",
		structuredImpact: true,
		implementation:   true,
		nearbyTestCandidateDefinitions: []structuredTypeScriptNearbyTestCandidate{
			{location: structuredTypeScriptNearbyTestSibling, suffix: ".test.ts"},
			{location: structuredTypeScriptNearbyTestSibling, suffix: ".spec.ts"},
			{location: structuredTypeScriptNearbyTestNestedTestsDir, suffix: ".test.ts"},
			{location: structuredTypeScriptNearbyTestWorkspaceTestsDir, suffix: ".test.ts"},
		},
	}
)

var structuredTypeScriptImpactTargets = []structuredTypeScriptImpactTarget{
	structuredTypeScriptDeclarationImpactTarget,
	structuredTypeScriptTSXImpactTarget,
	structuredTypeScriptImplementationImpactTarget,
}

func structuredTypeScriptImpactTargetForPath(path string) (structuredTypeScriptImpactTarget, bool) {
	clean := strings.ToLower(cleanStructuredTypeScriptDisplayPath(path))
	return structuredTypeScriptImpactTargetForCleanPath(clean)
}

func structuredTypeScriptImpactTargetForFileType(fileType string) (structuredTypeScriptImpactTarget, bool) {
	fileType = strings.ToLower(strings.TrimSpace(fileType))
	for _, target := range structuredTypeScriptImpactTargets {
		if target.fileType != "" && target.fileType == fileType {
			return target, true
		}
	}
	return structuredTypeScriptImpactTarget{}, false
}

func structuredTypeScriptImpactTargetForFilePattern(pattern string) (structuredTypeScriptImpactTarget, bool) {
	pattern = strings.ToLower(cleanStructuredTypeScriptFilePattern(pattern))
	if pattern == "" {
		return structuredTypeScriptImpactTarget{}, false
	}
	if strings.Contains(pattern, structuredTypeScriptTSXImpactTarget.suffix) {
		return structuredTypeScriptTSXImpactTarget, true
	}
	return structuredTypeScriptImpactTargetForCleanPath(pattern)
}

func structuredTypeScriptImpactTargetForCleanPath(path string) (structuredTypeScriptImpactTarget, bool) {
	if path == "" {
		return structuredTypeScriptImpactTarget{}, false
	}
	for _, target := range structuredTypeScriptImpactTargets {
		if strings.HasSuffix(path, target.suffix) {
			return target, true
		}
	}
	return structuredTypeScriptImpactTarget{}, false
}

func structuredTypeScriptSourceTargetForPath(path string) (structuredTypeScriptImpactTarget, bool) {
	target, ok := structuredTypeScriptImpactTargetForPath(path)
	if !ok || !target.structuredImpact {
		return structuredTypeScriptImpactTarget{}, false
	}
	return target, true
}

func structuredTypeScriptImplementationTargetForPath(path string) (structuredTypeScriptImpactTarget, bool) {
	target, ok := structuredTypeScriptSourceTargetForPath(path)
	if !ok || !target.implementation {
		return structuredTypeScriptImpactTarget{}, false
	}
	return target, true
}

func structuredTypeScriptDeclarationTargetForPath(path string) (structuredTypeScriptImpactTarget, bool) {
	target, ok := structuredTypeScriptSourceTargetForPath(path)
	if !ok || !target.declaration {
		return structuredTypeScriptImpactTarget{}, false
	}
	return target, true
}

func structuredTypeScriptImpactAllowsFileType(fileType string) bool {
	target, ok := structuredTypeScriptImpactTargetForFileType(fileType)
	return ok && target.structuredImpact
}

func structuredTypeScriptImpactAllowsFilePattern(pattern string) bool {
	target, ok := structuredTypeScriptImpactTargetForFilePattern(pattern)
	return ok && target.structuredImpact
}

func (target structuredTypeScriptImpactTarget) declarationImplementationPaths(path string) []string {
	if !target.declaration || len(target.pairedImplementationSuffixes) == 0 {
		return nil
	}

	key := structuredTypeScriptDefPathKey(path)
	if key == "" {
		return nil
	}
	lowerKey := strings.ToLower(key)
	if !strings.HasSuffix(lowerKey, target.suffix) {
		return nil
	}

	prefix := key[:len(key)-len(target.suffix)]
	paths := make([]string, 0, len(target.pairedImplementationSuffixes))
	for _, suffix := range target.pairedImplementationSuffixes {
		if suffix == "" {
			continue
		}
		paths = append(paths, prefix+suffix)
	}
	return paths
}

func (target structuredTypeScriptImpactTarget) nearbyTestCandidatePaths(defFile string) []string {
	if !target.structuredImpact || !target.implementation || len(target.nearbyTestCandidateDefinitions) == 0 {
		return nil
	}

	cleanFile := filepath.ToSlash(filepath.Clean(defFile))
	dir := filepath.ToSlash(filepath.Dir(cleanFile))
	base := typeScriptImpactBaseNameForTarget(cleanFile, target)

	candidates := make([]string, 0, len(target.nearbyTestCandidateDefinitions))
	for _, candidate := range target.nearbyTestCandidateDefinitions {
		switch candidate.location {
		case structuredTypeScriptNearbyTestSibling:
			candidates = append(candidates, filepath.ToSlash(filepath.Join(dir, base+candidate.suffix)))
		case structuredTypeScriptNearbyTestNestedTestsDir:
			candidates = append(candidates, filepath.ToSlash(filepath.Join(dir, "__tests__", base+candidate.suffix)))
		case structuredTypeScriptNearbyTestWorkspaceTestsDir:
			candidates = append(candidates, filepath.ToSlash(filepath.Join("tests", base+candidate.suffix)))
		}
	}
	return dedupeStringList(candidates)
}

func (target structuredTypeScriptImpactTarget) matchesNearbyTestPath(path string) bool {
	if !target.structuredImpact || !target.implementation {
		return false
	}
	lower := strings.ToLower(filepath.ToSlash(filepath.Clean(path)))
	for _, candidate := range target.nearbyTestCandidateDefinitions {
		if candidate.suffix != "" && strings.HasSuffix(lower, candidate.suffix) {
			return true
		}
	}
	return false
}

func typeScriptImpactBaseNameForTarget(path string, target structuredTypeScriptImpactTarget) string {
	base := filepath.Base(path)
	lowerBase := strings.ToLower(base)
	if strings.HasSuffix(lowerBase, target.suffix) {
		return base[:len(base)-len(target.suffix)]
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
}
