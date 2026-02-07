package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/prompt"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// RunPlanMode は Claude Code 風の Plan Mode を実行
// - 調査ツール(SafetyHigh)は即座に実行
// - 実装ツール(SafetyMedium/Low)の前に計画を生成・承認
func (a *Agent) RunPlanMode(ctx context.Context, userRequest string) error {
	// Plan Mode 用のプロンプトを SystemPrompt に追加（一度だけ）
	planningPrompt := promptplan.BuildPlanningPrompt()
	if !strings.Contains(a.SystemPrompt, planningPrompt) {
		a.SystemPrompt = prompt.BuildSystemPrompt(a.SystemPrompt, true)
	}

	// 実装前チェック：既存定義の重複を警告
	if warning := CheckBeforeImplementation(userRequest); warning != "" {
		yellow.Println(warning)
		userRequest = userRequest + "\n\n[SYSTEM NOTE: " + warning + " Please check existing code before creating new definitions.]"
	}

	// 並列モードが有効な場合は Supervisor に委譲
	cfg := config.GetGlobalConfig()
	config.MigratePlanModeConfig(&cfg.PlanMode)

	if cfg.PlanMode.Parallel {
		supervisor, err := NewSupervisor(a, &cfg.PlanMode)
		if err != nil {
			red.Printf("❌ Failed to create supervisor: %v\n", err)
			return err
		}

		// Supervisor実行（トークン上限エラー時は自動圧縮+リトライ）
		err = supervisor.Run(ctx, userRequest)
		if err != nil && token.IsTokenLimitError(err) {
			// リトライ関数を定義
			retryFunc := func() error {
				return supervisor.Run(ctx, userRequest)
			}

			// 自動圧縮+リトライを試みる
			if a.handleTokenLimitErrorWithRetry(err, retryFunc, true, supervisor) {
				return nil // リトライ成功
			}
		}
		return err
	}

	// 以下は順次実行モード（従来の処理）

	// Step 1: 調査フェーズ（SafetyHighツールを自由に実行）
	cyan.Println("\n🔍 Investigation phase - researching the codebase...")
	a.SetStatus(StateRunning, "Investigating", "調査中", "Wait for investigation", "調査完了を待ってください")

	// ユーザーリクエストを履歴に追加
	investigationPrompt := promptplan.BuildInvestigationPrompt(userRequest)

	a.History = append(a.History, api.Message{Role: "user", Content: investigationPrompt})

	// 調査フェーズ: SafetyHighツールを実行し、create_plan ツールで Plan を作成
	p, err := a.runInvestigationPhase(ctx)
	if err != nil {
		// トークン上限エラーの場合は自動圧縮+リトライ
		if token.IsTokenLimitError(err) {
			retryFunc := func() error {
				return a.RunPlanMode(ctx, userRequest)
			}
			if a.handleTokenLimitErrorWithRetry(err, retryFunc, true, nil) {
				return nil // リトライ成功
			}
		}
		return err
	}

	// 計画が空の場合（調査のみで完了、または実装不要）
	if p == nil {
		green.Println("\n✓ Investigation complete. No implementation needed.")
		a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
		return nil
	}

	if len(p.Steps) == 0 {
		green.Println("\n✓ Investigation complete. No implementation steps needed.")
		a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
		return nil
	}

	// 計画を構造化表示
	planDisplay := ui.NewPlanDisplay("Implementation Plan").
		SetSummary(p.Summary)

	for _, step := range p.Steps {
		planDisplay.AddStep(step.ID, step.Description, step.Tools, step.TargetFiles)
	}

	fmt.Println()
	fmt.Print(planDisplay.Render())

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
	err = a.runImplementationPhase(ctx, p)
	if err != nil {
		// トークン上限エラーの場合は自動圧縮+リトライ
		if token.IsTokenLimitError(err) {
			retryFunc := func() error {
				return a.RunPlanMode(ctx, userRequest)
			}
			if a.handleTokenLimitErrorWithRetry(err, retryFunc, true, nil) {
				return nil // リトライ成功
			}
		}
		a.SetStatus(StateAborted, "Implementation failed", "実装に失敗", "Review errors and retry", "エラーを確認して再試行")
		return err
	}

	// タスク完了サマリーを表示
	a.showTaskSummary()

	a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
	return nil
}
