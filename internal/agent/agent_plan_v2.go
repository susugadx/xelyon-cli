package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// RunPlanModeV2 は Claude Code 風の Plan Mode を実行
// - 調査ツール(SafetyHigh)は即座に実行
// - 実装ツール(SafetyMedium/Low)の前に計画を生成・承認
func (a *Agent) RunPlanModeV2(ctx context.Context, userRequest string) error {
	// 実装前チェック：既存定義の重複を警告
	if warning := CheckBeforeImplementation(userRequest); warning != "" {
		yellow.Println(warning)
		userRequest = userRequest + "\n\n[SYSTEM NOTE: " + warning + " Please check existing code before creating new definitions.]"
	}

	// Step 1: 調査フェーズ（SafetyHighツールを自由に実行）
	cyan.Println("\n🔍 Investigation phase - researching the codebase...")
	a.SetStatus(StateRunning, "Investigating", "調査中", "Wait for investigation", "調査完了を待ってください")

	// ユーザーリクエストを履歴に追加
	investigationPrompt := fmt.Sprintf(`USER REQUEST: %s

You are in PLAN MODE (Investigation Phase).

IMPORTANT RULES:
1. First, investigate the codebase to understand what needs to be done
2. You CAN use read-only tools freely: read_file, search_file, search_code, list_dir, git_status, git_log, git_diff, lint, test, web_search
3. Do NOT use any modification tools yet (write_file, str_replace, delete_file, bash, etc.)
4. When you have gathered enough information, output a plan for the implementation

After investigation, output your plan in this JSON format:
{"plan": {
  "summary": "Brief summary of what will be done",
  "steps": [
    {"id": 1, "description": "Step description", "tools": ["tool1"]},
    {"id": 2, "description": "Step description", "tools": ["tool2"]}
  ]
}}

Start your investigation now. Use read_file, search_code, list_dir etc. to understand the codebase.`, userRequest)

	a.History = append(a.History, api.Message{Role: "user", Content: investigationPrompt})

	// 調査フェーズ: SafetyHighツールを実行
	planJSON, err := a.runInvestigationPhase(ctx)
	if err != nil {
		return err
	}

	// 計画が空の場合（調査のみで完了、または実装不要）
	if planJSON == "" {
		green.Println("\n✓ Investigation complete. No implementation needed.")
		a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
		return nil
	}

	// Step 2: 計画をパースして表示
	plan, err := ParsePlanV2(planJSON)
	if err != nil {
		red.Printf("❌ Failed to parse plan: %v\n", err)
		yellow.Printf("Plan JSON:\n%s\n", planJSON)
		return err
	}

	if len(plan.Steps) == 0 {
		green.Println("\n✓ Investigation complete. No implementation steps needed.")
		a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
		return nil
	}

	// 計画を表示
	fmt.Println()
	cyan.Println("📋 Implementation Plan:")
	cyan.Println(strings.Repeat("─", 50))
	if plan.Summary != "" {
		fmt.Printf("Summary: %s\n\n", plan.Summary)
	}
	for _, step := range plan.Steps {
		fmt.Printf("  %d. %s\n", step.ID, step.Description)
		if len(step.Tools) > 0 {
			fmt.Printf("     Tools: %s\n", strings.Join(step.Tools, ", "))
		}
	}
	cyan.Println(strings.Repeat("─", 50))
	fmt.Println()

	// Step 3: ユーザー承認
	a.SetStatus(StateWaitingApproval, "Waiting for plan approval", "計画の承認待ち", "Answer y/n/c", "y/n/c で回答")
	approved, feedback := a.confirmPlan()
	if !approved {
		if feedback != "" {
			yellow.Printf("Plan rejected with feedback: %s\n", feedback)
			// フィードバック付き拒否の場合、再生成を試みる
			return a.RunPlanModeV2(ctx, userRequest+" (Previous plan feedback: "+feedback+")")
		}
		red.Println("Plan execution cancelled.")
		return nil
	}

	green.Println("✓ Plan approved. Starting implementation...")
	a.SetStatus(StateRunning, "Implementing", "実装中", "Wait for completion", "完了を待ってください")

	// Step 4: 実装フェーズ
	err = a.runImplementationPhase(ctx, plan)
	if err != nil {
		a.SetStatus(StateAborted, "Implementation failed", "実装に失敗", "Review errors and retry", "エラーを確認して再試行")
		return err
	}

	a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
	return nil
}

// runInvestigationPhase は調査フェーズを実行
// SafetyHighツールのみを実行し、計画JSONを返す
func (a *Agent) runInvestigationPhase(ctx context.Context) (string, error) {
	maxIterations := config.MaxToolIterations
	var lastToolCall *tools.ToolCall
	var sameCallCount int

	for i := 0; i < maxIterations; i++ {
		response, err := a.CurrentProvider.ChatWithTools(
			ctx,
			a.SystemPrompt,
			a.History,
			a.CurrentModel,
		)
		if err != nil {
			return "", fmt.Errorf("investigation failed: %w", err)
		}

		a.History = append(a.History, api.Message{Role: "assistant", Content: response})
		if a.Stats != nil {
			a.Stats.AssistantMessages++
		}

		// 計画JSONを検出
		if planJSON := ExtractPlanV2JSON(response); planJSON != "" {
			return planJSON, nil
		}

		// ツール呼び出しチェック
		toolCalls := tools.ParseToolCalls(response)
		if len(toolCalls) == 0 {
			// ツール呼び出しも計画もない場合は終了
			// AIが調査を終えて単に説明しているか、質問している場合
			fmt.Println(response)
			return "", nil
		}

		// ツールを実行
		for _, toolCall := range toolCalls {
			// ループ検知
			if a.shouldAbortToolLoop(toolCall, lastToolCall, &sameCallCount) {
				return "", fmt.Errorf("tool loop detected during investigation")
			}
			lastToolCall = toolCall

			safety := tools.GetToolSafety(toolCall.Tool)
			if safety != tools.SafetyHigh {
				// SafetyMedium/Low ツールが呼ばれた = 調査フェーズ終了、計画生成を要求
				cyan.Printf("\n⚡ Implementation tool detected: %s\n", toolCall.Tool)
				cyan.Println("   Requesting implementation plan...")

				// AIに計画生成を要求
				a.History = append(a.History, api.Message{
					Role: "user",
					Content: fmt.Sprintf(`[SYSTEM] You tried to use a modification tool (%s) during the investigation phase.

Before using modification tools, you must output an implementation plan.
Output your plan now in this JSON format:

{"plan": {
  "summary": "Brief summary of what will be done",
  "steps": [
    {"id": 1, "description": "Step description", "tools": ["tool1"]},
    {"id": 2, "description": "Step description", "tools": ["tool2"]}
  ]
}}`, toolCall.Tool),
				})

				// 次のイテレーションで計画を取得
				continue
			}

			// SafetyHighツールを実行
			if a.Stats != nil {
				a.Stats.AddToolExecution(toolCall.Tool)
			}

			result, _ := tools.Execute(toolCall)

			a.History = append(a.History, api.Message{
				Role:    "user",
				Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result),
			})
		}
	}

	yellow.Printf("⚠️  調査フェーズが%d回のツール実行に達しました。続けて指示してください。\n", maxIterations)
	return "", nil
}

// runImplementationPhase は実装フェーズを実行
func (a *Agent) runImplementationPhase(ctx context.Context, plan *PlanV2) error {
	for idx, step := range plan.Steps {
		cyan.Printf("\n[%d/%d] %s\n", idx+1, len(plan.Steps), step.Description)

		// ステップ実行を指示
		stepPrompt := fmt.Sprintf(`Execute step %d of the implementation plan:
%s

Use the appropriate tools to complete this step. You may use modification tools now.`, step.ID, step.Description)

		a.History = append(a.History, api.Message{Role: "user", Content: stepPrompt})

		// ステップ内のツール実行ループ
		maxStepIterations := 10
		for j := 0; j < maxStepIterations; j++ {
			response, err := a.CurrentProvider.ChatWithTools(
				ctx,
				a.SystemPrompt,
				a.History,
				a.CurrentModel,
			)
			if err != nil {
				return fmt.Errorf("step %d failed: %w", step.ID, err)
			}

			a.History = append(a.History, api.Message{Role: "assistant", Content: response})
			if a.Stats != nil {
				a.Stats.AssistantMessages++
			}

			// ツール呼び出しチェック
			toolCalls := tools.ParseToolCalls(response)
			if len(toolCalls) == 0 {
				// ツール呼び出しなし = ステップ完了
				green.Printf("✓ Step %d completed\n", step.ID)
				break
			}

			// ツールを実行
			for _, toolCall := range toolCalls {
				if a.Stats != nil {
					a.Stats.AddToolExecution(toolCall.Tool)
				}

				result, change := tools.Execute(toolCall)

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

				a.History = append(a.History, api.Message{
					Role:    "user",
					Content: fmt.Sprintf("[Tool Result for %s]\n%s", toolCall.Tool, result),
				})
			}
		}
	}

	green.Printf("\n✓ All %d steps completed!\n", len(plan.Steps))
	return nil
}

// PlanV2 は新しい計画フォーマット
type PlanV2 struct {
	Summary string       `json:"summary"`
	Steps   []PlanStepV2 `json:"steps"`
}

// PlanStepV2 は計画のステップ
type PlanStepV2 struct {
	ID          int      `json:"id"`
	Description string   `json:"description"`
	Tools       []string `json:"tools"`
}

// ExtractPlanV2JSON はレスポンスから計画JSONを抽出
func ExtractPlanV2JSON(response string) string {
	// {"plan": ... } パターンを探す
	patterns := []string{
		`{"plan"`,
		`{ "plan"`,
	}

	for _, pattern := range patterns {
		idx := strings.Index(response, pattern)
		if idx == -1 {
			continue
		}

		// 対応する閉じ括弧を探す
		depth := 0
		for i := idx; i < len(response); i++ {
			if response[i] == '{' {
				depth++
			} else if response[i] == '}' {
				depth--
				if depth == 0 {
					return response[idx : i+1]
				}
			}
		}
	}

	return ""
}

// ParsePlanV2 は計画JSONをパース
func ParsePlanV2(jsonStr string) (*PlanV2, error) {
	// {"plan": {...}} の形式から plan 部分を抽出
	type wrapper struct {
		Plan PlanV2 `json:"plan"`
	}

	var w wrapper
	if err := json.Unmarshal([]byte(jsonStr), &w); err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}

	return &w.Plan, nil
}
