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
	"github.com/susugadx/xelyon-cli/internal/skills"
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
	// スキル検出・ロード
	a.loadDetectedSkills(input)

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

	maxIterations := config.MaxToolIterations
	var lastToolCall *tools.ToolCall
	var sameCallCount int

	// 自動リトライ設定
	cfg := config.GetGlobalConfig()
	autoRetryMax := cfg.PlanMode.AutoRetry
	retryCount := 0

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

		// Plan JSON が検出された場合、ツール使用を促す
		if plan.ContainsPlanJSON(response) {
			yellow.Println("⚠️  Plan JSON detected in normal mode. Use create_plan tool instead.")
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
			a.handleNormalResponse(response)
			a.showTaskSummary()
			return nil
		}

		// ツールを実行し、失敗を検出
		var lastFailedResult string
		for _, toolCall := range toolCalls {
			// ループ検知
			if a.shouldAbortToolLoopWithResponse(response, toolCall, lastToolCall, &sameCallCount) {
				return fmt.Errorf("tool loop detected")
			}
			lastToolCall = toolCall

			// ツール実行（executeToolCallWithResult で結果も取得）
			result := a.executeToolCallWithResult(response, toolCall)

			// 失敗パターンをチェック
			if failed, _ := plan.ContainsFailure(result); failed {
				lastFailedResult = result
			}
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

	// 履歴に追加（テキストのみ - 画像はセッションに保存しない）
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: input,
	})

	// セッションに保存（テキストのみ）
	if a.session != nil {
		a.session.AddMessage("user", input, a.CurrentModel)
	}

	// 統計情報更新
	if a.Stats != nil {
		a.Stats.UserMessages++
	}

	// 画像付きAPI呼び出し（設定からタイムアウト取得）
	cfg := config.GetGlobalConfig()
	timeout := time.Duration(cfg.APIRetry.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	response, err := a.CurrentProvider.ChatWithImage(ctx, a.SystemPrompt, a.History[:len(a.History)-1], input, image, a.CurrentModel)
	if err != nil {
		red.Printf("エラー: %v\n", err)
		yellow.Println("API呼び出しに失敗しました。ネットワーク接続を確認してください。")
		return
	}

	// 通常の回答処理
	a.handleNormalResponse(response)
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
	cost := CalculateRequestCost(a.ProviderName, usage.InputTokens, usage.OutputTokens)

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

// loadDetectedSkills は入力からスキルを検出してシステムプロンプトに追加
func (a *Agent) loadDetectedSkills(input string) {
	detected := skills.DetectSkills(input)
	if len(detected) == 0 {
		return
	}

	var skillContents []string
	for _, name := range detected {
		content, err := skills.LoadSkill(name)
		if err == nil {
			skillContents = append(skillContents, content)
			green.Printf("📚 Skill loaded: %s\n", name)
		}
	}

	if len(skillContents) > 0 {
		a.SystemPrompt += "\n\n## Loaded Skills\n" + strings.Join(skillContents, "\n\n")
	}
}
