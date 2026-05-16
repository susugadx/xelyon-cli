package search

import (
	"path"
	"path/filepath"
	"strings"
)

func structuredGoImpactPathHintForFilePattern(opts SearchOptions, pattern string) (string, bool) {
	dir := structuredGoImpactStaticDirectoryForFilePattern(pattern)
	if dir == "" {
		return "", true
	}
	if filepath.IsAbs(dir) {
		return filepath.Clean(dir), true
	}

	return newStructuredGoImpactFilePatternScope(opts, dir).definitionPathHint()
}

func structuredGoImpactRawEvidenceDisplayPathAllowed(opts SearchOptions) bool {
	dir := structuredGoImpactStaticDirectoryForFilePattern(opts.FilePattern)
	if dir == "" {
		return true
	}
	if filepath.IsAbs(dir) {
		return false
	}
	return newStructuredGoImpactFilePatternScope(opts, dir).rawEvidenceDisplayPathAllowed()
}

func structuredGoImpactStaticDirectoryForFilePattern(pattern string) string {
	return staticDirectoryPrefixForGlob(cleanStructuredGoFilePattern(pattern))
}

type structuredGoImpactFilePatternScope struct {
	dir             string
	workspaceRoot   string
	searchTargetRel string
	matchRoot       string
	searchTargetAbs string
}

func newStructuredGoImpactFilePatternScope(opts SearchOptions, dir string) structuredGoImpactFilePatternScope {
	basis := resolveSearchPathBasisForOptions(opts)
	return structuredGoImpactFilePatternScope{
		dir:             cleanStaticDirectoryPrefix(dir),
		workspaceRoot:   structuredImpactWorkspaceRoot(opts),
		searchTargetRel: cleanStaticDirectoryPrefix(basis.Target),
		matchRoot:       strings.TrimSpace(basis.MatchRoot),
		searchTargetAbs: structuredImpactSearchTargetPath(opts),
	}
}

func (scope structuredGoImpactFilePatternScope) definitionPathHint() (string, bool) {
	if scope.dir == "" {
		return "", true
	}
	if scope.searchTargetRel != "" {
		if scope.globPrefixIncludesSearchTarget() && scope.hasWorkspaceRoot() {
			return scope.workspacePath(scope.dir), true
		}
		return "", false
	}
	if scope.hasWorkspaceRoot() && scope.matchRoot != "" {
		return scope.workspacePath(scope.dir), true
	}
	if scope.searchTargetAbs != "" {
		return filepath.Clean(filepath.Join(scope.searchTargetAbs, filepath.FromSlash(scope.dir))), true
	}
	if scope.hasWorkspaceRoot() {
		return scope.workspacePath(scope.dir), true
	}
	return "", false
}

func (scope structuredGoImpactFilePatternScope) rawEvidenceDisplayPathAllowed() bool {
	return scope.searchTargetRel == "" && scope.matchRoot == ""
}

func (scope structuredGoImpactFilePatternScope) globPrefixIncludesSearchTarget() bool {
	if scope.searchTargetRel == "" {
		return false
	}
	return scope.dir == scope.searchTargetRel || strings.HasPrefix(scope.dir, scope.searchTargetRel+"/")
}

func (scope structuredGoImpactFilePatternScope) hasWorkspaceRoot() bool {
	return scope.workspaceRoot != "" && scope.workspaceRoot != "."
}

func (scope structuredGoImpactFilePatternScope) workspacePath(dir string) string {
	return filepath.Clean(filepath.Join(scope.workspaceRoot, filepath.FromSlash(dir)))
}

func staticDirectoryPrefixForGlob(pattern string) string {
	pattern = filepath.ToSlash(strings.TrimSpace(pattern))
	if pattern == "" {
		return ""
	}
	globIndex := strings.IndexAny(pattern, "*?[")
	if globIndex < 0 {
		return cleanStaticDirectoryPrefix(path.Dir(pattern))
	}
	prefix := pattern[:globIndex]
	if strings.HasSuffix(prefix, "/") {
		return cleanStaticDirectoryPrefix(strings.TrimRight(prefix, "/"))
	}
	return cleanStaticDirectoryPrefix(path.Dir(prefix))
}

func cleanStaticDirectoryPrefix(prefix string) string {
	prefix = strings.TrimSpace(filepath.ToSlash(prefix))
	if prefix == "" || prefix == "." {
		return ""
	}
	return path.Clean(prefix)
}
