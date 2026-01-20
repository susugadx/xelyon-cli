package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// RunPlanMode は Claude Code 風の Plan Mode を実行
// - 調査ツール(SafetyHigh)は即座に実行
// - 実装ツール(SafetyMedium/Low)の前に計画を生成・承認
func (a *Agent) RunPlanMode(ctx context.Context, userRequest string) error {
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
	plan, err := ParsePlan(planJSON)
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
			return a.RunPlanMode(ctx, userRequest+" (Previous plan feedback: "+feedback+")")
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
