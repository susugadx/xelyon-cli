package directquery

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/file/listtool"
	"github.com/susugadx/xelyon-cli/internal/tools/file/readtool"
)

// executeDirectReadTargetsWithDetail runs direct file/range reads with the same semantics as read_file.
func executeDirectReadTargetsWithDetail(execCtx tools.ExecutionContext, targets []directQueryTarget, detail string) string {
	return readtool.RenderReadExecutionSections(executeDirectReadTargetsWithDetailSections(execCtx, targets, detail))
}

// executeDirectReadTargetsWithDetailSections runs direct file/range reads and returns structured sections.
func executeDirectReadTargetsWithDetailSections(execCtx tools.ExecutionContext, targets []directQueryTarget, detail string) []readtool.ReadExecutionSection {
	return readtool.ExecuteResolvedRequestsWithDetailSections(execCtx, buildResolvedReadRequestsFromDirectQueryTargets(targets), detail)
}

// executeDirectListDirTarget runs list_dir using a resolved direct directory target.
func executeDirectListDirTarget(execCtx tools.ExecutionContext, target directQueryTarget, depth int) string {
	if target.Kind != directQueryTargetDirectory {
		return "Error: path is not a directory"
	}
	return listtool.ExecuteResolvedTarget(execCtx, listtool.ResolvedTarget{
		Path:          target.ResolvedPath,
		AllowedRoots:  target.AllowedRoots,
		FileFilter:    target.FileFilter,
		WorkspaceRoot: target.WorkspaceRoot,
		BypassIgnores: target.BypassIgnores,
	}, depth)
}

func buildResolvedReadRequestsFromDirectQueryTargets(targets []directQueryTarget) []readtool.ResolvedRequest {
	requests := make([]readtool.ResolvedRequest, 0, len(targets))
	for _, target := range targets {
		if target.Kind != directQueryTargetFile {
			continue
		}

		requests = append(requests, readtool.ResolvedRequest{
			RawEntry:     readtool.FormatPathEntry(target.FilePath, target.StartLine, target.EndLine),
			FilePath:     target.FilePath,
			ResolvedPath: target.ResolvedPath,
			AllowedRoots: target.AllowedRoots,
			StartLine:    target.StartLine,
			EndLine:      target.EndLine,
		})
	}
	return requests
}
