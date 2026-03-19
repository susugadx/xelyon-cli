package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	promptnormal "github.com/susugadx/xelyon-cli/internal/prompt/normal"
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

	// タスク開始: changeStack のオフセットを記録（タスク単位のサマリー表示用）
	prevChanges := len(a.changeStack) - a.taskChangeOffset
	a.taskChangeOffset = len(a.changeStack)

	// read tracker をリセット（新しいタスクで過去のカウントを引き継がない）
	a.readTracker.reset()

	// completion hook の git diff 空チェック判定用ベースハッシュを記録
	a.taskBaseCommitHash = ""
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
		a.session.AddMessage("user", input, a.CurrentModel)
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

	var err error
	if a.PlanModeEnabled {
		// Plan Mode: planning 系ツールを有効化
		a.registry().ClearExcludedTools()
		err = a.RunPlanMode(ctx, input)
	} else {
		// Normal Mode: planning 系ツールを除外（FC 定義から非表示）
		a.registry().SetExcludedTools(prompt.PlanningToolNames)
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
	// 通常モード用の指示を追加（prompt パッケージから取得）
	normalModeInput := input + promptnormal.NormalModePrompt

	// 履歴に追加
	a.History = append(a.History, api.Message{Role: "user", Content: normalModeInput})

	cfg := a.cfg()
	hardLimit := normalizeToolLoopLimit(cfg.General.ToolLoopLimit)
	var lastToolCall *tools.ToolCall
	var sameCallCount int

	// 自動リトライ設定
	autoRetryMax := cfg.PlanMode.MaxRetry
	retryCount := 0

	var completionVerified bool // 完了検証ガード（タスク内1回限り）
	var hookRetryCount int      // フック失敗リトライカウンター

	// テキスト計画検出 → 直接実行リダイレクト（Normal Mode では create_plan 非表示のため）
	var textPlanRedirectCount int
	const maxTextPlanRedirects = 2 // ソフトリダイレクト回数
	const maxTextPlanHardLimit = 5 // これ以上繰り返したら break（無限ループ防止）

	var reachedHardLimit bool

	for i := 0; ; i++ {
		if hardLimit > 0 && i >= hardLimit {
			reachedHardLimit = true
			break
		}
		if hardLimit == 0 {
			emitLoopWarning(a, i)
		}

		a.refreshProjectPromptIfDirty(input)
		effectivePrompt := prompt.StripPlanningReferences(a.SystemPrompt)

		// API呼び出し
		var response string
		var err error
		requestCtx := a.requestContext(ctx)
		if i == 0 && image != nil {
			inputWithPrompt := input + promptnormal.NormalModePrompt
			compactedHistory, metrics := CompactOldToolResults(a.History[:len(a.History)-1], DefaultMaxLines, DefaultHeadLines, DefaultTailLines)
			a.addCompactionMetrics(metrics)
			response, err = a.CurrentProvider.ChatWithImage(
				requestCtx, effectivePrompt, compactedHistory, inputWithPrompt, image, a.CurrentModel,
			)
		} else {
			compactedHistory, metrics := CompactOldToolResults(a.History, DefaultMaxLines, DefaultHeadLines, DefaultTailLines)
			a.addCompactionMetrics(metrics)
			response, err = a.CurrentProvider.ChatWithTools(
				requestCtx,
				effectivePrompt,
				compactedHistory,
				a.CurrentModel,
			)
			// tool_choice が設定されていた場合は解除
			if tc, ok := a.CurrentProvider.(interface{ ClearToolChoice() }); ok {
				tc.ClearToolChoice()
			}
		}
		if err != nil {
			a.ui().StopSpinner()
			return fmt.Errorf("API call failed: %w", err)
		}

		// ツール呼び出しをパース（Plan JSON チェックより先に実行 — create_plan の FC が誤検出されるのを防止）
		toolCalls := a.parseToolCalls(response)

		// FC rescue: テキストから抽出された toolCall にダミー ID を注入
		// これにより下流の処理が FC 成功時と同じパス（role:"tool"）を通る
		for i, tc := range toolCalls {
			if tc.ID == "" {
				toolCalls[i].ID = fmt.Sprintf("call_rescue_%03d", i+1)
			}
		}

		// Plan JSON が検出された場合、パースして step-by-step 実行を試みる（ツール呼び出しがない場合のみ）
		if len(toolCalls) == 0 && plan.ContainsPlanJSON(response) {
			// FC失敗フォールバック: テキスト出力された Plan JSON を抽出して実行
			if planJSON := plan.ExtractPlanJSON(response); planJSON != "" {
				if p, err := plan.ParsePlan(planJSON); err == nil && len(p.Steps) > 0 {
					yellow.Fprintf(a.output(), "📋 FC fallback: extracted %d-step plan from text. Switching to step-by-step...\n", len(p.Steps))
					a.History = append(a.History, api.Message{
						Role:             "assistant",
						Content:          response,
						ReasoningContent: a.getLastReasoningContent(),
					})
					if err := a.runImplementationPhase(ctx, p); err != nil {
						return err
					}
					a.runCompletionHooksWithRetry(ctx)
					a.showTaskSummary()
					return nil
				}
			}
			// パース失敗 → 直接ツール実行を促す
			yellow.Fprintln(a.output(), "⚠️  Plan JSON detected but parse failed. Execute tools directly.")
			a.History = append(a.History, api.Message{
				Role:             "assistant",
				Content:          response,
				ReasoningContent: a.getLastReasoningContent(),
			})
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: "[SYSTEM] You are in NORMAL MODE. Do NOT output JSON directly. Execute the required changes directly using tools (read_file, str_replace, etc).",
			})
			continue
		}

		// デバッグログ
		if os.Getenv("XELYON_DEBUG_TOOLS") == "1" {
			_, _ = fmt.Fprintf(a.errorOutput(), "[DEBUG Tools] Response length: %d, ToolCalls found: %d\n", len(response), len(toolCalls))
			if len(response) < config.DebugPreviewLen {
				_, _ = fmt.Fprintf(a.errorOutput(), "[DEBUG Tools] Response: %s\n", response)
			} else {
				_, _ = fmt.Fprintf(a.errorOutput(), "[DEBUG Tools] Response (first %d): %s...\n", config.DebugPreviewLen, response[:config.DebugPreviewLen])
			}
			for i, tc := range toolCalls {
				_, _ = fmt.Fprintf(a.errorOutput(), "[DEBUG Tools] ToolCall[%d]: tool=%s, args=%v\n", i, tc.Tool, tc.Args)
			}
		}

		// ツール呼び出しなし = 通常の回答
		if len(toolCalls) == 0 {
			// テキスト計画の検出 → 直接ツール実行を要求
			// Normal Mode では create_plan が非表示のため、直接実行を促す
			// ただし完了宣言を含むレスポンスはスキップ（まとめの番号リストを誤検知しない）
			steps := extractTextPlan(response)
			if !containsCompletionDeclaration(response) && len(steps) >= 5 && isActionPlan(steps) {
				textPlanRedirectCount++

				// ハードリミット: これ以上繰り返してもツール使用に移行しない → break してユーザーに返す
				if textPlanRedirectCount > maxTextPlanHardLimit {
					yellow.Fprintf(a.output(), "⚠️  Text plan detected %d times without tool use. Returning response to user.\n", textPlanRedirectCount)
					a.History = append(a.History, api.Message{
						Role:             "assistant",
						Content:          response,
						ReasoningContent: a.getLastReasoningContent(),
					})
					break
				}

				// ソフトリダイレクト上限後: より強い指示
				if textPlanRedirectCount > maxTextPlanRedirects {
					yellow.Fprintf(a.output(), "⚠️  Text plan detected %d times. Forcing direct execution.\n", textPlanRedirectCount)
					a.History = append(a.History, api.Message{
						Role:             "assistant",
						Content:          response,
						ReasoningContent: a.getLastReasoningContent(),
					})
					a.History = append(a.History, api.Message{
						Role:    "user",
						Content: "[SYSTEM] STOP planning. Pick the FIRST change and execute it NOW using the appropriate tool (read_file, str_replace, etc). One tool call, no explanation.",
					})
					continue
				}

				yellow.Fprintf(a.output(), "⚠️  Text plan detected (%d steps). Execute tools directly instead. (%d/%d)\n",
					len(steps), textPlanRedirectCount, maxTextPlanRedirects)
				a.History = append(a.History, api.Message{
					Role:             "assistant",
					Content:          response,
					ReasoningContent: a.getLastReasoningContent(),
				})
				a.History = append(a.History, api.Message{
					Role:    "user",
					Content: "[SYSTEM] Do NOT output plans as numbered text. Execute the required changes directly using tools (read_file, str_replace, etc).",
				})
				continue
			}

			// Phase 1: LSP 完了検証（1回限り - 同一エラーのループ防止）
			if !completionVerified {
				needsContinue, feedback := a.verifyCompletionWithDiagnostics(response)
				if needsContinue {
					completionVerified = true
					yellow.Fprintln(a.output(), "⚠️  Completion verification: LSP errors found in modified files")
					a.History = append(a.History, api.Message{
						Role:             "assistant",
						Content:          response,
						ReasoningContent: a.getLastReasoningContent(),
					})
					a.History = append(a.History, api.Message{
						Role:    "user",
						Content: feedback,
					})
					continue
				}
			}

			// Phase 2: Completion hooks（毎回実行 - 修正後の再チェック必要、max_retry で制限）
			if containsCompletionDeclaration(response) && len(cfg.Hooks.OnCompletion) > 0 {
				changedFiles := a.getTaskChangedFiles()
				if len(changedFiles) > 0 {
					// Built-in: git diff empty check
					hookNeedsContinue, hookFeedback := a.checkGitDiffEmpty()
					// User hooks (only if built-in checks passed)
					if !hookNeedsContinue {
						hookNeedsContinue, hookFeedback = a.runCompletionHooks(changedFiles)
					}
					if hookNeedsContinue {
						hookRetryCount++
						maxRetry := cfg.Hooks.MaxRetry
						if maxRetry <= 0 {
							maxRetry = 3
						}
						if hookRetryCount >= maxRetry {
							yellow.Fprintf(a.output(), "⚠️  Hook retry limit reached (%d/%d). Proceeding with completion.\n", hookRetryCount, maxRetry)
						} else {
							yellow.Fprintf(a.output(), "⚠️  Completion hook failed (%d/%d). Asking AI to fix...\n", hookRetryCount, maxRetry)
							a.History = append(a.History, api.Message{
								Role:             "assistant",
								Content:          response,
								ReasoningContent: a.getLastReasoningContent(),
							})
							a.History = append(a.History, api.Message{
								Role:    "user",
								Content: hookFeedback,
							})
							continue
						}
					}
				}
			}

			a.handleNormalResponse(response)
			a.showTaskSummary()
			return nil
		}

		// ツールを実行し、失敗を検出
		var lastFailedResult string

		// Phase 1: create_plan を先に処理（テキストベースで独立した履歴管理）
		for _, toolCall := range toolCalls {
			if toolCall.Tool != "create_plan" {
				continue
			}
			if a.Stats != nil {
				a.Stats.AddToolExecution(toolCall.Tool)
			}
			result, _ := a.executeToolWithSpinner(ctx, toolCall)
			a.History = append(a.History, api.Message{
				Role:             "assistant",
				Content:          response,
				ReasoningContent: a.getLastReasoningContent(),
			})
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: fmt.Sprintf("[Tool Result for create_plan]\n%s", result),
			})

			yellow.Fprintln(a.output(), "⚠️  create_plan is deprecated, continuing in normal mode...")
		}

		// Phase 2: 実行対象のツール呼び出しをフィルタ（create_plan を除外）
		var execToolCalls []*tools.ToolCall
		for _, tc := range toolCalls {
			if tc.Tool == "create_plan" {
				continue
			}
			execToolCalls = append(execToolCalls, tc)
		}

		// Phase 3: 全ツール呼び出しを1つの assistant メッセージにまとめて履歴追加
		if len(execToolCalls) > 0 {
			a.addToolCallsToHistory(response, execToolCalls)
		}

		// Phase 4: ツールを実行し結果を追加（parallel-safe なツールは並列実行）
		// loopDetectFn: 履歴は変更しない（executor の Phase 2 でメッセージ追加する）
		loopDetectFn := func(tc *tools.ToolCall) bool {
			cfg := a.cfg()
			threshold := cfg.LoopDetection.Threshold
			if isSameToolCall(tc, lastToolCall) {
				sameCallCount++
				if sameCallCount >= threshold {
					yellow.Fprintf(a.output(), "⚠️  Warning: Same tool call repeated %d times, stopping to prevent infinite loop\n", sameCallCount)
					yellow.Fprintf(a.output(), "   Tool: %s\n", tc.Tool)
					return true
				}
			} else {
				sameCallCount = 1
			}
			lastToolCall = tc
			return false
		}

		toolLoopDetected := a.executeToolCallsWithParallel(ctx, execToolCalls,
			loopDetectFn,
			nil, // Normal Mode ではスキップ対象なし
			// 各ツール結果の処理
			func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
				a.noteProjectMapMutation(tc, change)

				// str_replace エラー処理
				a.handleStrReplaceErrors(tc, result)

				// comment 継続フロー処理
				a.handleCommentFlow(tc, result)

				// 変更履歴を保存
				a.handleFileChange(change)

				// 結果を履歴に追加（重複チェック → 入口圧縮）
				historyContent := a.compactToolResult(tc, result)
				if tc.ID != "" {
					toolMsg := api.Message{
						Role:       "tool",
						Content:    historyContent,
						ToolCallID: tc.ID,
						ToolName:   tc.Tool,
					}
					a.History = append(a.History, toolMsg)
					if a.session != nil {
						a.session.AddMessageFromAPI(toolMsg, a.CurrentModel)
					}
				} else {
					a.History = append(a.History, api.Message{
						Role:    "user",
						Content: fmt.Sprintf("[Tool Result for %s]\n%s", tc.Tool, historyContent),
					})
				}
				_, _ = fmt.Fprintln(a.output())

				// str_replace 成功時: LSP診断遅延バッファにファイルを追加
				if tc.Tool == "str_replace" && !strings.HasPrefix(result, "Error:") &&
					!strings.HasPrefix(result, "[CANCELLED]") && !strings.HasPrefix(result, "[COMMENT]") {
					if path := tc.Args["path"]; path != "" {
						a.addPendingLSPFile(path)
					}
				}

				// 失敗パターンをチェック
				if tc.Tool == "bash" || tools.IsWriteTool(tc.Tool) {
					if failed, _ := plan.ContainsFailure(result); failed {
						lastFailedResult = result
					}
				}
			},
		)
		if toolLoopDetected {
			return fmt.Errorf("tool loop detected")
		}

		// LSP診断遅延フラッシュ: 全ツール実行後に改めて診断を実行し結果を追記。
		// str_replace の直後ではなくここで実行することで、連続編集途中の
		// 「import not used」等の一時エラーによる誤 auto-retry を防ぐ。
		if diagMsg := a.flushLSPDiagnostics(); diagMsg != "" && len(a.History) > 0 {
			a.History[len(a.History)-1].Content += diagMsg
		}

		// 失敗検出時の自動リトライ処理
		if lastFailedResult != "" {
			if autoRetryMax > 0 && retryCount < autoRetryMax {
				retryCount++
				a.ui().ResetTerminalState()
				red.Fprintf(a.output(), "❌ Failed (retry %d/%d)\n", retryCount, autoRetryMax)
				yellow.Fprintf(a.output(), "🔄 Retrying...\n")

				// リトライ用プロンプトを追加
				a.History = append(a.History, api.Message{
					Role: "user",
					Content: fmt.Sprintf(`The previous tool execution FAILED with the following error:

%s

Please:
1. Analyze the error carefully
2. Identify the root cause
3. Try a different approach to fix this

Do NOT give up. Try again with a different approach.`, lastFailedResult),
				})
				continue
			}

			// 自動リトライが exhausted
			if autoRetryMax > 0 {
				a.ui().ResetTerminalState()
				red.Fprintf(a.output(), "❌ Failed (%d retries exhausted)\n", autoRetryMax)
				yellow.Fprintln(a.output(), "Could not complete the task automatically. Letting AI respond...")
			}
			// AI に任せて続行（リトライカウンターをリセット）
			retryCount = 0
		} else if retryCount > 0 {
			// 成功した場合（リトライ後）
			green.Fprintf(a.output(), "✅ Succeeded (on retry %d)\n", retryCount)
			retryCount = 0
		}
	}

	if reachedHardLimit {
		yellow.Fprintf(a.output(), "⚠️  Tool loop limit reached (%d iterations)\n", hardLimit)
	}
	a.showTaskSummary()
	return nil
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
		action := ui.InferAction(change.Tool)
		ts.AddChange(change.FilePath, action, change.LinesAdded, change.LinesRemoved)
	}

	_, _ = fmt.Fprint(a.output(), ts.Render())
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

	inDiff := a.Stats.InputTokens - startStats.InputTokens
	outDiff := a.Stats.OutputTokens - startStats.OutputTokens
	costDiff := a.Stats.EstimatedCost() - startStats.EstimatedCost()
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
