package agent

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// chat はAIと対話する
// PlanModeEnabled に応じて Plan Mode または通常モードで処理
func (a *Agent) chat(input string) {
	a.chatInternal(input, nil)
}

// ChatOnce は単一クエリを1ターンだけ実行してエラーを返す（--once 用）
// stdin を読まず、REPL ループに入らない。chatCore を共有する。
func (a *Agent) ChatOnce(input string) error {
	return a.chatCore(input, nil, true)
}

// chatInternal はAIと対話する内部実装（対話モード用ラッパー）
// image が nil でない場合は画像付きメッセージとして処理
func (a *Agent) chatInternal(input string, image *api.ImageData) {
	_ = a.chatCore(input, image, false)
}

// chatCore は chat / ChatOnce の共有実装
// oneShot=true の場合: エラーを返し、対話向け後処理（usage表示・圧縮・context提案）をスキップ
func (a *Agent) chatCore(input string, image *api.ImageData, oneShot bool) error {
	a.SetStatus(StateRunning, "Processing request", "処理中", "Wait for response", "応答を待ってください")
	a.resetSearchCodeTurnObservability()

	// タスク開始: changeStack のオフセットを記録（タスク単位のサマリー表示用）
	prevChanges := len(a.changeStack) - a.taskChangeOffset
	a.taskChangeOffset = len(a.changeStack)

	// completion hook の git diff 空チェック判定用ベースハッシュを記録
	a.taskBaseCommitHash = ""
	a.taskTestResult = nil
	a.taskTestCommand = ""
	if out, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		a.taskBaseCommitHash = strings.TrimSpace(string(out))
	}

	if prevChanges > 0 {
		yellow.Fprintf(a.output(), "⚠️  %d uncommitted changes from previous task\n", prevChanges)
		_, _ = fmt.Fprintln(a.output(), "💡 Run /commit or git commit to keep changes separate")
		_, _ = fmt.Fprintln(a.output())
	}

	// GitHub MCP ヒントを追加（GitHub関連リクエストの場合）
	input = a.AddGitHubHint(input)

	// 入力に応じて project rules/context と project map を差し替える。
	a.refreshProjectPrompt(input)

	// セッションに保存
	if a.session != nil {
		a.appendSessionMessage("user", input, a.CurrentModel)
	}

	// 統計情報更新: Userメッセージ数をカウント
	var startStats SessionStats
	if a.Stats != nil {
		a.statsMu.Lock()
		a.Stats.UserMessages++
		startStats = *a.Stats
		a.statsMu.Unlock()
	}

	// タイムアウト付きコンテキスト作成
	cfg := a.cfg()
	timeout := time.Duration(cfg.APIRetry.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	a.lastCancelReason = ""
	a.cancelFunc = cancel
	a.requestCtx = ctx
	a.debugCancelf("request started (timeout=%s, model=%s, provider=%s)", timeout, a.CurrentModel, a.ProviderName)
	defer func() {
		a.debugCancelf("request finished (ctx_err=%v, cancel_reason=%q)", ctx.Err(), a.lastCancelReason)
		a.requestCtx = nil
		a.cancelFunc = nil
		a.lastCancelReason = ""
	}()

	// トークン使用量の警告チェック（API呼び出し前、対話モードのみ）
	if !oneShot {
		a.checkTokenWarning()
	}

	// excludedTools を設定（oneShot 時は defer で復元）
	if oneShot {
		prev := a.registry().GetExcludedTools()
		defer a.registry().SetExcludedTools(prev)
	}

	editToolMode := ResolveEditToolMode(a.ProviderName, a.CurrentModel)

	var err error
	if a.PlanModeEnabled {
		// Plan Mode: planning 系ツールを有効化しつつ、編集ツールの排他制御は維持
		a.registry().SetExcludedTools(planModeExcludedTools(editToolMode))
		err = a.RunPlanMode(ctx, input)
	} else {
		// Normal Mode: planning 系ツールを除外しつつ、編集ツールの排他制御を維持
		a.registry().SetExcludedTools(normalModeExcludedTools(editToolMode))
		err = a.runNormalMode(ctx, input, image)
	}

	if err != nil {
		// oneShot: エラーをそのまま返す（対話的リトライ不要）
		if oneShot {
			a.ui().StopSpinner()
			return err
		}

		if errors.Is(err, context.Canceled) {
			yellow.Fprintln(a.output(), "\n⚠️  Response interrupted")
		} else {
			// トークン上限エラーの場合は自動圧縮+リトライを試みる
			if token.IsTokenLimitError(err) {
				// リトライ関数を定義
				retryFunc := func() error {
					// 同じコンテキストで再実行（除外設定は chatCore 冒頭で設定済み）
					ctx := context.Background()
					if a.PlanModeEnabled {
						return a.RunPlanMode(ctx, input)
					} else {
						return a.runNormalMode(ctx, input, image)
					}
				}

				// 自動圧縮+リトライを試みる
				if a.handleTokenLimitErrorWithRetry(err, retryFunc, a.PlanModeEnabled) {
					// リトライ成功時はここで終了
					a.ui().StopSpinner()
					a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
					return nil
				}
				// リトライ失敗時はエラーメッセージを表示（handleTokenLimitErrorWithRetry内で既に表示済み）
				red.Fprintf(a.output(), "Error: %v\n", err)
			} else {
				// トークン上限以外のエラー
				red.Fprintf(a.output(), "Error: %v\n", err)
			}
		}
		a.ui().StopSpinner()
		reason := a.statusReasonForError(err)
		a.SetStatus(StateAborted, reason, reason, "Try again", "再試行してください")
		return nil
	}

	// oneShot: 対話向け後処理をスキップ
	if oneShot {
		return nil
	}

	// リクエスト完了時の usage 表示
	a.printTaskUsage(startStats)

	// 自動圧縮チェック（成功時）
	a.maybeAutoCompress()

	// ターン完了後のcontext提案メッセージ表示
	currentTokens := a.EstimateTokens()
	contextK := currentTokens / 1000

	baseTokens := a.EstimateSystemPromptTokens()
	if a.CurrentProvider != nil && a.CurrentProvider.IsFunctionCallingEnabled() {
		baseTokens += a.estimateToolDefinitionTokens()
	}

	if currentTokens <= baseTokens {
		dim.Fprintf(a.output(), "💡 Context %dK - clean state ✓\n", contextK)
	} else {
		saved := currentTokens - baseTokens
		pricing := GetPricingInfo(a.ProviderName, a.CurrentModel)
		if pricing.InputCostPerM > 0 {
			savingPerTurn := float64(saved) / 1_000_000.0 * pricing.InputCostPerM * 0.5
			if savingPerTurn < 0.01 {
				dim.Fprintf(a.output(), "💡 Context %dK - clean state ✓\n", contextK)
			} else {
				dim.Fprintf(a.output(), "💡 Context %dK - /clear saves ~$%.2f/turn, /compress keeps key context\n", contextK, savingPerTurn)
			}
		} else {
			dim.Fprintf(a.output(), "💡 Context %dK - /clear or /compress to reduce context\n", contextK)
		}
	}

	a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
	return nil
}

// runNormalMode は通常モードでの処理（Plan Mode OFF 時）
// ツールを個別に確認しながら実行するループ（自動リトライ対応）
func (a *Agent) runNormalMode(ctx context.Context, input string, image *api.ImageData) error {
	return newTurnRunner(a, ctx).RunNormalMode(input, image)
}

// showTaskSummary は現在タスクの changeStack からサマリーを生成して表示
// taskChangeOffset 以降の変更のみを対象とする（Issue #118: タスク分離）
func (a *Agent) showTaskSummary() {
	taskChanges := a.changeStack[a.taskChangeOffset:]
	if len(taskChanges) == 0 {
		return
	}

	ts := ui.NewTaskSummary()
	for _, change := range taskChanges {
		if len(change.Details) > 0 {
			for _, detail := range change.Details {
				action := detail.Action
				if action == "" {
					action = ui.InferAction(change.Tool)
				}
				ts.AddChange(detail.FilePath, action, detail.LinesAdded, detail.LinesRemoved)
			}
			continue
		}
		action := ui.InferAction(change.Tool)
		ts.AddChange(change.FilePath, action, change.LinesAdded, change.LinesRemoved)
	}
	if a.taskTestResult != nil {
		ts.SetTestResult(*a.taskTestResult)
	}
	if a.taskTestCommand != "" {
		ts.SetTestCommand(a.taskTestCommand)
	}

	_, _ = fmt.Fprint(a.output(), ts.Render())
}

func (a *Agent) assistantUpdateMode() string {
	if a == nil {
		return api.AssistantUpdatesVerbose
	}
	mode := strings.TrimSpace(a.cfg().Output.AssistantUpdates)
	if mode != "" {
		return api.AssistantUpdateModeFromContext(api.WithAssistantUpdateMode(context.Background(), mode))
	}
	if a.PlanModeEnabled {
		return api.AssistantUpdatesVerbose
	}
	return api.AssistantUpdatesPhase
}

func (a *Agent) shouldStreamAssistantText() bool {
	return a.assistantUpdateMode() == api.AssistantUpdatesVerbose
}

func (a *Agent) shouldEmitAssistantPhaseUpdate() bool {
	return a.assistantUpdateMode() == api.AssistantUpdatesPhase
}

func (a *Agent) printFinalAssistantResponse(response string) {
	if a == nil || a.shouldStreamAssistantText() {
		return
	}
	display := strings.TrimSpace(response)
	if display == "" {
		return
	}
	_, _ = fmt.Fprintf(a.output(), "\n💬 %s\n", display)
}

func (a *Agent) maybePrintAssistantPhaseUpdate(response string, toolCalls []*tools.ToolCall) {
	if a == nil || !a.shouldEmitAssistantPhaseUpdate() {
		return
	}

	explanation, _ := extractExplanationAndTool(response)
	summary := firstAssistantSummaryLine(explanation)
	if summary == "" {
		summary = summarizeToolCallsForPhase(toolCalls)
	}
	if summary == "" {
		return
	}
	_, _ = fmt.Fprintf(a.output(), "\n💬 %s\n", summary)
}

func firstAssistantSummaryLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimLeft(line, "-*0123456789. ")
		line = strings.TrimSpace(line)
		runes := []rune(line)
		if len(runes) > 120 {
			return string(runes[:120]) + "..."
		}
		return line
	}
	return ""
}

func summarizeToolCallsForPhase(toolCalls []*tools.ToolCall) string {
	if len(toolCalls) == 0 {
		return ""
	}

	counts := make(map[string]int, len(toolCalls))
	order := make([]string, 0, len(toolCalls))
	for _, tc := range toolCalls {
		name := tc.Tool
		if _, ok := counts[name]; !ok {
			order = append(order, name)
		}
		counts[name]++
	}

	parts := make([]string, 0, len(order))
	for _, name := range order {
		if counts[name] == 1 {
			parts = append(parts, name)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s x%d", name, counts[name]))
	}
	return "Phase: " + strings.Join(parts, ", ")
}

func execToolCallsSummaryInput(toolCalls []*tools.ToolCall) []*tools.ToolCall {
	execToolCalls := make([]*tools.ToolCall, 0, len(toolCalls))
	for _, tc := range toolCalls {
		if tc.Tool == "create_plan" {
			continue
		}
		execToolCalls = append(execToolCalls, tc)
	}
	return execToolCalls
}

// chatWithImage は画像付きメッセージでAIと対話する
func (a *Agent) chatWithImage(input string, image *api.ImageData) {
	// プロバイダーが画像対応かチェック
	if !a.CurrentProvider.SupportsImages() {
		yellow.Fprintf(a.output(), "Warning: %s does not support images. The image will be ignored.\n", a.CurrentProvider.Name())
		a.chat(input)
		return
	}

	// 画像情報をログ
	green.Fprintf(a.output(), "🖼️  Sending image: %s (%s)\n", image.Path, api.FormatImageSize(image.Size))

	a.chatInternal(input, image)
}

// printTaskUsage はタスク全体の usage を表示
func (a *Agent) printTaskUsage(startStats SessionStats) {
	a.statsMu.Lock()
	if a.Stats == nil {
		a.statsMu.Unlock()
		return
	}

	turnUsage := api.Usage{
		InputTokens:         a.Stats.InputTokens - startStats.InputTokens,
		OutputTokens:        a.Stats.OutputTokens - startStats.OutputTokens,
		ThinkingTokens:      a.Stats.ThinkingTokens - startStats.ThinkingTokens,
		CachedInputTokens:   a.Stats.CachedInputTokens - startStats.CachedInputTokens,
		CacheCreationTokens: a.Stats.CacheCreationTokens - startStats.CacheCreationTokens,
	}
	costDiff := a.Stats.EstimatedCost() - startStats.EstimatedCost()
	a.Stats.LastTurnUsage = &turnUsage
	a.Stats.LastTurnCost = costDiff
	inDiff := turnUsage.InputTokens
	outDiff := turnUsage.OutputTokens
	a.statsMu.Unlock()

	total := inDiff + outDiff
	if total == 0 {
		return
	}

	// ✓ を緑色で表示、残りはdimまたは通常色
	green.Fprint(a.output(), "✓ ")
	if strings.ToLower(a.ProviderName) == "ollama" {
		// Ollama の場合はコスト非表示
		_, _ = fmt.Fprintf(a.output(), "In: %s + Out: %s = %s tok\n",
			FormatNumber(inDiff),
			FormatNumber(outDiff),
			FormatNumber(total))
	} else {
		_, _ = fmt.Fprintf(a.output(), "In: %s + Out: %s = %s tok ",
			FormatNumber(inDiff),
			FormatNumber(outDiff),
			FormatNumber(total))
		dim.Fprintf(a.output(), "(~$%.4f)\n", costDiff)
	}
}

// normalModeRetryInstruction は Normal Mode の retry プロンプトを段階的に返す。
func normalModeRetryInstruction(attempt int) string {
	const constraint = `
Reuse information already obtained from the failed command/output.
Do not re-run broad investigation unless the failure output is insufficient.`

	switch {
	case attempt <= 1:
		return `Do not patch blindly.
First, identify the concrete root cause in 1-2 sentences using the existing failure output.
Point to the exact file/function/command causing the failure.
Then make the smallest plausible fix and verify it immediately.

Do not write a new test yet unless the root cause is still unclear or verification is otherwise unreliable.` + constraint
	case attempt == 2:
		return `The previous fix did not work.

Do not repeat the same fix pattern.
Briefly explain why the previous attempt failed.
If the root cause is still uncertain, create the smallest possible reproduction or targeted test via bash.
If the root cause is already clear, skip test creation and apply the next evidence-based fix directly.

Then verify again.` + constraint
	default:
		return `Multiple retries have failed.

Your current approach is not working.
Explain which assumption was wrong.
Change strategy fundamentally and avoid repeating the same edit pattern.
Choose the smallest different approach that can validate the new hypothesis quickly.` + constraint
	}
}
