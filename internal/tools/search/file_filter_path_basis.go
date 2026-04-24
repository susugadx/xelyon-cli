package search

import "github.com/susugadx/xelyon-cli/internal/filefilter"

func resolveSearchPathBasisForOptions(opts SearchOptions) filefilter.SearchPathBasis {
	return filefilter.ResolveSearchPathBasisWithWorkspace(opts.Path, resolveSearchWorkspaceRoot(opts))
}

func resolveSearchWorkspaceRoot(opts SearchOptions) string {
	return filefilter.ResolveWorkspaceRoot(opts.ProjectMapRootPath, opts.InvocationCWD)
}
