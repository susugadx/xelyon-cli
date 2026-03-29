package applypatch

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func init() {
	tools.DefaultRegistry.Register(&ApplyPatchTool{})
}

// ApplyPatchTool は apply_patch ツールを登録する。
type ApplyPatchTool struct{}

func (t *ApplyPatchTool) Name() string { return "apply_patch" }

func (t *ApplyPatchTool) Description() string { return applyPatchDescription }

func (t *ApplyPatchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"patch": map[string]interface{}{
				"type":        "string",
				"description": "The patch text. Must start with '*** Begin Patch' and end with '*** End Patch'.",
			},
		},
		"required":             []string{"patch"},
		"additionalProperties": false,
	}
}

func (t *ApplyPatchTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	patchText := args["patch"]
	if patchText == "" {
		return "Error: patch is required", nil, nil
	}

	parsed, err := ParsePatch(patchText)
	if err != nil {
		return "", nil, err
	}

	autoApproved := shouldAutoApproveApplyPatch(execCtx, parsed)
	if !autoApproved {
		showApplyPatchPreview(execCtx.Output(), patchText, parsed.Hunks)
	}

	decision := confirmApplyPatch(execCtx, parsed, autoApproved)

	switch decision.Action {
	case common.ConfirmYes:
	case common.ConfirmComment:
		return fmt.Sprintf(`[COMMENT] User provided feedback for apply_patch.

Comment:
%s

Next actions:
- Read the affected files or patch context again.
- Revise the patch and ask for approval before applying it.

IMPORTANT: Do NOT apply the patch until the user approves.`, strings.TrimSpace(decision.Comment)), nil, nil
	default:
		return "[CANCELLED] apply_patch was not approved", nil, nil
	}

	result, err := applyParsedPatch(parsed)
	if err != nil {
		return "", nil, err
	}

	if autoApproved {
		showCodexStyleResult(execCtx.Output(), result)
	}

	return formatApplyResult(result), buildApplyPatchFileChange(result), nil
}

func confirmApplyPatch(execCtx tools.ExecutionContext, parsed *ParsedPatch, autoApproved bool) common.ConfirmDecision {
	const message = "Apply this patch? / このパッチを適用しますか？"

	if autoApproved {
		safety := getApplyPatchSafety(parsed)
		execCtx.Output().Green.Printf("Auto-approved (%s): %s\n", common.GetSafetyDescription(safety), "apply_patch")
		return common.ConfirmDecision{Action: common.ConfirmYes}
	}

	return common.ConfirmWithIO(execCtx.PromptIO(), message)
}

func shouldAutoApproveApplyPatch(execCtx tools.ExecutionContext, parsed *ParsedPatch) bool {
	options := execCtx.ConfirmOptions()
	if common.IsAutoApprovable("apply_patch", options.AutoApprove) {
		return true
	}

	safety := getApplyPatchSafety(parsed)
	if cfg := options.Config; cfg != nil && cfg.ToolConfirm.AutoApproveMedium && safety == common.SafetyMedium {
		return true
	}

	return false
}

func getApplyPatchSafety(parsed *ParsedPatch) common.ToolSafety {
	if parsed == nil {
		return common.SafetyLow
	}

	for _, hunk := range parsed.Hunks {
		if hunk.Type == "delete" {
			return common.SafetyLow
		}
		if hunk.Type == "update" && hunk.MovePath != "" {
			srcPath, srcErr := common.ValidatePath(hunk.Path)
			dstPath, dstErr := common.ValidatePath(hunk.MovePath)
			if srcErr != nil || dstErr != nil || srcPath != dstPath {
				return common.SafetyLow
			}
		}
	}

	return common.SafetyMedium
}

func showCodexStyleResult(out common.Output, result *ApplyResult) {
	if out.SuppressStdout() {
		return
	}

	showApplyPatchHeader(out, len(result.details))

	var files []ui.PatchFileDisplay
	for _, d := range result.details {
		files = append(files, ui.PatchFileDisplay{
			Path:       d.Path,
			Action:     d.Action,
			OldContent: d.OldContent,
			NewContent: d.NewContent,
		})
	}

	ui.ShowCodexStyleDiff(out.StdoutWriter(), files)
}

func showApplyPatchPreview(out common.Output, patchText string, hunks []Hunk) {
	if out.SuppressStdout() {
		return
	}

	showApplyPatchHeader(out, len(hunks))
	showApplyPatchPreviewBody(out, patchText, hunks)
}

func showApplyPatchHeader(out common.Output, operations int) {
	ui.FileOpHeader(out.StdoutWriter(), "apply_patch", fmt.Sprintf("%d files", operations))
}

func showApplyPatchPreviewBody(out common.Output, patchText string, hunks []Hunk) {
	ui.ShowPatchToWriter(out.StdoutWriter(), patchText)

	counts := countLinesPerPath(patchText)

	w := out.StdoutWriter()
	for _, hunk := range hunks {
		c := counts[hunk.Path]
		linesInfo := ""
		if c[0] > 0 || c[1] > 0 {
			linesInfo = fmt.Sprintf("(+%d, -%d)", c[0], c[1])
		}
		switch hunk.Type {
		case "add":
			ui.FileOpPathLine(w, "A", hunk.Path, linesInfo)
		case "delete":
			ui.FileOpPathLine(w, "D", hunk.Path, "")
		case "update":
			target := hunk.Path
			if hunk.MovePath != "" {
				target = hunk.Path + " -> " + hunk.MovePath
			}
			ui.FileOpPathLine(w, "M", target, linesInfo)
		}
	}
	out.Println()
}

func countLinesPerPath(patchText string) map[string][2]int {
	counts := make(map[string][2]int) // path -> [added, removed]
	lines := strings.Split(patchText, "\n")
	var currentPath string

	for _, line := range lines {
		if strings.HasPrefix(line, "*** Add File: ") {
			currentPath = strings.TrimSpace(strings.TrimPrefix(line, "*** Add File: "))
		} else if strings.HasPrefix(line, "*** Update File: ") {
			currentPath = strings.TrimSpace(strings.TrimPrefix(line, "*** Update File: "))
		} else if strings.HasPrefix(line, "*** Delete File: ") {
			currentPath = strings.TrimSpace(strings.TrimPrefix(line, "*** Delete File: "))
		} else if strings.HasPrefix(line, "*** Move to: ") {
			// Move doesn't change currentPath reference for + / - counting
		} else if strings.HasPrefix(line, "+") && currentPath != "" {
			c := counts[currentPath]
			c[0]++
			counts[currentPath] = c
		} else if strings.HasPrefix(line, "-") && currentPath != "" {
			c := counts[currentPath]
			c[1]++
			counts[currentPath] = c
		}
	}
	return counts
}

func formatApplyResult(result *ApplyResult) string {
	return fmt.Sprintf(
		"✓ Patch applied successfully.\nAdded: %s\nModified: %s\nDeleted: %s",
		formatApplyPaths(result.Added),
		formatApplyPaths(result.Modified),
		formatApplyPaths(result.Deleted),
	)
}

func formatApplyPaths(paths []string) string {
	if len(paths) == 0 {
		return "(none)"
	}
	return strings.Join(paths, ", ")
}

func buildApplyPatchFileChange(result *ApplyResult) *tools.FileChange {
	if result == nil {
		return nil
	}

	if len(result.details) == 0 {
		return nil
	}

	change := &tools.FileChange{
		Timestamp: common.GetCurrentTime(),
		Tool:      "apply_patch",
		Details:   make([]tools.FileChangeDetail, 0, len(result.details)),
	}

	for _, detail := range result.details {
		change.Details = append(change.Details, tools.FileChangeDetail{
			FilePath:     detail.Path,
			Action:       detail.Action,
			LinesAdded:   detail.LinesAdded,
			LinesRemoved: detail.LinesRemoved,
		})
	}
	change.FilePath = result.details[0].Path

	if len(result.details) == 1 {
		change.Description = "Patched file " + result.details[0].Path
	} else {
		change.Description = fmt.Sprintf("Patched %d files", len(result.details))
	}

	return change
}
