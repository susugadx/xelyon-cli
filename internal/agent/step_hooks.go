package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// runStepCompleteHooks は hooks.on_step_complete に定義されたシェルコマンドを実行する。
// テンプレート変数 {{step_id}}, {{step_description}}, {{step_status}} を展開する。
// いずれかのコマンドが失敗した場合、needsContinue=true と AI 向けのフィードバックを返す。
// すべて成功した場合は needsContinue=false を返す。
func (a *Agent) runStepCompleteHooks(stepID int, stepDescription, stepStatus string) (needsContinue bool, feedback string) {
	globalCfg := a.cfg()
	hooks := globalCfg.Hooks.OnStepComplete
	out := a.output()

	// No step hooks → nothing to verify
	if len(hooks) == 0 {
		return false, ""
	}

	timeout := time.Duration(globalCfg.Hooks.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	for _, cmd := range hooks {
		// テンプレート変数を展開
		expandedCmd := expandStepTemplate(cmd, stepID, stepDescription, stepStatus)
		yellow.Fprintf(out, "🏁 Running step complete hook: %s\n", expandedCmd)

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		proc := exec.CommandContext(ctx, "bash", "-c", expandedCmd)

		// 作業ディレクトリを設定
		if cwd, err := os.Getwd(); err == nil {
			proc.Dir = cwd
		}

		// stdout と stderr を結合して取得
		output, err := proc.CombinedOutput()
		cancel()

		if err != nil {
			// 出力を最大2000文字に切り詰め
			outputStr := string(output)
			if len(outputStr) > 2000 {
				outputStr = outputStr[:2000] + "\n... (truncated)"
			}

			// 終了コードを取得
			exitCode := -1
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}

			red.Fprintf(out, "  Step hook failed (exit code %d): %s\n", exitCode, expandedCmd)

			feedback = fmt.Sprintf(`[SYSTEM] Step %d completion verification failed. Hook command %q failed (exit code %d):

%s

Please fix these errors before proceeding to the next step. Do NOT skip these issues.`, stepID, expandedCmd, exitCode, outputStr)

			return true, feedback
		}

		green.Fprintf(out, "  Step hook passed: %s\n", expandedCmd)
	}

	return false, ""
}

// expandStepTemplate はステップ完了フック用のテンプレート変数を展開する
func expandStepTemplate(cmd string, stepID int, stepDescription, stepStatus string) string {
	replacements := map[string]string{
		"{{step_id}}":          fmt.Sprintf("%d", stepID),
		"{{step_description}}": stepDescription,
		"{{step_status}}":      stepStatus,
	}

	result := cmd
	for key, value := range replacements {
		result = strings.ReplaceAll(result, key, value)
	}
	return result
}

// runStepCompleteHooksWithRetry は step complete hooks を最大 MaxRetry 回実行する。
// フック失敗時は AI にフィードバックして修正を試み、再実行する。
// 戻り値: hooks がすべてパスした場合 true、max_retry 到達で打ち切った場合 false。
func (a *Agent) runStepCompleteHooksWithRetry(ctx context.Context, stepID int, stepDescription, stepStatus string) bool {
	globalCfg := a.cfg()
	out := a.output()

	// No step hooks configured → nothing to verify
	if len(globalCfg.Hooks.OnStepComplete) == 0 {
		return true
	}

	maxRetry := globalCfg.Hooks.MaxRetry
	if maxRetry <= 0 {
		maxRetry = 3
	}

	for attempt := 1; attempt <= maxRetry; attempt++ {
		needsContinue, feedback := a.runStepCompleteHooks(stepID, stepDescription, stepStatus)
		if !needsContinue {
			return true // all hooks passed
		}

		if attempt >= maxRetry {
			yellow.Fprintf(out, "⚠️  Step hook retry limit reached (%d/%d). Proceeding to next step.\n", attempt, maxRetry)
			return false
		}

		// AI にフィードバックして修正を試みる
		yellow.Fprintf(out, "⚠️  Step hook failed (%d/%d). Asking AI to fix...\n", attempt, maxRetry)
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: feedback,
		})

		response, err := a.CurrentProvider.ChatWithTools(a.requestContext(ctx), a.SystemPrompt, a.History, a.CurrentModel)
		if err != nil {
			yellow.Fprintf(out, "⚠️  AI fix attempt failed: %v\n", err)
			return false
		}

		// ツール呼び出しをパースして履歴管理
		toolCalls := a.parseToolCalls(response)

		// FC rescue: テキストから抽出された toolCall にダミー ID を注入
		for i, tc := range toolCalls {
			if tc.ID == "" {
				toolCalls[i].ID = fmt.Sprintf("call_rescue_%03d", i+1)
			}
		}

		if len(toolCalls) > 0 {
			execToolCalls := toolCalls

			// バッチで assistant メッセージを追加し、ツールを実行
			a.addToolCallsToHistory(response, execToolCalls)
			for _, tc := range execToolCalls {
				a.executeToolOnly(tc)
			}
		} else {
			// テキストのみの応答
			a.History = append(a.History, api.Message{
				Role:             "assistant",
				Content:          response,
				ReasoningContent: a.getLastReasoningContent(),
			})
		}
	}

	return false
}
