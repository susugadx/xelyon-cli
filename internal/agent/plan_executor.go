package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// runImplementationPhase は実装フェーズを実行（順次実行）
func (a *Agent) runImplementationPhase(ctx context.Context, p *plan.Plan) error {
	for {
		// 次のステップを取得
		nextID := p.GetNextStep()
		if nextID == -1 {
			break // 全て完了
		}
		step := p.GetStep(nextID)
		if step == nil {
			break
		}

		fmt.Printf("\n%s\n", ui.FormatStepProgress(nextID, len(p.Steps), step.Description, "running"))
		if err := a.executeStepV2(ctx, p, step, nextID-1, 0); err != nil {
			return err
		}
		p.UpdateStatus(nextID, "completed", "")

		// ステップ完了フックを実行
		if hooks := a.cfg().Hooks; len(hooks.OnStepComplete) > 0 {
			if !a.runStepCompleteHooksWithRetry(ctx, nextID, step.Description, "completed") {
				yellow.Printf("⚠️  Step %d hooks failed but proceeding to next step\n", nextID)
			}
		}

		// 全て完了したか確認
		if p.IsCompleted() {
			break
		}
	}

	green.Printf("\n✓ All %d steps completed!\n", len(p.Steps))

	// セッションを保存
	if a.storage != nil && a.session != nil {
		a.syncResponseIDToSession()
		if err := a.storage.Save(a.session); err != nil {
			yellow.Printf("Warning: Failed to save session: %v\n", err)
		}
	}

	// git diff empty check は executeStepV2 の Level 1/Level 2 ガードでカバー済み
	// runImplementationPhase レベルではチェックしない（調査系プランなど変更なしが正常なケースがある）

	return nil
}

// executeStepV2 は単一ステップを実行（失敗検知・リトライ対応）
func (a *Agent) executeStepV2(ctx context.Context, p *plan.Plan, step *plan.PlanStep, idx int, retryCount int) error {
	maxRetries := config.PlanMaxRetries

	if retryCount > 0 {
		ui.StopGlobalSpinner()
		yellow.Printf("🔄 Retry attempt %d/%d for step %d...\n", retryCount, maxRetries, step.ID)
	}

	// ステップ実行を指示
	stepPrompt := promptplan.BuildStepPrompt(step.ID, step.Description, step.Tools)

	// リトライ時は履歴に追加しない
	if retryCount == 0 {
		a.History = append(a.History, api.Message{Role: "user", Content: stepPrompt})
	}

	// ステップ内のツール実行ループ
	cfg := a.cfg()
	maxStepIterations := cfg.General.ToolLoopLimit
	maxContinues := config.PlanMaxAutoContinues
	continueCount := 0
	var lastFailedResult string
	var lastFailReason string

	// ステップ完了検証用トラッカー
	stepHadWrites := false             // 書き込み系ツールが実行されたか
	stepHadNoChangeNeeded := false     // 書き込み系ツールが「変更不要」と返したか
	beforeDiffHash := getGitDiffHash() // Level 2 用: ステップ開始時の diff ハッシュ
	var stepCompletionVerified bool    // LSP完了検証ガード（ステップ内1回限り）

	// ループ検知用トラッカー
	var lastToolCall *tools.ToolCall
	var sameCallCount int

	for j := 0; j < maxStepIterations; j++ {
		compactedHistory, metrics := CompactOldToolResults(a.History, DefaultKeepTurns, DefaultMaxLines, DefaultHeadLines, DefaultTailLines)
		a.addCompactionMetrics(metrics)
		response, err := a.CurrentProvider.ChatWithTools(
			a.requestContext(ctx),
			a.SystemPrompt,
			compactedHistory,
			a.CurrentModel,
		)
		if err != nil {
			ui.StopGlobalSpinner()
			return fmt.Errorf("step %d failed: %w", step.ID, err)
		}

		// ツール呼び出しチェック
		toolCalls := a.parseToolCalls(response)

		// FC rescue: テキストから抽出された toolCall にダミー ID を注入
		// これにより下流の処理が FC 成功時と同じパス（role:"tool"）を通る
		for i, tc := range toolCalls {
			if tc.ID == "" {
				toolCalls[i].ID = fmt.Sprintf("call_rescue_%03d", i+1)
			}
		}

		execToolCalls := toolCalls

		if len(execToolCalls) > 0 {
			// ツール呼び出しあり: addToolCallsToHistory で履歴に追加（セッション保存対応）
			a.addToolCallsToHistory(response, execToolCalls)
		} else {
			// ツール呼び出しなし（ステップ完了時の最終応答）: 通常の履歴追加 + セッション保存
			assistantMsg := api.Message{
				Role:             "assistant",
				Content:          response,
				ReasoningContent: a.getLastReasoningContent(),
			}
			a.History = append(a.History, assistantMsg)

			// セッションに保存
			if a.session != nil {
				a.session.AddMessageFromAPI(assistantMsg, a.CurrentModel)
			}

			if a.Stats != nil {
				a.Stats.AssistantMessages++
			}
		}

		if len(execToolCalls) == 0 {
			// AIが質問している場合、自動続行を試みる
			if isAIQuestionWithToolParser(response, a.parseToolCalls) && continueCount < maxContinues {
				continueCount++
				yellow.Printf("⚠️  AI asked a question, auto-continuing (%d/%d)...\n", continueCount, maxContinues)

				a.History = append(a.History, api.Message{
					Role:    "user",
					Content: "[AUTO-CONTINUE] Yes, proceed with the step. Execute the required tools directly without asking for confirmation.",
				})
				continue
			}

			// Level 0: ツール実行なし + 完了宣言 + 既に diff 変化あり → 別ステップで実施済みとして正常終了
			// 条件3（diff 変化）がないと AI がサボってもスキップされてしまうため必須
			if containsCompletionDeclaration(response) && beforeDiffHash != "" {
				afterDiffHash := getGitDiffHash()
				if afterDiffHash != "" && afterDiffHash != beforeDiffHash {
					green.Printf("✓ Step %d completed (already applied)\n", step.ID)
					return nil
				}
			}

			// Level 2: 書き込みツール実行済みだが diff 変化なし → 強制続行
			if stepHadWrites && !stepHadNoChangeNeeded {
				afterDiffHash := getGitDiffHash()
				if beforeDiffHash != "" && afterDiffHash != "" && beforeDiffHash == afterDiffHash {
					if continueCount < maxContinues {
						continueCount++
						yellow.Printf("⚠️  Step %d: write tools executed but no file changes detected (%d/%d)\n",
							step.ID, continueCount, maxContinues)
						a.History = append(a.History, api.Message{
							Role: "user",
							Content: fmt.Sprintf("[SYSTEM] Step %d executed write tools but git diff shows no new changes. "+
								"The tool may have failed silently. Verify and retry.", step.ID),
						})
						continue
					}
				}
			}

			// LSP完了検証（Normal Mode と対称にする - 1回限り）
			if !stepCompletionVerified {
				if needsContinue, feedback := a.verifyCompletionWithDiagnostics(response); needsContinue {
					stepCompletionVerified = true
					yellow.Println("⚠️  Step completion verification: LSP errors found in modified files")
					a.History = append(a.History, api.Message{
						Role:    "user",
						Content: feedback,
					})
					continue
				}
			}

			// ツール呼び出しなし = ステップ完了
			green.Printf("✓ Step %d completed\n", step.ID)
			return nil
		}

		// ツールを実行（parallel-safe なツールは並列実行）
		// loopDetectFn: 履歴は変更しない（executor の Phase 2 でメッセージ追加する）
		loopDetectFn := func(tc *tools.ToolCall) bool {
			cfg := a.cfg()
			threshold := cfg.LoopDetection.Threshold
			if isSameToolCall(tc, lastToolCall) {
				sameCallCount++
				if sameCallCount >= threshold {
					yellow.Printf("⚠️  Warning: Same tool call repeated %d times, stopping to prevent infinite loop\n", sameCallCount)
					yellow.Printf("   Tool: %s\n", tc.Tool)
					return true
				}
			} else {
				sameCallCount = 1
			}
			lastToolCall = tc
			return false
		}

		// skipFn: deprecated planning tools を実行前に除外
		skipFn := func(tc *tools.ToolCall) (bool, string) {
			if tc.Tool == "create_plan" || tc.Tool == "update_plan" {
				yellow.Printf("⚠️  Ignored deprecated planning tool call: %s\n", tc.Tool)
				return true, fmt.Sprintf("[%s] Ignored: planning tools are deprecated. Continue with current step.", tc.Tool)
			}
			return false, ""
		}

		a.executeToolCallsWithParallel(ctx, execToolCalls,
			loopDetectFn,
			skipFn,
			// 各ツール結果の処理
			func(_ int, toolCall *tools.ToolCall, result string, change *tools.FileChange) {
				// str_replace 成功時: LSP診断遅延バッファにファイルを追加
				if toolCall.Tool == "str_replace" && !strings.HasPrefix(result, "Error:") &&
					!strings.HasPrefix(result, "[CANCELLED]") && !strings.HasPrefix(result, "[COMMENT]") {
					if path := toolCall.Args["path"]; path != "" {
						a.addPendingLSPFile(path)
					}
				}

				// ステップ完了検証用: 書き込み系ツールの実行を記録
				if tools.IsWriteTool(toolCall.Tool) {
					stepHadWrites = true
					if strings.Contains(result, "no files found") ||
						strings.Contains(result, "Total matches: 0") ||
						strings.Contains(result, "no change needed") {
						stepHadNoChangeNeeded = true
					}
				}

				// 失敗パターンをチェック
				if toolCall.Tool == "bash" || tools.IsWriteTool(toolCall.Tool) {
					if failed, reason := plan.ContainsFailure(result); failed {
						lastFailedResult = result
						lastFailReason = reason
					}
				}

				// 変更履歴を保存
				a.handleFileChange(change)

				// ツール結果を履歴に追加（同一内容の重複は参照に差し替え）
				historyContent := a.deduplicateToolResult(toolCall.Tool, result)
				if toolCall.ID != "" {
					toolMsg := api.Message{
						Role:       "tool",
						Content:    historyContent,
						ToolCallID: toolCall.ID,
						ToolName:   toolCall.Tool,
					}
					a.History = append(a.History, toolMsg)
					if a.session != nil {
						a.session.AddMessageFromAPI(toolMsg, a.CurrentModel)
					}
				} else {
					a.History = append(a.History, api.Message{
						Role:    "user",
						Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, historyContent),
					})
				}
			},
		)

		// LSP診断遅延フラッシュ: 全ツール実行後に改めて診断を実行し結果を追記。
		// str_replace の直後ではなくここで実行することで、連続編集途中の
		// 「import not used」等の一時エラーによる誤 auto-retry を防ぐ。
		if diagMsg := a.flushLSPDiagnostics(); diagMsg != "" && len(a.History) > 0 {
			a.History[len(a.History)-1].Content += diagMsg
		}

		// 失敗検出時の処理
		if lastFailedResult != "" {
			cfg := a.cfg()
			autoRetryMax := cfg.PlanMode.MaxRetry

			// 自動リトライが有効で、まだ上限に達していない場合
			if autoRetryMax > 0 && retryCount < autoRetryMax {
				ui.StopGlobalSpinner()
				red.Printf("❌ Step %d Failed (auto-retry %d/%d)\n", step.ID, retryCount+1, autoRetryMax)
				yellow.Printf("🔄 Retrying...\n")

				// リトライ用プロンプトを追加
				a.History = append(a.History, api.Message{
					Role: "user",
					Content: fmt.Sprintf(`The previous step FAILED with the following error:

%s

Please:
1. Analyze the error carefully
2. Identify the root cause
3. Fix the code or configuration
4. Re-run the step to verify the fix

Do NOT skip this step. The issue must be resolved before proceeding.`, lastFailedResult),
				})
				return a.executeStepV2(ctx, p, step, idx, retryCount+1)
			}

			// 自動リトライが exhausted または無効 → Selector UI で確認
			a.SetStatus(StateWaitingApproval, "Step failed - waiting for action", "ステップ失敗 - アクション待ち", "Choose action", "アクションを選択")
			ui.StopGlobalSpinner()

			for {
				action, comment := promptFailureActionWithSelector(step, lastFailedResult, lastFailReason, autoRetryMax)

				switch action {
				case plan.FailureActionRetry:
					if retryCount >= maxRetries {
						red.Printf("⚠️  Max retries (%d) reached for step %d\n", maxRetries, step.ID)
						continue
					}
					// 手動リトライ: リトライカウンターをリセットして新しいシーケンスを開始
					a.History = append(a.History, api.Message{
						Role: "user",
						Content: fmt.Sprintf(`The previous step FAILED with the following error:

%s

Please:
1. Analyze the error carefully
2. Identify the root cause
3. Fix the code or configuration
4. Re-run the step to verify the fix

Do NOT skip this step. The issue must be resolved before proceeding.`, lastFailedResult),
					})
					return a.executeStepV2(ctx, p, step, idx, 0) // リセット: retryCount=0
				case plan.FailureActionComment:
					if retryCount >= maxRetries {
						red.Printf("⚠️  Max retries (%d) reached for step %d\n", maxRetries, step.ID)
						continue
					}
					// ユーザーの指示付きリトライ: リトライカウンターをリセット
					a.History = append(a.History, api.Message{
						Role: "user",
						Content: fmt.Sprintf(`The previous step FAILED. Here are the user's instructions for fixing it:

%s

Error that occurred:
%s

Please follow these instructions to fix the issue and retry the step.`, comment, lastFailedResult),
					})
					return a.executeStepV2(ctx, p, step, idx, 0) // リセット: retryCount=0
				case plan.FailureActionSkip:
					yellow.Printf("⏭️  Step %d skipped by user\n", step.ID)
					return nil
				case plan.FailureActionAbort:
					red.Printf("🛑 Step %d aborted by user\n", step.ID)
					return fmt.Errorf("step %d aborted by user: %s", step.ID, lastFailReason)
				}
			}
		}
	}

	green.Printf("✓ Step %d completed\n", step.ID)
	return nil
}

// isAIQuestionWithToolParser は AI が質問しているかを判定する。
// tool call parser は明示注入し、Agent には依存しない。
func isAIQuestionWithToolParser(response string, parseToolCalls func(string) []*tools.ToolCall) bool {
	// ツール呼び出しがある場合は質問とみなさない
	if parseToolCalls != nil && len(parseToolCalls(response)) > 0 {
		return false
	}

	questionPatterns := []string{
		"続行しますか", "よろしいですか", "確認してください", "どうしますか",
		"選択してください", "指定してください", "教えてください",
		"Should I", "Do you want", "Would you like", "Shall I",
		"Can you confirm", "Please confirm", "proceed?", "continue?",
	}

	lowered := strings.ToLower(response)
	for _, pattern := range questionPatterns {
		if strings.Contains(lowered, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func isAIQuestion(response string) bool {
	return isAIQuestionWithToolParser(response, tools.ParseToolCalls)
}

// getGitDiffHash は git diff HEAD + untracked files の出力を SHA256 ハッシュ化して返す。
// ファイル名だけでなく内容の変化も検知する（同じファイルへの追加変更を検出）。
// untracked ファイルはファイル名リストだけでなく内容もハッシュに含めることで、
// git add 前のファイルへの str_replace 等の変更を正しく検知する。
// git が使えない場合は空文字を返す（Level 2 スキップ用）。
func getGitDiffHash() string {
	// 1. tracked の差分（unstaged + staged）
	out, err := exec.Command("git", "diff", "HEAD").CombinedOutput()
	if err != nil {
		return ""
	}
	h := sha256.New()
	h.Write(out)

	// 2. untracked ファイルの名前と内容を両方ハッシュに含める
	untrackedOut, _ := exec.Command("git", "ls-files", "--others", "--exclude-standard").CombinedOutput()
	h.Write(untrackedOut) // ファイル名リスト（新規追加検知）
	for _, f := range strings.Split(strings.TrimSpace(string(untrackedOut)), "\n") {
		if f == "" {
			continue
		}
		content, err := os.ReadFile(f)
		if err == nil {
			h.Write(content) // 内容（編集検知）
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// confirmPlan は計画の承認確認
func (a *Agent) confirmPlan() (approved bool, feedback string) {
	result := tools.ConfirmInteractive("Approve this plan?")

	switch result.Action {
	case "yes":
		return true, ""
	case "no":
		return false, ""
	case "comment":
		return false, strings.TrimSpace(result.Comment)
	default:
		return false, ""
	}
}
