package directquery

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/file/pathpolicy"
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
	Resolution directQueryResolution
	Error      string
}

type scopedDirectTargetOutcome struct {
	Kind   scopedDirectResolutionKind
	Target directQueryTarget
	Error  string
}

// resolveScopedGatherContextDirectResolution owns scope-aware soft direct
// resolution. It resolves only entries that can be interpreted exactly inside
// the supplied scope and returns no result instead of guessing outside that
// scope.
func resolveScopedGatherContextDirectResolution(execCtx tools.ExecutionContext, input directQueryInput, policy Policy) scopedDirectResolutionOutcome {
	if !inputHasOnlyScopedDirectCandidates(input) {
		return scopedDirectResolutionOutcome{Kind: scopedDirectResolutionNone}
	}

	ignoreMatcher := newScopedLookupIgnoreMatcher(execCtx)
	scopes := resolveScopedExactLookupScopes(execCtx, policy)
	if len(scopes) == 0 {
		return scopedDirectResolutionOutcome{Kind: scopedDirectResolutionNone}
	}

	targets := make([]directQueryTarget, 0, len(input.Entries))
	for _, entry := range input.Entries {
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

	if len(targets) == 1 && targets[0].Kind == directQueryTargetDirectory {
		return scopedDirectResolutionOutcome{
			Kind: scopedDirectResolutionResolved,
			Resolution: directQueryResolution{
				Kind:    directQueryResolutionDirectory,
				Targets: targets,
			},
		}
	}
	for _, target := range targets {
		if target.Kind != directQueryTargetFile {
			return scopedDirectResolutionOutcome{Kind: scopedDirectResolutionNone}
		}
	}
	return scopedDirectResolutionOutcome{
		Kind: scopedDirectResolutionResolved,
		Resolution: directQueryResolution{
			Kind:    directQueryResolutionFiles,
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
	if entry.ExplicitRelative {
		if len(scopes) == 0 {
			return scopedDirectTargetOutcome{
				Kind:  scopedDirectResolutionMissing,
				Error: "Error: direct path not found: " + entry.RawEntry,
			}
		}
		target, ok := resolveScopedRelativeDirectTargetInScope(scopes[0], entry, fileFilter)
		if !ok {
			return scopedDirectTargetOutcome{
				Kind:  scopedDirectResolutionMissing,
				Error: "Error: direct path not found: " + entry.RawEntry,
			}
		}
		return scopedDirectTargetOutcome{
			Kind:   scopedDirectResolutionResolved,
			Target: target,
		}
	}

	matches := make([]directQueryTarget, 0, 1)
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
			Error: "Error: direct path not found: " + entry.RawEntry,
		}
	}
	return scopedDirectTargetOutcome{
		Kind:   scopedDirectResolutionResolved,
		Target: matches[0],
	}
}

func resolveScopedRelativeDirectTargetInScope(scope scopedExactLookupScope, entry directQueryEntryInput, fileFilter string) (directQueryTarget, bool) {
	candidatePath := filepath.Join(scope.resolvedPath, entry.CleanedPath)
	resolvedPath, ok := resolveExistingScopedLookupPath(candidatePath, []string{scope.resolvedPath})
	if !ok {
		return directQueryTarget{}, false
	}
	return buildScopedTargetFromResolvedPath(scope, resolvedPath, entry, fileFilter)
}

func relativeScopedLookupDisplayPath(root, path string) (string, bool) {
	root = pathpolicy.NormalizeWorkspaceRoot(root)
	path = pathpolicy.NormalizeWorkspaceRoot(path)
	if root == "" || path == "" || !pathpolicy.IsPathWithinRoot(path, root) {
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

func buildScopedTargetFromResolvedPath(scope scopedExactLookupScope, resolvedPath string, entry directQueryEntryInput, fileFilter string) (directQueryTarget, bool) {
	displayPath, ok := relativeScopedLookupDisplayPath(scope.displayRoot, resolvedPath)
	if !ok {
		return directQueryTarget{}, false
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return directQueryTarget{}, false
	}
	if info.IsDir() {
		if entry.StartLine > 0 || entry.EndLine > 0 {
			return directQueryTarget{}, false
		}
		return directQueryTarget{
			RawEntry:      displayPath,
			FilePath:      displayPath,
			ResolvedPath:  resolvedPath,
			AllowedRoots:  []string{scope.resolvedPath},
			WorkspaceRoot: scope.displayRoot,
			FileFilter:    fileFilter,
			BypassIgnores: false,
			Kind:          directQueryTargetDirectory,
		}, true
	}
	return directQueryTarget{
		RawEntry:      normalizeDirectQueryRawEntry(displayPath, entry.StartLine, entry.EndLine),
		FilePath:      displayPath,
		ResolvedPath:  resolvedPath,
		AllowedRoots:  []string{scope.resolvedPath},
		WorkspaceRoot: scope.displayRoot,
		FileFilter:    fileFilter,
		BypassIgnores: false,
		StartLine:     entry.StartLine,
		EndLine:       entry.EndLine,
		Kind:          directQueryTargetFile,
	}, true
}

func entryCanUseScopedDirectResolution(entry directQueryEntryInput) bool {
	if entry.Syntax == directQuerySyntaxNone {
		return false
	}
	if strings.TrimSpace(entry.CleanedPath) == "" {
		return false
	}
	if filepath.IsAbs(entry.CleanedPath) || hasWindowsPathPrefix(entry.RawPath) {
		return false
	}
	return true
}

func usesScopedRelativeDirectPath(entry directQueryEntryInput) bool {
	return strings.ContainsAny(entry.RawPath, `/\`)
}
