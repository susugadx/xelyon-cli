package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// promptFailureActionWithSelector は失敗時の Selector UI を表示
// retryCount はこれまでの自動リトライ回数（表示用）
// 戻り値: (アクション, コメント文字列) - コメントアクション時のみ第2引数が非空
func promptFailureActionWithSelector(promptIO ui.PromptIO, step *plan.PlanStep, result string, reason string, retryCount int) (plan.FailureAction, string) {
	promptIO = ui.NormalizePromptIO(promptIO)
	out := promptIO.Out

	_, _ = fmt.Fprintln(out)
	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if retryCount > 0 {
		red.Fprintf(out, "❌ Step %d Failed (stalled after %d retries)\n", step.ID, retryCount)
	} else {
		red.Fprintf(out, "❌ Step %d Failed: %s\n", step.ID, step.Description)
	}
	red.Fprintf(out, "   Reason: %s\n", reason)
	cyan.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// エラー内容を表示
	lines := strings.Split(result, "\n")
	maxShow := config.ErrorOutputMaxLines
	if len(lines) > maxShow {
		_, _ = fmt.Fprintln(out)
		yellow.Fprintf(out, "Error output (last %d lines):\n", maxShow)
		for _, line := range lines[len(lines)-maxShow:] {
			_, _ = fmt.Fprintf(out, "  %s\n", line)
		}
		_, _ = fmt.Fprintf(out, "  ... (%d more lines above)\n", len(lines)-maxShow)
	} else {
		_, _ = fmt.Fprintln(out)
		yellow.Fprintln(out, "Error output:")
		for _, line := range lines {
			_, _ = fmt.Fprintf(out, "  %s\n", line)
		}
	}

	_, _ = fmt.Fprintln(out)
	cyan.Fprintln(out, "? What do you want to do?")
	_, _ = fmt.Fprintln(out)

	// Selector 風のオプション表示
	options := []ui.SelectOption{
		{Label: "Retry", Description: "Try again with different approach", Value: "retry"},
		{Label: "Comment", Description: "Give instructions", Value: "comment"},
		{Label: "Skip", Description: "Continue to next step", Value: "skip"},
		{Label: "Abort", Description: "Stop execution", Value: "abort"},
	}

	return runFailureSelectorUI(promptIO, options)
}

// runFailureSelectorUI は失敗時の選択UIを実行
func runFailureSelectorUI(promptIO ui.PromptIO, options []ui.SelectOption) (plan.FailureAction, string) {
	promptIO = ui.NormalizePromptIO(promptIO)
	out := promptIO.Out

	// オプションを表示
	for i, opt := range options {
		marker := "  "
		if i == 0 {
			marker = "▶ "
		}
		_, _ = fmt.Fprintf(out, "  %s%d. %-8s - %s\n", marker, i+1, opt.Label, opt.Description)
	}

	// ヒント表示
	_, _ = fmt.Fprintln(out)
	dim.Fprintf(out, "  (Enter=Retry, 2/c=Comment, 3/s=Skip, 4/a=Abort)\n")
	_, _ = fmt.Fprintln(out)

	// 入力を受け付け
	for {
		cyan.Fprint(out, "Choice [1]: ")

		response, err := promptIO.ReadSimpleLine()
		if err != nil {
			return plan.FailureActionAbort, ""
		}
		response = strings.ToLower(strings.TrimSpace(response))

		switch response {
		case "", "1", "r", "retry":
			green.Fprintln(out, "✓ Retry")
			return plan.FailureActionRetry, ""
		case "2", "c", "comment":
			green.Fprintln(out, "✓ Comment")
			comment, _ := common.ReadMultiLineCommentWithIO(promptIO)
			return plan.FailureActionComment, comment
		case "3", "s", "skip":
			green.Fprintln(out, "✓ Skip")
			return plan.FailureActionSkip, ""
		case "4", "a", "abort":
			green.Fprintln(out, "✓ Abort")
			return plan.FailureActionAbort, ""
		default:
			yellow.Fprintln(out, "Invalid input. Please enter 1/2/3/4 or r/c/s/a.")
		}
	}
}
