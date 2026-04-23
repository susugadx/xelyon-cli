package applypatch

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

const (
	applyPatchCommentTemplate = `[COMMENT] User provided feedback for apply_patch.

Comment:
%s

Next actions:
- Read the affected files or patch context again.
- Revise the patch and ask for approval before applying it.

IMPORTANT: Do NOT apply the patch until the user approves.`
	applyPatchCancelledMessage = "[CANCELLED] apply_patch was not approved"
)

func (t *ApplyPatchTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	patchText, ok := extractApplyPatchText(args)
	if !ok {
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

	approved, cancelMessage := resolveApplyPatchDecision(confirmApplyPatch(execCtx, parsed, autoApproved))
	if !approved {
		return cancelMessage, nil, nil
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

func extractApplyPatchText(args map[string]string) (string, bool) {
	patchText := args["patch"]
	return patchText, patchText != ""
}

func resolveApplyPatchDecision(decision common.ConfirmDecision) (bool, string) {
	switch decision.Action {
	case common.ConfirmYes:
		return true, ""
	case common.ConfirmComment:
		return false, formatApplyPatchComment(decision.Comment)
	default:
		return false, applyPatchCancelledMessage
	}
}

func formatApplyPatchComment(comment string) string {
	return fmt.Sprintf(applyPatchCommentTemplate, strings.TrimSpace(comment))
}
