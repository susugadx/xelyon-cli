package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
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
	a.SetStatus(StateRunning, "Processing request", "処理中", "Wait for response", "応答を待ってください")

	// タスク開始: changeStack のオフセットを記録（タスク単位のサマリー表示用）
	prevChanges := len(a.changeStack) - a.taskChangeOffset
	a.taskChangeOffset = len(a.changeStack)
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
	if a.Stats != nil {
		a.Stats.UserMessages++
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
		err = a.runNormalMode(ctx, input)
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
						return a.runNormalMode(ctx, input)
					}
				}

				// 自動圧縮+リトライを試みる
				if a.handleTokenLimitErrorWithRetry(err, retryFunc, a.PlanModeEnabled, nil) {
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
	a.printLastUsage()

	// 自動圧縮チェック（成功時）
	a.maybeAutoCompress()

	a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
}

// runNormalMode は通常モードでの処理（Plan Mode OFF 時）
// ツールを個別に確認しながら実行するループ（自動リトライ対応）
func (a *Agent) runNormalMode(ctx context.Context, input string) error {
	// 通常モード用の指示を追加（prompt パッケージから取得）
	normalModeInput := input + promptnormal.NormalModePrompt

	// 履歴に追加
	a.History = append(a.History, api.Message{Role: "user", Content: normalModeInput})

	cfg := config.GetGlobalConfig()
	maxIterations := cfg.General.ToolLoopLimit
	var lastToolCall *tools.ToolCall
	var sameCallCount int

	// 自動リトライ設定
	autoRetryMax := cfg.PlanMode.AutoRetry
	retryCount := 0

	var completionVerified bool // 完了検証ガード（タスク内1回限り）
	var hookRetryCount int      // フック失敗リトライカウンター

	// Step Tracking: テキスト計画の未完了検知
	var pendingSteps int     // テキスト計画のステップ数
	var completedActions int // 実行済みアクション（write系 + bash）の数
	var forceContCount int   // 強制続行の回数
	const maxForceCont = 3   // 強制続行の上限

	for i := 0; i < maxIterations; i++ {
		// API呼び出し
		response, err := a.CurrentProvider.ChatWithTools(
			ctx,
			a.SystemPrompt,
			a.History,
			a.CurrentModel,
		)
		if err != nil {
			ui.StopGlobalSpinner()
			return fmt.Errorf("API call failed: %w", err)
		}

		// Plan JSON が検出された場合、パースして step-by-step 実行を試みる
		if plan.ContainsPlanJSON(response) {
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

		// Step Tracking: テキスト計画を記録（最初の検出のみ）
		if pendingSteps == 0 {
			if steps := extractTextPlan(response); len(steps) >= 2 && isActionPlan(steps) {
				pendingSteps = len(steps)
				completedActions = 0
				forceContCount = 0
			}
		}

		// ツール呼び出しをパース
		toolCalls := tools.ParseToolCalls(response)

		// FC rescue: テキストから抽出された toolCall にダミー ID を注入
		// これにより下流の処理が FC 成功時と同じパス（role:"tool"）を通る
		for i, tc := range toolCalls {
			if tc.ID == "" {
				toolCalls[i].ID = fmt.Sprintf("call_rescue_%03d", i+1)
			}
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
			// Step Tracking: 計画のステップが未完了なら強制続行
			if pendingSteps > 0 && completedActions < pendingSteps {
				forceContCount++
				if forceContCount <= maxForceCont {
					yellow.Printf("⚠️  Step tracking: %d/%d steps completed. Forcing continuation... (%d/%d)\n",
						completedActions, pendingSteps, forceContCount, maxForceCont)
					a.History = append(a.History, api.Message{
						Role:             "assistant",
						Content:          response,
						ReasoningContent: a.getLastReasoningContent(),
					})
					a.History = append(a.History, api.Message{
						Role:    "user",
						Content: fmt.Sprintf("[SYSTEM] You declared %d steps but only completed %d. Do NOT summarize. Continue with the remaining steps immediately.", pendingSteps, completedActions),
					})
					continue
				}
				yellow.Printf("⚠️  Step tracking: %d/%d steps completed but AI not progressing. Giving up.\n",
					completedActions, pendingSteps)
				pendingSteps = 0
			}

			// テキスト計画の自動検出（3ステップ以上の作業計画）
			if steps := extractTextPlan(response); len(steps) >= 3 && isActionPlan(steps) {
				yellow.Printf("📋 Auto-detected %d-step plan in text. Switching to step-by-step execution...\n", len(steps))
				p := buildAutoPlan(steps)
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
					hookNeedsContinue, hookFeedback := checkGitDiffEmpty()
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
		writeExecuted := false
		skippedWrites := 0
		skippedCommands := 0

		for _, toolCall := range toolCalls {
			// Normal Mode で create_plan が FC で呼ばれたら step-by-step 実行に切り替え
			if toolCall.Tool == "create_plan" {
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
				continue
			}

			// Write Throttle: 書き込み系ツールは1ターン1回まで
			if a.shouldThrottleWrite(toolCall) && writeExecuted {
				skippedWrites++
				continue
			}

			// 書き込みがスキップされた後は bash もスキップ
			if skippedWrites > 0 && toolCall.Tool == "bash" {
				skippedCommands++
				continue
			}

			// ループ検知
			if a.shouldAbortToolLoopWithResponse(response, toolCall, lastToolCall, &sameCallCount) {
				return fmt.Errorf("tool loop detected")
			}
			lastToolCall = toolCall

			// ツール実行（executeToolCallWithResult で結果も取得）
			result := a.executeToolCallWithResult(response, toolCall)

			// Step Tracking: アクション実行をカウント（write系 + bash）
			if tools.IsWriteTool(toolCall.Tool) || toolCall.Tool == "bash" {
				completedActions++
			}

			// 書き込み成功後の自動 read-back（ツール結果に追記）
			if a.shouldThrottleWrite(toolCall) {
				writeExecuted = true
				if !strings.HasPrefix(result, "Error:") {
					a.autoReadBack(toolCall)
				}
			}

			// 失敗パターンをチェック
			if failed, _ := plan.ContainsFailure(result); failed {
				lastFailedResult = result
			}
		}

		// スキップされた書き込みとコマンドがあれば通知（最後のツール結果に追記）
		if skippedWrites > 0 || skippedCommands > 0 {
			a.injectWriteThrottleMessage(skippedWrites, skippedCommands)
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

	// 通常モード用の指示を追加（画像分析時もツールを使えるようにする）
	inputWithPrompt := input + promptnormal.NormalModePrompt

	// 画像情報をログ
	green.Printf("🖼️  Sending image: %s (%s)\n", image.Path, api.FormatImageSize(image.Size))

	// 履歴に追加（テキストのみ - 画像はセッションに保存しない）
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: inputWithPrompt,
	})

	// セッションに保存（テキストのみ）
	if a.session != nil {
		a.session.AddMessage("user", input, a.CurrentModel)
	}

	// 統計情報更新
	if a.Stats != nil {
		a.Stats.UserMessages++
	}

	// API呼び出し（設定からタイムアウト取得）
	cfg := config.GetGlobalConfig()
	timeout := time.Duration(cfg.APIRetry.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	a.cancelFunc = cancel

	maxIterations := cfg.General.ToolLoopLimit
	var lastToolCall *tools.ToolCall
	var sameCallCount int

	// 自動リトリー設定
	autoRetryMax := cfg.PlanMode.AutoRetry
	retryCount := 0

	var completionVerified bool // 完了検証ガード（タスク内1回限り）
	var hookRetryCount int      // フック失敗リトライカウンター

	for i := 0; i < maxIterations; i++ {
		var response string
		var err error

		if i == 0 {
			// 初回のみ画像付きで呼び出し
			response, err = a.CurrentProvider.ChatWithImage(ctx, a.SystemPrompt, a.History[:len(a.History)-1], inputWithPrompt, image, a.CurrentModel)
		} else {
			// 2回目以降（ツール実行結果の送信など）は通常のツール呼び出し
			response, err = a.CurrentProvider.ChatWithTools(ctx, a.SystemPrompt, a.History, a.CurrentModel)
		}

		if err != nil {
			ui.StopGlobalSpinner()
			red.Printf("Error: %v\n", err)
			return
		}

		// Plan JSON が検出された場合、パースして step-by-step 実行を試みる
		if plan.ContainsPlanJSON(response) {
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
						red.Printf("Error: %v\n", err)
						return
					}
					a.runCompletionHooksWithRetry(ctx)
					a.showTaskSummary()
					return
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

		// ツール呼び出しをパース
		toolCalls := tools.ParseToolCalls(response)

		// FC rescue: テキストから抽出された toolCall にダミー ID を注入
		for i, tc := range toolCalls {
			if tc.ID == "" {
				toolCalls[i].ID = fmt.Sprintf("call_rescue_%03d", i+1)
			}
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
			// テキスト計画の自動検出（3ステップ以上の作業計画）
			if steps := extractTextPlan(response); len(steps) >= 3 && isActionPlan(steps) {
				yellow.Printf("📋 Auto-detected %d-step plan in text. Switching to step-by-step execution...\n", len(steps))
				p := buildAutoPlan(steps)
				a.History = append(a.History, api.Message{
					Role:             "assistant",
					Content:          response,
					ReasoningContent: a.getLastReasoningContent(),
				})
				if err := a.runImplementationPhase(ctx, p); err != nil {
					red.Printf("Error: %v\n", err)
					return
				}
				a.runCompletionHooksWithRetry(ctx)
				a.showTaskSummary()
				return
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
					hookNeedsContinue, hookFeedback := checkGitDiffEmpty()
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
			return
		}

		// ツールを実行し、失敗を検出
		var lastFailedResult string
		writeExecuted := false
		skippedWrites := 0
		skippedCommands := 0

		for _, toolCall := range toolCalls {
			// Normal Mode で create_plan が FC で呼ばれたら step-by-step 実行に切り替え
			if toolCall.Tool == "create_plan" {
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
							red.Printf("Error: %v\n", err)
							return
						}
						a.runCompletionHooksWithRetry(ctx)
						a.showTaskSummary()
						return
					}
				}
				yellow.Println("⚠️  create_plan failed, continuing in normal mode...")
				continue
			}

			// Write Throttle: 書き込み系ツールは1ターン1回まで
			if a.shouldThrottleWrite(toolCall) && writeExecuted {
				skippedWrites++
				continue
			}

			// 書き込みがスキップされた後は bash もスキップ
			if skippedWrites > 0 && toolCall.Tool == "bash" {
				skippedCommands++
				continue
			}

			// ループ検知
			if a.shouldAbortToolLoopWithResponse(response, toolCall, lastToolCall, &sameCallCount) {
				red.Println("Error: tool loop detected")
				return
			}
			lastToolCall = toolCall

			// ツール実行（executeToolCallWithResult で結果も取得）
			result := a.executeToolCallWithResult(response, toolCall)

			// 書き込み成功後の自動 read-back（ツール結果に追記）
			if a.shouldThrottleWrite(toolCall) {
				writeExecuted = true
				if !strings.HasPrefix(result, "Error:") {
					a.autoReadBack(toolCall)
				}
			}

			// 失敗パターンをチェック
			if failed, _ := plan.ContainsFailure(result); failed {
				lastFailedResult = result
			}
		}

		// スキップされた書き込みとコマンドがあれば通知（最後のツール結果に追記）
		if skippedWrites > 0 || skippedCommands > 0 {
			a.injectWriteThrottleMessage(skippedWrites, skippedCommands)
		}

		// 失敗検出時の自動リトリー処理
		if lastFailedResult != "" {
			if autoRetryMax > 0 && retryCount < autoRetryMax {
				retryCount++
				fmt.Print("\033[?25h") // カーソルを表示（スピナー停止）
				red.Printf("❌ Failed (retry %d/%d)\n", retryCount, autoRetryMax)
				yellow.Printf("🔄 Retrying...\n")

				// リトリー用プロンプトを追加
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

			// 自動リトリーが exhausted
			if autoRetryMax > 0 {
				fmt.Print("\033[?25h") // カーソルを表示（スピナー停止）
				red.Printf("❌ Failed (%d retries exhausted)\n", autoRetryMax)
				yellow.Println("Could not complete the task automatically. Letting AI respond...")
			}
			// AI に任せて続行（リトリーカウンターをリセット）
			retryCount = 0
		} else if retryCount > 0 {
			// 成功した場合（リトリー後）
			green.Printf("✅ Succeeded (on retry %d)\n", retryCount)
			retryCount = 0
		}
	}

	yellow.Printf("⚠️  Tool loop limit reached (%d iterations)\n", maxIterations)
	a.showTaskSummary()
}

// printLastUsage はリクエスト完了時の usage を表示
func (a *Agent) printLastUsage() {
	a.statsMu.Lock()
	usage := a.Stats.LastUsage
	a.Stats.LastUsage = nil // 表示後クリア
	a.statsMu.Unlock()

	if usage == nil {
		return
	}

	total := usage.InputTokens + usage.OutputTokens
	cost := CalculateRequestCost(a.ProviderName, a.CurrentModel, usage.InputTokens, usage.OutputTokens)

	// ✓ を緑色で表示、残りはdimまたは通常色
	green.Print("✓ ")
	if strings.ToLower(a.ProviderName) == "ollama" {
		// Ollama の場合はコスト非表示
		fmt.Printf("In: %s + Out: %s = %s tok\n",
			FormatNumber(usage.InputTokens),
			FormatNumber(usage.OutputTokens),
			FormatNumber(total))
	} else {
		fmt.Printf("In: %s + Out: %s = %s tok ",
			FormatNumber(usage.InputTokens),
			FormatNumber(usage.OutputTokens),
			FormatNumber(total))
		dim.Printf("(~$%.4f)\n", cost)
	}
}
