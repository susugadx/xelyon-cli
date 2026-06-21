package mutation

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const (
	strReplaceModeDefault   = ""
	strReplaceModeLineRange = "line range"
	strReplaceModeBatch     = "batch"
)

type strReplaceDiffPreview struct {
	targetPath         string
	removedLines       int
	addedLines         int
	before             string
	after              string
	lineNumOffset      int
	largeChangeWarning string
}

func buildStrReplaceConfirmHandlers(out common.Output, path, mode string) mutationConfirmHandlers {
	return mutationConfirmHandlers{
		onComment: func(comment string) fileMutationResult {
			return newCommentMutationResult(buildDeferredStrReplaceResult("[COMMENT]", mode, path, comment))
		},
		onCancel: func() fileMutationResult {
			out.Yellow.Println(resolveStrReplaceCancelMessage(mode))
			return newCancelledMutationResult(buildDeferredStrReplaceResult("[CANCELLED]", mode, path, ""))
		},
	}
}

func resolveStrReplaceCancelMessage(mode string) string {
	if mode == strReplaceModeBatch {
		return "⚠️  User cancelled the batch replacement"
	}
	return "⚠️  User cancelled the replacement"
}

func showStrReplaceDiffPreview(ctx fileMutationContext, preview strReplaceDiffPreview) {
	if ctx.out.SuppressStdout() {
		return
	}

	w := ctx.out.StdoutWriter()
	ctx.out.Println()
	ui.FileOpHeader(w, "str_replace", preview.targetPath)
	ui.FileOpStatsLine(w, preview.removedLines, preview.addedLines)

	if strings.TrimSpace(preview.largeChangeWarning) != "" {
		ctx.out.Yellow.Println(preview.largeChangeWarning)
	}

	opts := &ui.DiffOptions{
		ContextLines:  ctx.cfg.Diff.ContextLines,
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: ctx.cfg.Diff.MaxTotalLines,
		LineNumOffset: preview.lineNumOffset,
	}
	ui.ShowColoredDiffToWriter(ctx.out.StdoutWriter(), preview.before, preview.after, opts)
}
