package file

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

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
