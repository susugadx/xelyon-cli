package agent

import (
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// promptConfirmWithRuntime は slash command 用の確認を runtime の入出力で行う。
// NOTE: コメント入力は AI ツール確認専用のため、ここではキャンセルとして扱う。
func promptConfirmWithRuntime(runtime *ui.Runtime, prompt string) bool {
	if hasTUIRuntimePrompter(runtime) {
		return true
	}
	result := common.ConfirmInteractiveWithIO(runtime.PromptIO(), prompt)
	if result.Action == "comment" {
		yellow.Fprintln(runtime.Output(), "⚠️  Comment mode is for AI tool confirmations only. Treating as cancel.")
		return false
	}
	return result.Action == "yes"
}

func hasTUIRuntimePrompter(runtime *ui.Runtime) bool {
	if runtime == nil {
		return false
	}
	_, ok := runtime.Prompter().(ui.CommandConfirmBypassPrompter)
	return ok
}
