package file

import "github.com/susugadx/xelyon-cli/internal/tools"

// ExecuteDirectReadTargetsWithDetail runs direct file/range reads with the same semantics as read_file.
func ExecuteDirectReadTargetsWithDetail(execCtx tools.ExecutionContext, targets []DirectQueryTarget, detail string) string {
	return renderReadExecutionSections(ExecuteDirectReadTargetsWithDetailSections(execCtx, targets, detail))
}

// ExecuteDirectReadTargetsWithDetailSections runs direct file/range reads and returns structured sections.
func ExecuteDirectReadTargetsWithDetailSections(execCtx tools.ExecutionContext, targets []DirectQueryTarget, detail string) []ReadExecutionSection {
	mode, errResult := resolveReadDetail(detail, "")
	if errResult != "" {
		return []ReadExecutionSection{{Output: errResult, Failed: true}}
	}

	requests := buildReadRequestsFromDirectQueryTargets(targets, mode)
	return executeReadFilesRequestsSections(
		execCtx.Output(),
		execCtx.EffectiveConfig(),
		execCtx.EffectiveToolCache(),
		requests,
		0,
		execCtx.EffectiveLocatorRegistry(),
	)
}

// ExecuteDirectListDirTarget runs list_dir using a resolved direct directory target.
func ExecuteDirectListDirTarget(execCtx tools.ExecutionContext, target DirectQueryTarget, depth int) string {
	if target.Kind != DirectQueryTargetDirectory {
		return "Error: path is not a directory"
	}
	ignoreMode := listDirApplyIgnores
	if target.BypassIgnores {
		ignoreMode = listDirBypassIgnores
	}
	return executeListDirWithRuntimeMode(execCtx.EffectiveConfig(), execCtx.EffectiveToolCache(), target.ResolvedPath, depth, target.AllowedRoots, target.FileFilter, target.WorkspaceRoot, execCtx.ProjectMap, execCtx.ProjectMapStateKey, ignoreMode)
}

func buildReadRequestsFromDirectQueryTargets(targets []DirectQueryTarget, detail readDetailMode) []readRequest {
	requests := make([]readRequest, 0, len(targets))
	for _, target := range targets {
		if target.Kind != DirectQueryTargetFile {
			continue
		}

		source := readRequestSourcePathWhole
		if target.StartLine > 0 || target.EndLine > 0 {
			source = readRequestSourcePathRange
		}
		requests = append(requests, readRequest{
			RawEntry:     normalizeDirectQueryRawEntry(target.FilePath, target.StartLine, target.EndLine),
			FilePath:     target.FilePath,
			ResolvedPath: target.ResolvedPath,
			AllowedRoots: target.AllowedRoots,
			StartLine:    target.StartLine,
			EndLine:      target.EndLine,
			Source:       source,
			Detail:       detail,
		})
	}
	return requests
}
