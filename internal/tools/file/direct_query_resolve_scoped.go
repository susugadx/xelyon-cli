package file

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/tools"
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
