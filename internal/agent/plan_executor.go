package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
		if hooks := config.GetGlobalConfig().Hooks; len(hooks.OnStepComplete) > 0 {
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
	cfg := config.GetGlobalConfig()
	maxStepIterations := cfg.General.ToolLoopLimit
	maxContinues := config.PlanMaxAutoContinues
	continueCount := 0
	var lastFailedResult string
	var lastFailReason string

	// ステップ完了検証用トラッカー
	executedTools := make(map[string]bool) // ステップ内で実行されたツール名
	stepHadWrites := false                 // 書き込み系ツールが実行されたか
	stepHadNoChangeNeeded := false         // 書き込み系ツールが「変更不要」と返したか
	beforeDiffHash := getGitDiffHash()     // Level 2 用: ステップ開始時の diff ハッシュ
	var stepCompletionVerified bool        // LSP完了検証ガード（ステップ内1回限り）

	// ループ検知用トラッカー
	var lastToolCall *tools.ToolCall
	var sameCallCount int

	for j := 0; j < maxStepIterations; j++ {
		response, err := a.CurrentProvider.ChatWithTools(
			ctx,
			a.SystemPrompt,
			a.History,
			a.CurrentModel,
		)
		if err != nil {
			ui.StopGlobalSpinner()
			return fmt.Errorf("step %d failed: %w", step.ID, err)
		}

		// ツール呼び出しチェック
		toolCalls := tools.ParseToolCalls(response)
		skippedWrites := 0
		skippedCommands := 0
		writeQueued := false

		// FC rescue: テキストから抽出された toolCall にダミー ID を注入
		// これにより下流の処理が FC 成功時と同じパス（role:"tool"）を通る
		for i, tc := range toolCalls {
			if tc.ID == "" {
				toolCalls[i].ID = fmt.Sprintf("call_rescue_%03d", i+1)
			}
		}

		// Write Throttle と bash スキップの適用
		var execToolCalls []*tools.ToolCall // 実行対象のツール呼び出し
		for _, tc := range toolCalls {
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
			if isAIQuestion(response) && continueCount < maxContinues {
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

			// Level 1: 計画の tools に書き込み系があるのに未実行 → 強制続行
			if missing := checkMissingWriteTools(step.Tools, executedTools); len(missing) > 0 {
				// エスケープ: AI が「既に適用済み」と宣言 + diff に対象ファイルの変更あり
				if containsAlreadyAppliedDeclaration(response) && diffContainsFileChanges(step.Files) {
					green.Printf("✓ Step %d completed (verified in diff)\n", step.ID)
					return nil
				}
				if continueCount < maxContinues {
					continueCount++
					yellow.Printf("⚠️  Step %d: expected %s but never executed (%d/%d)\n",
						step.ID, strings.Join(missing, ", "), continueCount, maxContinues)
					a.History = append(a.History, api.Message{
						Role: "user",
						Content: fmt.Sprintf("[SYSTEM] Step %d requires %s but none were executed. "+
							"Do NOT declare completion. Execute the required tools now.", step.ID, strings.Join(missing, ", ")),
					})
					continue
				}
				// maxContinues 到達 → Selector UI で確認
				yellow.Printf("⚠️  Step %d: required %s never executed after %d attempts.\n",
					step.ID, strings.Join(missing, ", "), maxContinues)
				ui.StopGlobalSpinner()
				a.SetStatus(StateWaitingApproval, "Step incomplete - waiting for action", "ステップ未完了 - アクション待ち", "Choose action", "アクションを選択")
				failMsg := fmt.Sprintf("Required tools [%s] were never executed after %d attempts", strings.Join(missing, ", "), maxContinues)
				action, comment := promptFailureActionWithSelector(step, failMsg, "required tools not executed", 0)
				switch action {
				case plan.FailureActionRetry:
					return a.executeStepV2(ctx, p, step, idx, 0)
				case plan.FailureActionComment:
					a.History = append(a.History, api.Message{
						Role:    "user",
						Content: fmt.Sprintf("[USER] %s", comment),
					})
					return a.executeStepV2(ctx, p, step, idx, 0)
				case plan.FailureActionSkip:
					yellow.Printf("⏭️  Step %d skipped by user\n", step.ID)
					return nil
				case plan.FailureActionAbort:
					return fmt.Errorf("step %d aborted by user", step.ID)
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

		// ツールを実行
		for tcIdx, toolCall := range execToolCalls {
			// ループ検知（assistant メッセージは追加済みなので response="" で呼ぶ）
			if a.shouldAbortToolLoopWithResponse("", toolCall, lastToolCall, &sameCallCount) {
				// 残りの未実行 TC にダミー結果を追加
				for _, remaining := range execToolCalls[tcIdx+1:] {
					if remaining.ID != "" {
						a.History = append(a.History, api.Message{
							Role:       "tool",
							Content:    "[SYSTEM] Skipped due to tool loop detection.",
							ToolCallID: remaining.ID,
							ToolName:   remaining.Tool,
						})
					}
				}
				break
			}
			lastToolCall = toolCall

			// Plan 実行中は create_plan を無視（再帰防止）
			if toolCall.Tool == "create_plan" {
				// create_plan を無視したことを履歴に追加
				ignoreMsg := "[create_plan] Ignored: already executing a plan. Continue with current step."
				if toolCall.ID != "" {
					// Function Calling: role="tool" で tool_call_id 付きで送信
					toolMsg := api.Message{
						Role:       "tool",
						Content:    ignoreMsg,
						ToolCallID: toolCall.ID,
						ToolName:   toolCall.Tool,
					}
					a.History = append(a.History, toolMsg)
					if a.session != nil {
						a.session.AddMessageFromAPI(toolMsg, a.CurrentModel)
					}
				} else {
					// テキストベース: role="user" で送信
					a.History = append(a.History, api.Message{
						Role:    "user",
						Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, ignoreMsg),
					})
				}
				continue
			}

			// ツール実行（executeToolOnly と同じパターン）
			result, change := tools.Execute(toolCall)

			// 書き込み成功後の自動 read-back 処理
			if a.shouldThrottleWrite(toolCall) {
				if !strings.HasPrefix(result, "Error:") {
					a.autoReadBack(toolCall)
				}
			}

			// str_replace 成功時: LSP診断遅延バッファにファイルを追加
			// 連続 str_replace 途中の一時的エラーによる誤 auto-retry を防ぐため、
			// 診断は全ツール実行後にまとめて行う（flushLSPDiagnostics）。
			if toolCall.Tool == "str_replace" && !strings.HasPrefix(result, "Error:") &&
				!strings.HasPrefix(result, "[CANCELLED]") && !strings.HasPrefix(result, "[COMMENT]") {
				if path := toolCall.Args["path"]; path != "" {
					a.addPendingLSPFile(path)
				}
			}

			// ステップ完了検証用: 実行されたツールを記録
			executedTools[toolCall.Tool] = true
			if tools.IsWriteTool(toolCall.Tool) {
				stepHadWrites = true

				// 変更不要・対象なしの場合はLevel 2ガードをスキップ
				if strings.Contains(result, "no files found") ||
					strings.Contains(result, "Total matches: 0") ||
					strings.Contains(result, "no change needed") {
					stepHadNoChangeNeeded = true
				}
			}

			// 失敗パターンをチェック
			if failed, reason := plan.ContainsFailure(result); failed {
				lastFailedResult = result
				lastFailReason = reason
			}

			// 変更履歴を保存
			if change != nil {
				a.changeStack = append(a.changeStack, *change)
				if len(a.changeStack) > config.MaxChangeStack {
					a.changeStack = a.changeStack[1:]
				}

				if a.changeStorage != nil && a.session != nil {
					if err := a.changeStorage.AppendChange(a.session.ID, *change); err != nil {
						yellow.Printf("Warning: Failed to persist change: %v\n", err)
					}
				}
			}

			// ツール結果を履歴に追加（executeToolOnly と同じパターン）
			if toolCall.ID != "" {
				// Function Calling: role="tool" で tool_call_id 付きで送信
				toolMsg := api.Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: toolCall.ID,
					ToolName:   toolCall.Tool,
				}
				a.History = append(a.History, toolMsg)

				// セッションに tool result を保存
				if a.session != nil {
					a.session.AddMessageFromAPI(toolMsg, a.CurrentModel)
				}
			} else {
				// テキストベース: role="user" で送信（従来方式）
				a.History = append(a.History, api.Message{
					Role:    "user",
					Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result),
				})
			}
		}

		// スキップされた書き込みとコマンドがあればメッセージを注入して通知
		if skippedWrites > 0 || skippedCommands > 0 {
			a.injectWriteThrottleMessage(skippedWrites, skippedCommands)
		}

		// LSP診断遅延フラッシュ: 全ツール実行後に改めて診断を実行し結果を追記。
		// str_replace の直後ではなくここで実行することで、連続編集途中の
		// 「import not used」等の一時エラーによる誤 auto-retry を防ぐ。
		if diagMsg := a.flushLSPDiagnostics(); diagMsg != "" && len(a.History) > 0 {
			a.History[len(a.History)-1].Content += diagMsg
		}

		// 失敗検出時の処理
		if lastFailedResult != "" {
			cfg := config.GetGlobalConfig()
			autoRetryMax := cfg.PlanMode.AutoRetry

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

// isAIQuestion はAIが質問しているかを判定
func isAIQuestion(response string) bool {
	// ツール呼び出しがある場合は質問とみなさない
	if len(tools.ParseToolCalls(response)) > 0 {
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

// checkMissingWriteTools は step.Tools に含まれる書き込み系ツールのうち
// 実際に実行されなかったものを返す。
func checkMissingWriteTools(stepTools []string, executedTools map[string]bool) []string {
	var missing []string
	for _, t := range stepTools {
		if tools.IsWriteTool(t) && !executedTools[t] {
			missing = append(missing, t)
		}
	}
	return missing
}

// containsAlreadyAppliedDeclaration はAI応答に「前ステップで適用済み」系パターンが含まれるか検出
func containsAlreadyAppliedDeclaration(response string) bool {
	lowered := strings.ToLower(response)
	patterns := []string{
		"already applied",
		"already been applied",
		"already been made",
		"already been implemented",
		"already modified",
		"already changed",
		"already updated",
		"already exists",
		"previous step",
		"earlier step",
		"already handled",
		"already done",
		"already completed",
	}
	for _, p := range patterns {
		if strings.Contains(lowered, p) {
			return true
		}
	}
	// 日本語パターン
	jpPatterns := []string{
		"既に適用",
		"既に修正",
		"既に変更",
		"既に実装",
		"前のステップ",
		"先ほどのステップ",
		"適用済み",
		"修正済み",
		"変更済み",
	}
	for _, p := range jpPatterns {
		if strings.Contains(response, p) {
			return true
		}
	}
	return false
}

// diffContainsFileChanges は git diff HEAD に指定ファイルへの変更が含まれるか確認
func diffContainsFileChanges(files []string) bool {
	if len(files) == 0 {
		return false
	}
	out, err := exec.Command("git", "diff", "HEAD", "--name-only").CombinedOutput()
	if err != nil {
		return false
	}
	diffOutput := strings.TrimSpace(string(out))
	if diffOutput == "" {
		return false
	}
	changedFiles := strings.Split(diffOutput, "\n")
	changedSet := make(map[string]bool)
	for _, f := range changedFiles {
		changedSet[strings.TrimSpace(f)] = true
	}
	for _, target := range files {
		if changedSet[target] {
			return true
		}
		// パス末尾マッチ（"internal/api/foo.go" vs "foo.go"）
		for changed := range changedSet {
			if strings.HasSuffix(changed, "/"+target) || strings.HasSuffix(target, "/"+changed) {
				return true
			}
		}
	}
	return false
}

// getGitDiffHash は git diff HEAD + untracked files の出力を SHA256 ハッシュ化して返す。
// ファイル名だけでなく内容の変化も検知する（同じファイルへの追加変更を検出）。
// git が使えない場合は空文字を返す（Level 2 スキップ用）。
func getGitDiffHash() string {
	out, err := exec.Command("git", "diff", "HEAD").CombinedOutput()
	if err != nil {
		return ""
	}
	// untracked files も含める
	untrackedOut, _ := exec.Command("git", "ls-files", "--others", "--exclude-standard").CombinedOutput()
	h := sha256.New()
	h.Write(out)
	h.Write(untrackedOut)
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
