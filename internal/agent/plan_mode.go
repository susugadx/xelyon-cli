package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	promptplan "github.com/susugadx/xelyon-cli/internal/prompt/plan"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// RunPlanMode は Claude Code 風の Plan Mode を実行
// - 調査ツール(SafetyHigh)は即座に実行
// - 実装ツール(SafetyMedium/Low)の前に計画を生成・承認
func (a *Agent) RunPlanMode(ctx context.Context, userRequest string) error {
	out := a.output()

	// Plan Mode 用のプロンプトを SystemPrompt に追加（一度だけ）
	// キャッシュ境界マーカーを挟んで追加することで、base prompt のキャッシュを維持する
	planningPrompt := promptplan.BuildPlanningPrompt()
	if !strings.Contains(a.SystemPrompt, planningPrompt) {
		a.SystemPrompt = a.SystemPrompt + api.SystemPromptCacheBoundary + planningPrompt
	}

	// 実装前チェック：既存定義の重複を警告
	if warning := CheckBeforeImplementation(userRequest); warning != "" {
		yellow.Fprintln(out, warning)
		userRequest = userRequest + "\n\n[SYSTEM NOTE: " + warning + " Please check existing code before creating new definitions.]"
	}

	// Step 1: 調査フェーズ（SafetyHighツールを自由に実行）
	cyan.Fprintln(out, "\n🔍 Investigation phase - researching the codebase...")
	a.SetStatus(StateRunning, "Investigating", "調査中", "Wait for investigation", "調査完了を待ってください")

	// ユーザーリクエストを履歴に追加
	investigationPrompt := promptplan.BuildInvestigationPrompt(userRequest)

	a.History = append(a.History, api.Message{Role: "user", Content: investigationPrompt})

	// 調査フェーズ: SafetyHighツールを実行し、Plan JSON をテキスト出力→抽出/パースして Plan を作成
	p, err := a.runInvestigationPhase(ctx)
	if err != nil {
		// トークン上限エラーの場合は自動圧縮+リトライ
		if token.IsTokenLimitError(err) {
			retryFunc := func() error {
				return a.RunPlanMode(ctx, userRequest)
			}
			if a.handleTokenLimitErrorWithRetry(err, retryFunc, true) {
				return nil // リトライ成功
			}
		}
		return err
	}

	// 計画が空の場合（調査のみで完了、または実装不要）
	if p == nil {
		green.Fprintln(out, "\n✓ Investigation complete. No implementation needed.")
		a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
		return nil
	}

	if len(p.Steps) == 0 {
		green.Fprintln(out, "\n✓ Investigation complete. No implementation steps needed.")
		a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
		return nil
	}

	// 計画を構造化表示
	planDisplay := ui.NewPlanDisplay("Implementation Plan").
		SetSummary(p.Summary)

	for _, step := range p.Steps {
		planDisplay.AddStep(step.ID, step.Description, step.Tools, step.TargetFiles)
	}

	_, _ = fmt.Fprintln(out)
	_, _ = fmt.Fprint(out, planDisplay.Render())

	// Step 3: ユーザー承認
	a.SetStatus(StateWaitingApproval, "Waiting for plan approval", "計画の承認待ち", "Answer y/n/c", "y/n/c で回答")
	approved, feedback := a.confirmPlan()
	if !approved {
		if feedback != "" {
			yellow.Fprintf(out, "Plan rejected with feedback: %s\n", feedback)
			// フィードバック付き拒否の場合、再生成を試みる
			return a.RunPlanMode(ctx, userRequest+" (Previous plan feedback: "+feedback+")")
		}
		red.Fprintln(out, "Plan execution cancelled.")
		return nil
	}

	green.Fprintln(out, "✓ Plan approved. Starting implementation...")
	if a.cfg().PlanMode.ClearContextOnApproval {
		a.clearContextForImplementation(p, userRequest)
	}
	a.SetStatus(StateRunning, "Implementing", "実装中", "Wait for completion", "完了を待ってください")

	// Step 4: 実装フェーズ
	err = a.runImplementationPhase(ctx, p)
	if err != nil {
		// トークン上限エラーの場合は自動圧縮+リトライ
		if token.IsTokenLimitError(err) {
			retryFunc := func() error {
				return a.RunPlanMode(ctx, userRequest)
			}
			if a.handleTokenLimitErrorWithRetry(err, retryFunc, true) {
				return nil // リトライ成功
			}
		}
		a.SetStatus(StateAborted, "Implementation failed", "実装に失敗", "Review errors and retry", "エラーを確認して再試行")
		return err
	}

	// Completion hooks（Plan Mode）
	a.runCompletionHooksWithRetry(ctx)

	// タスク完了サマリーを表示
	a.showTaskSummary()

	a.SetStatus(StateWaitingInput, "Ready for input", "入力待ち", "Type your request or /help", "リクエスト、または /help を入力")
	return nil
}

func (a *Agent) clearContextForImplementation(p *plan.Plan, userRequest string) {
	beforeTokens := a.EstimateTokens()
	planSummary := buildPlanContextSummary(p, userRequest)

	a.History = []api.Message{{
		Role:    "user",
		Content: planSummary,
	}}

	a.compactedItems = nil
	a.isCompactedMode = false
	a.readTracker.reset()

	if a.session != nil {
		a.session.Messages = nil
		a.session.CompactedItems = nil
		a.session.IsCompactedMode = false
		a.session.Model = a.CurrentModel
		a.session.ResponseID = ""
		a.session.AddMessageFromAPI(a.History[0], a.CurrentModel)
	}

	if clearable, ok := a.CurrentProvider.(interface{ ClearResponseID() }); ok {
		clearable.ClearResponseID()
	}

	if a.storage != nil && a.session != nil {
		if err := a.storage.Rewrite(a.session); err != nil {
			yellow.Fprintf(a.output(), "Warning: Failed to rewrite session after context clear: %v\n", err)
		}
	}

	afterTokens := a.EstimateTokens()
	dim.Fprintf(a.output(), "   Context cleared for implementation (%dK -> %dK)\n", beforeTokens/1000, afterTokens/1000)
}

func buildPlanContextSummary(p *plan.Plan, userRequest string) string {
	if p == nil {
		return "[Approved Implementation Plan]\n\nProceed with implementation step by step."
	}

	req := strings.TrimSpace(userRequest)
	if req == "" {
		req = strings.TrimSpace(p.UserRequest)
	}

	var b strings.Builder
	b.WriteString("[Approved Implementation Plan]\n\n")

	if req != "" {
		b.WriteString("Original request: ")
		b.WriteString(req)
		b.WriteString("\n\n")
	}

	if summary := strings.TrimSpace(p.Summary); summary != "" {
		b.WriteString("Summary: ")
		b.WriteString(summary)
		b.WriteString("\n\n")
	}

	b.WriteString("Steps:\n")
	for _, step := range p.Steps {
		fmt.Fprintf(&b, "  %d. %s\n", step.ID, step.Description)
		if len(step.Tools) > 0 {
			fmt.Fprintf(&b, "     Tools: %s\n", strings.Join(step.Tools, ", "))
		}

		files := step.TargetFiles
		if len(files) == 0 {
			files = step.Files
		}
		if len(files) > 0 {
			fmt.Fprintf(&b, "     Files: %s\n", strings.Join(files, ", "))
		}
	}

	b.WriteString("\nProceed with implementation step by step.")
	return b.String()
}
