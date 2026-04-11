package file

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	searchtool "github.com/susugadx/xelyon-cli/internal/tools/search"
)

type scopedExactLookupScope struct {
	displayRoot  string
	resolvedPath string
}

type scopedDirectResolutionKind string

const (
	scopedDirectResolutionNone      scopedDirectResolutionKind = "none"
	scopedDirectResolutionResolved  scopedDirectResolutionKind = "resolved"
	scopedDirectResolutionFiltered  scopedDirectResolutionKind = "filtered"
	scopedDirectResolutionMissing   scopedDirectResolutionKind = "missing"
	scopedDirectResolutionAmbiguous scopedDirectResolutionKind = "ambiguous"
)

type scopedDirectResolutionOutcome struct {
	Kind       scopedDirectResolutionKind
	Resolution DirectQueryResolution
	Error      string
}

type scopedDirectTargetOutcome struct {
	Kind   scopedDirectResolutionKind
	Target DirectQueryTarget
	Error  string
}

// resolveScopedGatherContextDirectResolution owns scope-aware soft direct
// resolution. It resolves only entries that can be interpreted exactly inside
// the supplied scope and returns no result instead of guessing outside that
// scope.
func resolveScopedGatherContextDirectResolution(execCtx tools.ExecutionContext, input directQueryInput, policy GatherContextDirectRoutePolicy) scopedDirectResolutionOutcome {
	if !inputHasOnlyScopedDirectCandidates(input) {
		return scopedDirectResolutionOutcome{Kind: scopedDirectResolutionNone}
	}

	ignoreMatcher := newScopedLookupIgnoreMatcher(execCtx)
	scopes := resolveScopedExactLookupScopes(execCtx, policy)
	if len(scopes) == 0 {
		return scopedDirectResolutionOutcome{Kind: scopedDirectResolutionNone}
	}

	targets := make([]DirectQueryTarget, 0, len(input.entries))
	for _, entry := range input.entries {
		targetOutcome := resolveScopedGatherContextTarget(scopes, ignoreMatcher, entry, policy.FileFilter)
		switch targetOutcome.Kind {
		case scopedDirectResolutionResolved:
			targets = append(targets, targetOutcome.Target)
		case scopedDirectResolutionFiltered:
			return scopedDirectResolutionOutcome{Kind: scopedDirectResolutionFiltered}
		case scopedDirectResolutionMissing:
			return scopedDirectResolutionOutcome{
				Kind:  scopedDirectResolutionMissing,
				Error: targetOutcome.Error,
			}
		case scopedDirectResolutionAmbiguous:
			return scopedDirectResolutionOutcome{Kind: scopedDirectResolutionAmbiguous}
		default:
			return scopedDirectResolutionOutcome{Kind: scopedDirectResolutionNone}
		}
	}

	if len(targets) == 1 && targets[0].Kind == DirectQueryTargetDirectory {
		return scopedDirectResolutionOutcome{
			Kind: scopedDirectResolutionResolved,
			Resolution: DirectQueryResolution{
				Kind:    DirectQueryResolutionDirectory,
				Targets: targets,
			},
		}
	}
	for _, target := range targets {
		if target.Kind != DirectQueryTargetFile {
			return scopedDirectResolutionOutcome{Kind: scopedDirectResolutionNone}
		}
	}
	return scopedDirectResolutionOutcome{
		Kind: scopedDirectResolutionResolved,
		Resolution: DirectQueryResolution{
			Kind:    DirectQueryResolutionFiles,
			Targets: targets,
		},
	}
}

func resolveScopedGatherContextTarget(scopes []scopedExactLookupScope, ignoreMatcher *pathmatch.Matcher, entry directQueryEntryInput, fileFilter string) scopedDirectTargetOutcome {
	if usesScopedRelativeDirectPath(entry) {
		return resolveScopedRelativeDirectTarget(scopes, entry, fileFilter)
	}
	return resolveScopedBasenameDirectTarget(scopes, ignoreMatcher, entry, fileFilter)
}

func resolveScopedExactLookupScopes(execCtx tools.ExecutionContext, policy GatherContextDirectRoutePolicy) []scopedExactLookupScope {
	roots := directQueryRoots(execCtx)
	if len(roots) == 0 {
		return nil
	}

	scopePath := strings.TrimSpace(policy.ScopedPath)
	scopes := make([]scopedExactLookupScope, 0, len(roots))
	seen := make(map[string]struct{})
	addScope := func(resolvedPath, displayRoot string) {
		resolvedPath = normalizeWorkspaceRoot(resolvedPath)
		displayRoot = normalizeWorkspaceRoot(displayRoot)
		if resolvedPath == "" || displayRoot == "" {
			return
		}
		if _, ok := seen[resolvedPath]; ok {
			return
		}
		seen[resolvedPath] = struct{}{}
		scopes = append(scopes, scopedExactLookupScope{
			displayRoot:  displayRoot,
			resolvedPath: resolvedPath,
		})
	}

	if scopePath == "" {
		for _, root := range roots {
			addScope(root, root)
		}
		return scopes
	}

	if filepath.IsAbs(scopePath) || hasWindowsPathPrefix(scopePath) {
		resolvedPath, ok := resolveExistingScopedLookupPath(scopePath, roots)
		if !ok {
			return nil
		}
		addScope(resolvedPath, preferredScopedLookupDisplayRoot(resolvedPath, roots))
		return scopes
	}

	baseRoot := preferredScopedLookupBaseRoot(execCtx)
	if baseRoot == "" {
		return nil
	}
	resolvedPath, ok := resolveExistingScopedLookupPath(filepath.Join(baseRoot, scopePath), []string{baseRoot})
	if !ok {
		return nil
	}
	addScope(resolvedPath, baseRoot)
	return scopes
}

func preferredScopedLookupBaseRoot(execCtx tools.ExecutionContext) string {
	if root := normalizeWorkspaceRoot(execCtx.InvocationCWD); root != "" {
		return root
	}
	return normalizeWorkspaceRoot(execCtx.ProjectMapRootPath)
}

func newScopedLookupIgnoreMatcher(execCtx tools.ExecutionContext) *pathmatch.Matcher {
	patterns := config.ResolveSharedIgnorePatterns(execCtx.EffectiveConfig(), config.LoadProjectConfig())
	if len(patterns) == 0 {
		return nil
	}
	return pathmatch.NewMatcher(patterns)
}

func resolveExistingScopedLookupPath(path string, allowedRoots []string) (string, bool) {
	out := common.NewOutput(io.Discard, io.Discard)
	resolvedPath, errResult := resolveValidatedPathWithRoots(out, path, allowedRoots, "path is empty")
	if errResult != "" {
		return "", false
	}
	if _, err := os.Stat(resolvedPath); err != nil {
		return "", false
	}
	return resolvedPath, true
}

func preferredScopedLookupDisplayRoot(path string, roots []string) string {
	path = normalizeWorkspaceRoot(path)
	for _, root := range roots {
		root = normalizeWorkspaceRoot(root)
		if root == "" {
			continue
		}
		if isPathWithinRoot(path, root) {
			return root
		}
	}
	return ""
}

func resolveScopedRelativeDirectTarget(scopes []scopedExactLookupScope, entry directQueryEntryInput, fileFilter string) scopedDirectTargetOutcome {
	if entry.explicitRelative {
		if len(scopes) == 0 {
			return scopedDirectTargetOutcome{
				Kind:  scopedDirectResolutionMissing,
				Error: "Error: direct path not found: " + entry.rawEntry,
			}
		}
		target, ok := resolveScopedRelativeDirectTargetInScope(scopes[0], entry, fileFilter)
		if !ok {
			return scopedDirectTargetOutcome{
				Kind:  scopedDirectResolutionMissing,
				Error: "Error: direct path not found: " + entry.rawEntry,
			}
		}
		return scopedDirectTargetOutcome{
			Kind:   scopedDirectResolutionResolved,
			Target: target,
		}
	}

	matches := make([]DirectQueryTarget, 0, 1)
	seen := make(map[string]struct{})
	for _, scope := range scopes {
		target, ok := resolveScopedRelativeDirectTargetInScope(scope, entry, fileFilter)
		if !ok {
			continue
		}
		if _, exists := seen[target.ResolvedPath]; exists {
			continue
		}
		seen[target.ResolvedPath] = struct{}{}
		matches = append(matches, target)
		if len(matches) > 1 {
			return scopedDirectTargetOutcome{Kind: scopedDirectResolutionAmbiguous}
		}
	}
	if len(matches) != 1 {
		return scopedDirectTargetOutcome{
			Kind:  scopedDirectResolutionMissing,
			Error: "Error: direct path not found: " + entry.rawEntry,
		}
	}
	return scopedDirectTargetOutcome{
		Kind:   scopedDirectResolutionResolved,
		Target: matches[0],
	}
}

func resolveScopedRelativeDirectTargetInScope(scope scopedExactLookupScope, entry directQueryEntryInput, fileFilter string) (DirectQueryTarget, bool) {
	candidatePath := filepath.Join(scope.resolvedPath, entry.cleanedPath)
	resolvedPath, ok := resolveExistingScopedLookupPath(candidatePath, []string{scope.resolvedPath})
	if !ok {
		return DirectQueryTarget{}, false
	}
	return buildScopedTargetFromResolvedPath(scope, resolvedPath, entry, fileFilter)
}

func resolveScopedBasenameDirectTarget(scopes []scopedExactLookupScope, ignoreMatcher *pathmatch.Matcher, entry directQueryEntryInput, fileFilter string) scopedDirectTargetOutcome {
	trimmedFilter := strings.TrimSpace(fileFilter)
	matches := make([]DirectQueryTarget, 0, 1)
	seen := make(map[string]struct{})
	for _, scope := range scopes {
		limit := 0
		if trimmedFilter == "" {
			limit = 2 - len(matches)
		}
		for _, target := range collectScopedBasenameTargets(scope, ignoreMatcher, entry, fileFilter, seen, limit) {
			matches = append(matches, target)
			if trimmedFilter == "" && len(matches) > 1 {
				return scopedDirectTargetOutcome{Kind: scopedDirectResolutionAmbiguous}
			}
		}
	}
	return selectScopedBasenameDirectTarget(matches, trimmedFilter, entry.rawEntry)
}

func selectScopedBasenameDirectTarget(matches []DirectQueryTarget, fileFilter string, rawEntry string) scopedDirectTargetOutcome {
	if len(matches) == 0 {
		return scopedDirectTargetOutcome{
			Kind:  scopedDirectResolutionMissing,
			Error: "Error: direct path not found: " + rawEntry,
		}
	}

	if fileFilter != "" {
		// Bare/scoped basename resolution is a soft direct route. file_filter
		// must be honored for the final exact target, but a mismatch should not
		// masquerade as a direct read.
		filtered := filterScopedBasenameTargets(matches, fileFilter)
		if len(filtered) == 1 {
			return scopedDirectTargetOutcome{
				Kind:   scopedDirectResolutionResolved,
				Target: filtered[0],
			}
		}
		if len(filtered) == 0 {
			return scopedDirectTargetOutcome{Kind: scopedDirectResolutionFiltered}
		}
		return scopedDirectTargetOutcome{Kind: scopedDirectResolutionAmbiguous}
	}

	if len(matches) == 1 {
		return scopedDirectTargetOutcome{
			Kind:   scopedDirectResolutionResolved,
			Target: matches[0],
		}
	}
	return scopedDirectTargetOutcome{Kind: scopedDirectResolutionAmbiguous}
}

func filterScopedBasenameTargets(matches []DirectQueryTarget, fileFilter string) []DirectQueryTarget {
	filtered := make([]DirectQueryTarget, 0, len(matches))
	for _, target := range matches {
		if target.Kind != DirectQueryTargetFile {
			filtered = append(filtered, target)
			continue
		}
		if searchtool.MatchesRawFileFilter(target.FilePath, fileFilter) {
			filtered = append(filtered, target)
		}
	}
	return filtered
}

func collectScopedBasenameTargets(scope scopedExactLookupScope, ignoreMatcher *pathmatch.Matcher, entry directQueryEntryInput, fileFilter string, seen map[string]struct{}, limit int) []DirectQueryTarget {
	if limit <= 0 {
		limit = 0
	}

	info, err := os.Stat(scope.resolvedPath)
	if err != nil {
		return nil
	}

	if !info.IsDir() {
		target, ok := buildScopedBasenameTarget(scope, scope.resolvedPath, entry, fileFilter, seen)
		if !ok {
			return nil
		}
		return []DirectQueryTarget{target}
	}

	matches := make([]DirectQueryTarget, 0, 1)
	_ = filepath.WalkDir(scope.resolvedPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path != scope.resolvedPath {
			relPath, ok := relativeScopedLookupDisplayPath(scope.displayRoot, path)
			if ok && ignoreMatcher != nil && ignoreMatcher.Match(relPath, d.IsDir()) {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}
		if d.IsDir() {
			return nil
		}

		target, ok := buildScopedBasenameTarget(scope, path, entry, fileFilter, seen)
		if !ok {
			return nil
		}

		matches = append(matches, target)
		if limit > 0 && len(matches) >= limit {
			return fs.SkipAll
		}
		return nil
	})
	return matches
}

func buildScopedBasenameTarget(scope scopedExactLookupScope, candidatePath string, entry directQueryEntryInput, fileFilter string, seen map[string]struct{}) (DirectQueryTarget, bool) {
	if filepath.Base(candidatePath) != entry.cleanedPath {
		return DirectQueryTarget{}, false
	}

	resolvedPath, ok := resolveExistingScopedLookupPath(candidatePath, []string{scope.resolvedPath})
	if !ok {
		return DirectQueryTarget{}, false
	}
	if _, exists := seen[resolvedPath]; exists {
		return DirectQueryTarget{}, false
	}

	target, ok := buildScopedTargetFromResolvedPath(scope, resolvedPath, entry, fileFilter)
	if !ok {
		return DirectQueryTarget{}, false
	}
	seen[resolvedPath] = struct{}{}
	return target, true
}

func relativeScopedLookupDisplayPath(root, path string) (string, bool) {
	root = normalizeWorkspaceRoot(root)
	path = normalizeWorkspaceRoot(path)
	if root == "" || path == "" || !isPathWithinRoot(path, root) {
		return "", false
	}

	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return "", false
	}
	relPath = filepath.Clean(relPath)
	if relPath == "." || relPath == "" || strings.HasPrefix(relPath, "..") {
		return "", false
	}
	return relPath, true
}

func buildScopedTargetFromResolvedPath(scope scopedExactLookupScope, resolvedPath string, entry directQueryEntryInput, fileFilter string) (DirectQueryTarget, bool) {
	displayPath, ok := relativeScopedLookupDisplayPath(scope.displayRoot, resolvedPath)
	if !ok {
		return DirectQueryTarget{}, false
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return DirectQueryTarget{}, false
	}
	if info.IsDir() {
		if entry.startLine > 0 || entry.endLine > 0 {
			return DirectQueryTarget{}, false
		}
		return DirectQueryTarget{
			RawEntry:      displayPath,
			FilePath:      displayPath,
			ResolvedPath:  resolvedPath,
			AllowedRoots:  []string{scope.resolvedPath},
			WorkspaceRoot: scope.displayRoot,
			FileFilter:    fileFilter,
			BypassIgnores: false,
			Kind:          DirectQueryTargetDirectory,
		}, true
	}
	return DirectQueryTarget{
		RawEntry:      normalizeDirectQueryRawEntry(displayPath, entry.startLine, entry.endLine),
		FilePath:      displayPath,
		ResolvedPath:  resolvedPath,
		AllowedRoots:  []string{scope.resolvedPath},
		WorkspaceRoot: scope.displayRoot,
		FileFilter:    fileFilter,
		BypassIgnores: false,
		StartLine:     entry.startLine,
		EndLine:       entry.endLine,
		Kind:          DirectQueryTargetFile,
	}, true
}

func entryCanUseScopedDirectResolution(entry directQueryEntryInput) bool {
	if entry.syntax == directQuerySyntaxNone {
		return false
	}
	if strings.TrimSpace(entry.cleanedPath) == "" {
		return false
	}
	if filepath.IsAbs(entry.cleanedPath) || hasWindowsPathPrefix(entry.rawPath) {
		return false
	}
	return true
}

func usesScopedRelativeDirectPath(entry directQueryEntryInput) bool {
	return strings.ContainsAny(entry.rawPath, `/\`)
}
