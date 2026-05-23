package search

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/filefilter"
)

func resolveSearchPathBasisForOptions(opts SearchOptions) filefilter.SearchPathBasis {
	return filefilter.ResolveSearchPathBasisWithWorkspace(opts.Path, resolveSearchWorkspaceRoot(opts))
}

func resolveSearchWorkspaceRoot(opts SearchOptions) string {
	return filefilter.ResolveWorkspaceRoot(opts.ProjectMapRootPath, opts.InvocationCWD)
}

func searchTargetPathForOptions(opts SearchOptions) string {
	basis := resolveSearchPathBasisForOptions(opts)
	target := strings.TrimSpace(basis.Target)
	if target == "" {
		target = "."
	}
	if filepath.IsAbs(target) {
		return filepath.Clean(target)
	}

	base := strings.TrimSpace(basis.Workdir)
	if base == "" {
		base = invocationCWDOrGetwd(opts)
	}
	if base == "" {
		return ""
	}
	return filepath.Clean(filepath.Join(base, target))
}
