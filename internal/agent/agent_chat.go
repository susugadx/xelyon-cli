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
	promptnormal "github.com/susugadx/xelyon-cli/internal/prompt/normal"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// chat はAIと対話する
// PlanModeEnabled に応じて Plan Mode または通常モードで処理
func (a *Agent) chat(input string) {
	a.chatInternal(input, nil)
}

// chatInternal はAIと対話する内部実装
// image が nil でない場合は画像付きメッセージとして処理
func (a *Agent) chatInternal(input string, image *api.ImageData) {
	a.SetStatus(StateRunning, "Processing request", "処理中", "Wait for response", "応答を待ってください")

	// タスク開始: changeStack のオフセットを記録（タスク単位のサマリー表示用）
	prevChanges := len(a.changeStack) - a.taskChangeOffset
	a.taskChangeOffset = len(a.changeStack)

	// completion hook の git diff 空チェック判定用ベースハッシュを記録
	a.taskBaseCommitHash = ""
	if out, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		a.taskBaseCommitHash = strings.TrimSpace(string(out))
	}

	if prevChanges > 0 {
		yellow.Printf("⚠️  %d uncommitted changes from previous task\n", prevChanges)
		fmt.Println("💡 Run /commit or git commit to keep changes separate")
		fmt.Println()
	}

	// GitHub MCP ヒントを追加（GitHub関連リクエストの場合）
	input = a.AddGitHubHint(input)

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
	cfg := config.GetGlobalConfig()
	timeout := time.Duration(cfg.APIRetry.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	a.cancelFunc = cancel

	// トークン使用量の警告チェック（API呼び出し前）
	a.checkTokenWarning()

	var err error
	if a.PlanModeEnabled {
		// Plan Mode: 調査 → 計画 → 承認 → 実行
		err = a.RunPlanMode(ctx, input)
	} else {
		// 通常モード: ツール実行ループ
		err = a.runNormalMode(ctx, input, image)
	}

	if err != nil {
		if errors.Is(err, context.Canceled) {
			yellow.Println("\n⚠️  Response interrupted")
		} else {
			// トークン上限エラーの場合は自動圧縮+リトライを試みる
			if token.IsTokenLimitError(err) {
				// リトライ関数を定義
				retryFunc := func() error {
					// 同じコンテキストで再実行
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
					ui.StopGlobalSpinner()
					a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
					return
				}
				// リトライ失敗時はエラーメッセージを表示（handleTokenLimitErrorWithRetry内で既に表示済み）
				red.Printf("Error: %v\n", err)
			} else {
				// トークン上限以外のエラー
				red.Printf("Error: %v\n", err)
			}
		}
		ui.StopGlobalSpinner()
		a.SetStatus(StateAborted, "Request failed", "リクエスト失敗", "Try again", "再試行してください")
		return
	}

	// リクエスト完了時の usage 表示
	a.printTaskUsage(startStats)

	// 自動圧縮チェック（成功時）
	a.maybeAutoCompress()

	a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
}

// runNormalMode は通常モードでの処理（Plan Mode OFF 時）
// ツールを個別に確認しながら実行するループ（自動リトライ対応）
func (a *Agent) runNormalMode(ctx context.Context, input string, image *api.ImageData) error {
	// 通常モード用の指示を追加（prompt パッケージから取得）
	normalModeInput := input + promptnormal.NormalModePrompt

	// 履歴に追加
	a.History = append(a.History, api.Message{Role: "user", Content: normalModeInput})

	cfg := config.GetGlobalConfig()
	maxIterations := cfg.General.ToolLoopLimit
	var lastToolCall *tools.ToolCall
	var sameCallCount int

	// 自動リトライ設定
	autoRetryMax := cfg.PlanMode.MaxRetry
	retryCount := 0

	var completionVerified bool // 完了検証ガード（タスク内1回限り）
	var hookRetryCount int      // フック失敗リトライカウンター

	// テキスト計画 → create_plan リダイレクト
	var textPlanRedirectCount int
	const maxTextPlanRedirects = 2

	for i := 0; i < maxIterations; i++ {
		// API呼び出し
		var response string
		var err error
		if i == 0 && image != nil {
			inputWithPrompt := input + promptnormal.NormalModePrompt
			response, err = a.CurrentProvider.ChatWithImage(
				ctx, a.SystemPrompt, a.History[:len(a.History)-1], inputWithPrompt, image, a.CurrentModel,
			)
		} else {
			response, err = a.CurrentProvider.ChatWithTools(
				ctx,
				a.SystemPrompt,
				a.History,
				a.CurrentModel,
			)
			// tool_choice が設定されていた場合は解除
			if tc, ok := a.CurrentProvider.(interface{ ClearToolChoice() }); ok {
				tc.ClearToolChoice()
			}
		}
		if err != nil {
			ui.StopGlobalSpinner()
			return fmt.Errorf("API call failed: %w", err)
		}

		// ツール呼び出しをパース（Plan JSON チェックより先に実行 — create_plan の FC が誤検出されるのを防止）
		toolCalls := tools.ParseToolCalls(response)

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
					yellow.Printf("📋 FC fallback: extracted %d-step plan from text. Switching to step-by-step...\n", len(p.Steps))
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
			// パース失敗 → 従来どおりリダイレクト
			yellow.Println("⚠️  Plan JSON detected but parse failed. Use create_plan tool instead.")
			a.History = append(a.History, api.Message{
				Role:             "assistant",
				Content:          response,
				ReasoningContent: a.getLastReasoningContent(),
			})
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: "[SYSTEM] You are in NORMAL MODE. Do NOT output JSON directly. Use create_plan tool or execute tools DIRECTLY.",
			})
			continue
		}

		// デバッグログ
		if os.Getenv("XELYON_DEBUG_TOOLS") == "1" {
			fmt.Printf("[DEBUG Tools] Response length: %d, ToolCalls found: %d\n", len(response), len(toolCalls))
			if len(response) < config.DebugPreviewLen {
				fmt.Printf("[DEBUG Tools] Response: %s\n", response)
			} else {
				fmt.Printf("[DEBUG Tools] Response (first %d): %s...\n", config.DebugPreviewLen, response[:config.DebugPreviewLen])
			}
			for i, tc := range toolCalls {
				fmt.Printf("[DEBUG Tools] ToolCall[%d]: tool=%s, args=%v\n", i, tc.Tool, tc.Args)
			}
		}

		// ツール呼び出しなし = 通常の回答
		if len(toolCalls) == 0 {
			// テキスト計画の検出 → create_plan ツール使用を要求
			// ただし完了宣言を含むレスポンスはスキップ（まとめの番号リストを誤検知しない）
			steps := extractTextPlan(response)
			if !containsCompletionDeclaration(response) && len(steps) >= 3 && isActionPlan(steps) {
				textPlanRedirectCount++
				if textPlanRedirectCount > maxTextPlanRedirects {
					yellow.Printf("⚠️  AI failed to use create_plan after %d attempts. Execute tools directly.\n", maxTextPlanRedirects)
					a.History = append(a.History, api.Message{
						Role:             "assistant",
						Content:          response,
						ReasoningContent: a.getLastReasoningContent(),
					})
					a.History = append(a.History, api.Message{
						Role:    "user",
						Content: "[SYSTEM] create_plan is not available. Execute the required changes directly using tools (str_replace, bash, etc).",
					})
					continue
				}

				// 2回目: tool_choice で FC 強制
				if textPlanRedirectCount >= 2 {
					yellow.Printf("⚠️  Text plan detected again. Forcing create_plan via tool_choice... (%d/%d)\n",
						textPlanRedirectCount, maxTextPlanRedirects)
					// tool_choice を一時的に設定
					if tc, ok := a.CurrentProvider.(interface{ SetToolChoice(name string) }); ok {
						tc.SetToolChoice("create_plan")
					}
				}

				yellow.Printf("⚠️  Text plan detected (%d steps). Redirecting to create_plan tool... (%d/%d)\n",
					len(steps), textPlanRedirectCount, maxTextPlanRedirects)
				a.History = append(a.History, api.Message{
					Role:             "assistant",
					Content:          response,
					ReasoningContent: a.getLastReasoningContent(),
				})
				a.History = append(a.History, api.Message{
					Role: "user",
					Content: "[SYSTEM] You output a text plan instead of using the create_plan tool. " +
						"Text plans cannot be tracked or verified. " +
						"Use the create_plan tool with proper steps and tools fields to create a structured plan. " +
						"Do NOT output plans as numbered text.",
				})
				continue
			}

			// Phase 1: LSP 完了検証（1回限り - 同一エラーのループ防止）
			if !completionVerified {
				needsContinue, feedback := a.verifyCompletionWithDiagnostics(response)
				if needsContinue {
					completionVerified = true
					yellow.Println("⚠️  Completion verification: LSP errors found in modified files")
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
							yellow.Printf("⚠️  Hook retry limit reached (%d/%d). Proceeding with completion.\n", hookRetryCount, maxRetry)
						} else {
							yellow.Printf("⚠️  Completion hook failed (%d/%d). Asking AI to fix...\n", hookRetryCount, maxRetry)
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
		skippedWrites := 0
		skippedCommands := 0

		// Phase 1: create_plan を先に処理（テキストベースで独立した履歴管理）
		for _, toolCall := range toolCalls {
			if toolCall.Tool != "create_plan" {
				continue
			}
			if a.Stats != nil {
				a.Stats.AddToolExecution(toolCall.Tool)
			}
			result, _ := tools.Execute(toolCall)
			a.History = append(a.History, api.Message{
				Role:             "assistant",
				Content:          response,
				ReasoningContent: a.getLastReasoningContent(),
			})
			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: fmt.Sprintf("[Tool Result for create_plan]\n%s", result),
			})

			createPlanTool := a.getCreatePlanTool()
			if createPlanTool != nil {
				if p := createPlanTool.LastPlan(); p != nil {
					green.Printf("📋 Plan created in normal mode (%d steps). Switching to step-by-step execution...\n", len(p.Steps))
					if err := a.runImplementationPhase(ctx, p); err != nil {
						return err
					}
					a.runCompletionHooksWithRetry(ctx)
					a.showTaskSummary()
					return nil
				}
			}
			yellow.Println("⚠️  create_plan failed, continuing in normal mode...")
		}

		// Phase 2: 実行対象のツール呼び出しをフィルタ（create_plan, write throttle, bash skip を除外）
		var execToolCalls []*tools.ToolCall
		writeQueued := false
		for _, tc := range toolCalls {
			if tc.Tool == "create_plan" {
				continue
			}
			if a.shouldThrottleWrite(tc) && writeQueued {
				skippedWrites++
				continue
			}
			if skippedWrites > 0 && tc.Tool == "bash" {
				skippedCommands++
				continue
			}
			execToolCalls = append(execToolCalls, tc)
			if a.shouldThrottleWrite(tc) {
				writeQueued = true
			}
		}

		// Phase 3: 全ツール呼び出しを1つの assistant メッセージにまとめて履歴追加
		if len(execToolCalls) > 0 {
			a.addToolCallsToHistory(response, execToolCalls)
		}

		// Phase 4: ツールを実行し結果を追加
		for idx, tc := range execToolCalls {
			// ループ検知（assistant メッセージは追加済みなので response="" で呼ぶ）
			if a.shouldAbortToolLoopWithResponse("", tc, lastToolCall, &sameCallCount) {
				// shouldAbortToolLoopWithResponse が現在の TC の tool result を追加済み。
				// 残りの未実行 TC にダミー結果を追加（API が tool result 欠落でエラーにならないようにする）。
				for _, remaining := range execToolCalls[idx+1:] {
					if remaining.ID != "" {
						a.History = append(a.History, api.Message{
							Role:       "tool",
							Content:    "[SYSTEM] Skipped due to tool loop detection.",
							ToolCallID: remaining.ID,
							ToolName:   remaining.Tool,
						})
					}
				}
				return fmt.Errorf("tool loop detected")
			}
			lastToolCall = tc

			result := a.executeToolOnly(tc)

			// 書き込み成功後の自動 read-back（ツール結果に追記）
			if a.shouldThrottleWrite(tc) {
				if !strings.HasPrefix(result, "Error:") {
					a.autoReadBack(tc)
				}
			}

			// str_replace 成功時: LSP診断遅延バッファにファイルを追加
			// 連続 str_replace 途中の一時的エラーによる誤 auto-retry を防ぐため、
			// 診断は全ツール実行後にまとめて行う（flushLSPDiagnostics）。
			if tc.Tool == "str_replace" && !strings.HasPrefix(result, "Error:") &&
				!strings.HasPrefix(result, "[CANCELLED]") && !strings.HasPrefix(result, "[COMMENT]") {
				if path := tc.Args["path"]; path != "" {
					a.addPendingLSPFile(path)
				}
			}

			// 書き込みツール成功後にインデックス更新をトリガー
			if tools.IsWriteTool(tc.Tool) && !strings.HasPrefix(result, "Error:") {
				a.triggerIndexUpdate()
			}

			// 失敗パターンをチェック
			// bash 等のコマンド実行系と、ファイル変更系（str_replace, write_file等）のみチェック
			if tc.Tool == "bash" || tools.IsWriteTool(tc.Tool) {
				if failed, _ := plan.ContainsFailure(result); failed {
					lastFailedResult = result
				}
			}
		}

		// スキップされた書き込みとコマンドがあれば通知（最後のツール結果に追記）
		if skippedWrites > 0 || skippedCommands > 0 {
			a.injectWriteThrottleMessage(skippedWrites, skippedCommands)
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
				fmt.Print("\033[?25h") // カーソルを表示（スピナー停止）
				red.Printf("❌ Failed (retry %d/%d)\n", retryCount, autoRetryMax)
				yellow.Printf("🔄 Retrying...\n")

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
				fmt.Print("\033[?25h") // カーソルを表示（スピナー停止）
				red.Printf("❌ Failed (%d retries exhausted)\n", autoRetryMax)
				yellow.Println("Could not complete the task automatically. Letting AI respond...")
			}
			// AI に任せて続行（リトライカウンターをリセット）
			retryCount = 0
		} else if retryCount > 0 {
			// 成功した場合（リトライ後）
			green.Printf("✅ Succeeded (on retry %d)\n", retryCount)
			retryCount = 0
		}
	}

	yellow.Printf("⚠️  Tool loop limit reached (%d iterations)\n", maxIterations)
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

	fmt.Print(ts.Render())
}

// chatWithImage は画像付きメッセージでAIと対話する
func (a *Agent) chatWithImage(input string, image *api.ImageData) {
	// プロバイダーが画像対応かチェック
	if !a.CurrentProvider.SupportsImages() {
		yellow.Printf("Warning: %s does not support images. The image will be ignored.\n", a.CurrentProvider.Name())
		a.chat(input)
		return
	}

	// 画像情報をログ
	green.Printf("🖼️  Sending image: %s (%s)\n", image.Path, api.FormatImageSize(image.Size))

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
	green.Print("✓ ")
	if strings.ToLower(a.ProviderName) == "ollama" {
		// Ollama の場合はコスト非表示
		fmt.Printf("In: %s + Out: %s = %s tok\n",
			FormatNumber(inDiff),
			FormatNumber(outDiff),
			FormatNumber(total))
	} else {
		fmt.Printf("In: %s + Out: %s = %s tok ",
			FormatNumber(inDiff),
			FormatNumber(outDiff),
			FormatNumber(total))
		dim.Printf("(~$%.4f)\n", costDiff)
	}
}
