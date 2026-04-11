package file

import "github.com/susugadx/xelyon-cli/internal/tools"

// ParsePathEntry は path[:line[-end]] 形式を解析する。
func ParsePathEntry(entry string) (string, int, int) {
	return parsePath(entry)
}

// ExecuteReadPathsWithDetail は read_file の path/read semantics を helper として公開する。
func ExecuteReadPathsWithDetail(execCtx tools.ExecutionContext, paths []string, detail string) string {
	return renderReadExecutionSections(ExecuteReadPathsWithDetailSections(execCtx, paths, detail))
}

// ExecuteReadTargetsWithDetail は locator targets を read_file と同じ runtime で読む。
func ExecuteReadTargetsWithDetail(execCtx tools.ExecutionContext, targets string, detail string) string {
	return renderReadExecutionSections(ExecuteReadTargetsWithDetailSections(execCtx, targets, detail))
}

// ExecuteListDirWithContext は list_dir runtime を helper として公開する。
func ExecuteListDirWithContext(execCtx tools.ExecutionContext, path string, depth int) string {
	return executeListDirWithRuntime(execCtx.EffectiveConfig(), execCtx.EffectiveToolCache(), path, depth, nil, "", "", execCtx.ProjectMap, execCtx.ProjectMapStateKey)
}
