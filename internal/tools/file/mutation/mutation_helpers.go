package mutation

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/tools/file/pathpolicy"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type fileMutationContext struct {
	promptIO ui.PromptIO
	out      common.Output
	cfg      *config.Config
	path     string
	absPath  string
}

type mutationConfirmHandlers struct {
	onComment func(comment string) fileMutationResult
	onCancel  func() fileMutationResult
}

func prepareFileMutation(promptIO ui.PromptIO, options common.ConfirmOptions, path, emptyMessage string) (fileMutationContext, fileMutationResult, error) {
	promptIO = ui.NormalizePromptIO(promptIO)
	out := common.NewOutput(promptIO.Out, promptIO.Err)

	absPath, errResult := pathpolicy.ResolveValidatedPath(out, path, emptyMessage)
	if errResult != "" {
		return fileMutationContext{}, newErrorMutationResult(errResult), nil
	}

	return fileMutationContext{
		promptIO: promptIO,
		out:      out,
		cfg:      options.Config,
		path:     path,
		absPath:  absPath,
	}, fileMutationResult{}, nil
}

func buildCommentResponse(toolName, comment string, nextActions ...string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "[COMMENT] User provided feedback for %s.\n\n", toolName)
	sb.WriteString("Comment:\n")
	sb.WriteString(strings.TrimSpace(comment))
	sb.WriteString("\n\nNext actions:\n")
	for _, action := range nextActions {
		sb.WriteString("- ")
		sb.WriteString(action)
		sb.WriteString("\n")
	}
	fmt.Fprintf(&sb, "\nIMPORTANT: Do NOT %s until the user approves.", mutationVerb(toolName))
	return strings.TrimSpace(sb.String())
}

func mutationVerb(toolName string) string {
	switch toolName {
	case "write_file":
		return "write the file"
	case "delete_file":
		return "delete the file"
	default:
		return "apply the change"
	}
}

func confirmFileMutation(ctx fileMutationContext, options common.ConfirmOptions, toolName, message string, handlers mutationConfirmHandlers) (fileMutationResult, bool) {
	dec := common.ConfirmToolAction(ctx.promptIO, options, toolName, message, common.ToolConfirmContext{TargetPath: ctx.absPath})
	switch dec.Action {
	case common.ConfirmYes:
		return fileMutationResult{}, true
	case common.ConfirmComment:
		if handlers.onComment != nil {
			return handlers.onComment(dec.Comment), false
		}
		return newCommentMutationResult("Cancelled by user"), false
	default:
		if handlers.onCancel != nil {
			return handlers.onCancel(), false
		}
		return newCancelledMutationResult("Cancelled by user"), false
	}
}

func showCappedSinglePatchPreview(out common.Output, cfg *config.Config, preview ui.PatchFilePreview, contentLines []string, lineType rune) {
	capped := buildCappedPatchPreviewLines(contentLines, lineType, resolvePreviewLineCap(cfg))
	if len(preview.Hunks) == 0 {
		preview.Hunks = []ui.PatchHunkPreview{{
			StartLine: 1,
			Lines:     capped.lines,
		}}
	}
	if lineType == '+' && preview.Added == 0 {
		preview.Added = capped.totalLines
	}
	if lineType == '-' && preview.Removed == 0 {
		preview.Removed = capped.totalLines
	}

	ui.ShowSinglePatchPreview(out.StdoutWriter(), preview)
	if capped.truncated {
		out.Dim.Printf("  preview truncated: showing first %d of %d lines\n", len(capped.lines), capped.totalLines)
		if capped.truncatedByBytes {
			out.Dim.Printf("  preview truncated at %dKB\n", maxFullBodyPreviewBytes/1024)
		}
	}
}
