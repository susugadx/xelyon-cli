package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/taskstate"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/applypatch"
)

type editReadinessTargetExtraction struct {
	targets []taskstate.EditReadinessTarget
	unknown bool
}

func extractEditReadinessTargets(toolCall *tools.ToolCall) editReadinessTargetExtraction {
	if toolCall == nil {
		return editReadinessTargetExtraction{}
	}
	switch toolCall.Tool {
	case "str_replace", "write_file", "delete_file":
		return extractSinglePathEditReadinessTarget(toolCall)
	case "apply_patch":
		return extractApplyPatchEditReadinessTargets(toolCall)
	default:
		return editReadinessTargetExtraction{}
	}
}

func extractSinglePathEditReadinessTarget(toolCall *tools.ToolCall) editReadinessTargetExtraction {
	path := strings.TrimSpace(toolCall.Args["path"])
	if path == "" {
		return editReadinessTargetExtraction{unknown: true}
	}
	return editReadinessTargetExtraction{targets: []taskstate.EditReadinessTarget{newEditReadinessTarget(toolCall, path)}}
}

func extractApplyPatchEditReadinessTargets(toolCall *tools.ToolCall) editReadinessTargetExtraction {
	patch := strings.TrimSpace(toolCall.Args["patch"])
	if patch == "" {
		return editReadinessTargetExtraction{unknown: true}
	}
	parsed, err := applypatch.ParsePatch(patch)
	if err != nil || parsed == nil || len(parsed.Hunks) == 0 {
		return editReadinessTargetExtraction{unknown: true}
	}
	targets := make([]taskstate.EditReadinessTarget, 0, len(parsed.Hunks))
	seen := make(map[string]struct{}, len(parsed.Hunks)*2)
	addTarget := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		targets = append(targets, newEditReadinessTarget(toolCall, path))
	}
	for _, hunk := range parsed.Hunks {
		addTarget(hunk.Path)
		addTarget(hunk.MovePath)
	}
	if len(targets) == 0 {
		return editReadinessTargetExtraction{unknown: true}
	}
	return editReadinessTargetExtraction{targets: targets}
}

func newEditReadinessTarget(toolCall *tools.ToolCall, path string) taskstate.EditReadinessTarget {
	return taskstate.EditReadinessTarget{
		Path:       path,
		ToolName:   toolCall.Tool,
		ToolCallID: toolCall.ID,
	}
}

func editReadinessToolName(toolCall *tools.ToolCall) string {
	if toolCall == nil {
		return ""
	}
	return toolCall.Tool
}

func editReadinessToolCallID(toolCall *tools.ToolCall) string {
	if toolCall == nil {
		return ""
	}
	return toolCall.ID
}
