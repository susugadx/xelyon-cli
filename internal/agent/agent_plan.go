package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// RunPlanMode はPlan Modeで計画を生成・承認・実行
func (a *Agent) RunPlanMode(ctx context.Context, userRequest string) error {
	// Step 1: 計画生成をAIに依頼
	cyan.Println("\n📋 Generating execution plan...")

	// ユーザーリクエストを履歴に追加
	planMessage := api.Message{
		Role: "user",
		Content: fmt.Sprintf(`USER REQUEST: %s

IMPORTANT: You are in Plan Mode. Do NOT execute any tools yet.

Your task is to:
1. Analyze the user's request above
2. Break it down into a sequence of concrete steps
3. Output ONLY a JSON plan in this exact format (no other text):

{"steps": [
  {"id": 1, "description": "First step description", "tools": ["tool1", "tool2"], "depends_on": [], "parallel": false},
  {"id": 2, "description": "Second step description", "tools": ["tool3"], "depends_on": [1], "parallel": false}
]}

Rules:
- Start with id=1 and increment sequentially
- "description": Brief description of what this step does
- "tools": List of tool names to use (can be empty if no tools needed)
- "depends_on": List of step IDs that must complete before this step (empty array [] if no dependencies)
- "parallel": true if this step can run in parallel with other steps with same dependencies

Example plan:
{"steps": [
  {"id": 1, "description": "Read current Agent struct definition", "tools": ["read_file"], "depends_on": [], "parallel": false},
  {"id": 2, "description": "Add DryRunMode field to Agent struct", "tools": ["str_replace"], "depends_on": [1], "parallel": false},
  {"id": 3, "description": "Implement /dryrun command handler", "tools": ["write_file"], "depends_on": [1], "parallel": false},
  {"id": 4, "description": "Write tests for dry run functionality", "tools": ["write_file"], "depends_on": [2, 3], "parallel": false}
]}

Output the JSON plan now:`, userRequest),
	}
	historyWithRequest := append(a.History, planMessage)

	// AIに計画生成を依頼
	response, err := a.CurrentProvider.ChatWithTools(
		ctx,
		a.SystemPrompt,
		historyWithRequest,
		a.CurrentModel,
	)
	if err != nil {
		return fmt.Errorf("failed to generate plan: %w", err)
	}

	// デバッグ: レスポンスを表示（開発用）
	// fmt.Printf("DEBUG: AI response:\n%s\n", response)

	// Step 2: レスポンスからPlan JSONを抽出
	planJSON, err := ExtractPlanJSON(response)
	if err != nil {
		red.Printf("❌ Failed to extract plan JSON from AI response\n")
		yellow.Printf("AI Response:\n%s\n", response)
		return fmt.Errorf("failed to extract plan JSON: %w (AI may have returned a tool call instead of a plan)", err)
	}

	// Step 3: Planをパース
	plan, err := ParsePlan(planJSON)
	if err != nil {
		red.Printf("❌ Failed to parse plan JSON\n")
		yellow.Printf("Extracted JSON:\n%s\n", planJSON)
		return fmt.Errorf("failed to parse plan: %w", err)
	}

	// 空の計画チェック
	if len(plan.Steps) == 0 {
		red.Println("❌ Generated plan has no steps")
		return fmt.Errorf("empty plan generated")
	}

	// Step 4: 計画を表示
	fmt.Println()
	fmt.Println(FormatPlan(plan))
	fmt.Println()

	// Step 5: ユーザー承認
	approved, feedback := a.confirmPlan()
	if !approved {
		if feedback != "" {
			yellow.Printf("Plan rejected with feedback: %s\n", feedback)
			// フィードバック付き拒否の場合、再生成を試みる
			return a.RunPlanMode(ctx, userRequest+" (Previous plan feedback: "+feedback+")")
		}
		red.Println("Plan execution cancelled.")
		return nil
	}

	green.Println("✓ Plan approved. Starting execution...")

	// Step 6: 自律実行
	return a.executePlan(ctx, plan)
}

// confirmPlan は計画の承認確認
func (a *Agent) confirmPlan() (approved bool, feedback string) {
	yellow.Print("\nApprove this plan? [y/n/c(omment)]: ")

	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		red.Printf("Failed to read input: %v\n", err)
		return false, ""
	}
	input = strings.ToLower(strings.TrimSpace(input))

	switch input {
	case "y", "yes":
		return true, ""
	case "n", "no":
		return false, ""
	case "c", "comment":
		yellow.Print("Enter your feedback: ")
		var comment string
		if _, err := fmt.Scanln(&comment); err != nil {
			red.Printf("Failed to read input: %v\n", err)
			return false, ""
		}
		return false, strings.TrimSpace(comment)
	default:
		yellow.Println("Invalid input. Please enter y/n/c.")
		return a.confirmPlan() // 再帰的に再確認
	}
}

// executePlan は承認された計画を実行
func (a *Agent) executePlan(ctx context.Context, plan *Plan) error {
	for {
		// 次に実行すべきステップを取得
		nextStepID := plan.GetNextStep()
		if nextStepID == -1 {
			// すべて完了 or ブロック中
			if plan.IsCompleted() {
				green.Println("\n✓ All steps completed successfully!")
				return nil
			}
			if plan.HasFailed() {
				red.Println("\n✗ Plan execution failed.")
				return fmt.Errorf("plan execution failed")
			}
			// ブロック中（依存関係未解決）
			yellow.Println("\n⚠ Plan execution blocked. Waiting for dependencies...")
			return fmt.Errorf("plan execution blocked")
		}

		step := plan.GetStep(nextStepID)
		if step == nil {
			return fmt.Errorf("step %d not found", nextStepID)
		}

		// 並列実行可能なステップを取得
		parallelSteps := plan.GetParallelSteps()
		if len(parallelSteps) > 1 {
			// 並列実行
			if err := a.executeParallelSteps(ctx, plan, parallelSteps); err != nil {
				return err
			}
		} else {
			// 順次実行
			if err := a.executeStep(ctx, plan, step); err != nil {
				return err
			}
		}
	}
}

// executeStep は単一ステップを実行
func (a *Agent) executeStep(ctx context.Context, plan *Plan, step *PlanStep) error {
	cyan.Printf("\n[%d/%d] %s\n", step.ID, len(plan.Steps), step.Description)

	// ステータスを実行中に更新
	plan.UpdateStatus(step.ID, "running", "")

	// AIにステップ実行を依頼
	stepPrompt := fmt.Sprintf(`Execute this step from the plan:
Step %d: %s
Tools to use: %s

Execute this step autonomously. Only ask for confirmation if you need to perform SafetyLow operations (delete_file, bash with dangerous commands, git_push to main, etc.).`,
		step.ID,
		step.Description,
		strings.Join(step.Tools, ", "),
	)

	// 履歴に追加してAI実行
	a.History = append(a.History, api.Message{Role: "user", Content: stepPrompt})

	response, err := a.CurrentProvider.ChatWithTools(
		ctx,
		a.SystemPrompt,
		a.History,
		a.CurrentModel,
	)
	if err != nil {
		plan.UpdateStatus(step.ID, "failed", fmt.Sprintf("Error: %v", err))
		return fmt.Errorf("step %d failed: %w", step.ID, err)
	}

	// レスポンスを履歴に追加
	a.History = append(a.History, api.Message{Role: "assistant", Content: response})

	// 統計情報更新
	if a.Stats != nil {
		a.Stats.AssistantMessages++
	}

	// ツール呼び出しを処理
	// ループ: レスポンスにツール呼び出しが含まれている限り実行
	maxToolCalls := 10 // 無限ループ防止
	for i := 0; i < maxToolCalls; i++ {
		toolCall := tools.ParseToolCall(response)
		if toolCall == nil {
			// ツール呼び出しなし → ステップ完了
			break
		}

		// レスポンスから説明部分とツール呼び出しを分離
		explanation, _ := extractExplanationAndTool(response)

		// 説明部分を先に表示
		if explanation != "" {
			cyan.Println("\n💭 AI Explanation:")
			fmt.Println(explanation)
			fmt.Println()
		}

		// 統計情報更新: ツール実行回数
		if a.Stats != nil {
			a.Stats.AddToolExecution(toolCall.Tool)
		}

		// ツール実行
		result, change := tools.Execute(toolCall)

		// 変更履歴を保存
		if change != nil {
			a.changeStack = append(a.changeStack, *change)
			if len(a.changeStack) > config.MaxChangeStack {
				a.changeStack = a.changeStack[1:]
			}

			// 永続的変更履歴に保存
			if a.changeStorage != nil && a.session != nil {
				if err := a.changeStorage.AppendChange(a.session.ID, *change); err != nil {
					yellow.Printf("Warning: Failed to persist change: %v\n", err)
				}
			}
		}

		// 結果を履歴に追加
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: fmt.Sprintf("[Tool Result]\n%s", result),
		})

		// 次のAI応答を取得
		response, err = a.CurrentProvider.ChatWithTools(
			ctx,
			a.SystemPrompt,
			a.History,
			a.CurrentModel,
		)
		if err != nil {
			plan.UpdateStatus(step.ID, "failed", fmt.Sprintf("Error: %v", err))
			return fmt.Errorf("step %d failed: %w", step.ID, err)
		}

		// レスポンスを履歴に追加
		a.History = append(a.History, api.Message{Role: "assistant", Content: response})

		// 統計情報更新
		if a.Stats != nil {
			a.Stats.AssistantMessages++
		}
	}

	// ステップ完了
	plan.UpdateStatus(step.ID, "completed", "Success")
	green.Printf("✓ Step %d completed\n", step.ID)

	return nil
}

// executeParallelSteps は複数ステップを並列実行
func (a *Agent) executeParallelSteps(ctx context.Context, plan *Plan, stepIDs []int) error {
	cyan.Printf("\n🔀 Executing %d steps in parallel...\n", len(stepIDs))

	// 並列実行用のゴルーチンを起動
	// TODO: internal/agent/parallel.go で実装
	for _, stepID := range stepIDs {
		step := plan.GetStep(stepID)
		if step == nil {
			continue
		}
		if err := a.executeStep(ctx, plan, step); err != nil {
			return err
		}
	}

	return nil
}
