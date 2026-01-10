package agent

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

// ParallelExecutor は複数ステップの並列実行を管理
type ParallelExecutor struct {
	agent *Agent
	plan  *Plan
}

// NewParallelExecutor は新しいParallelExecutorを作成
func NewParallelExecutor(agent *Agent, plan *Plan) *ParallelExecutor {
	return &ParallelExecutor{
		agent: agent,
		plan:  plan,
	}
}

// ExecuteSteps は複数のステップを並列実行
func (e *ParallelExecutor) ExecuteSteps(ctx context.Context, stepIDs []int) error {
	if len(stepIDs) == 0 {
		return nil
	}

	// 1つだけの場合は通常実行
	if len(stepIDs) == 1 {
		step := e.plan.GetStep(stepIDs[0])
		if step == nil {
			return fmt.Errorf("step %d not found", stepIDs[0])
		}
		return e.agent.executeStep(ctx, e.plan, step)
	}

	// 複数の場合は並列実行
	cyan.Printf("\n🔀 Executing %d steps in parallel...\n", len(stepIDs))

	// errgroupで並列実行と エラーハンドリング
	g, gctx := errgroup.WithContext(ctx)

	// 結果を保存するためのmutex
	var mu sync.Mutex

	// 各ステップを並列実行
	for _, stepID := range stepIDs {
		stepID := stepID // クロージャー用にコピー
		step := e.plan.GetStep(stepID)
		if step == nil {
			continue
		}

		g.Go(func() error {
			// ステップ実行
			cyan.Printf("[Step %d] Starting: %s\n", step.ID, step.Description)

			// ステータスを実行中に更新（排他制御）
			mu.Lock()
			e.plan.UpdateStatus(step.ID, "running", "")
			mu.Unlock()

			// 実行（agent.executeStepと同じロジック）
			err := e.executeStepParallel(gctx, step)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				e.plan.UpdateStatus(step.ID, "failed", fmt.Sprintf("Error: %v", err))
				red.Printf("[Step %d] Failed: %v\n", step.ID, err)
				return fmt.Errorf("step %d failed: %w", step.ID, err)
			}

			e.plan.UpdateStatus(step.ID, "completed", "Success")
			green.Printf("[Step %d] Completed: %s\n", step.ID, step.Description)
			return nil
		})
	}

	// すべてのステップの完了を待つ
	if err := g.Wait(); err != nil {
		return err
	}

	green.Println("✓ All parallel steps completed successfully!")
	return nil
}

// executeStepParallel は単一ステップを並列実行用に実行
// agent.executeStepと同じロジックだが、並列実行を考慮
func (e *ParallelExecutor) executeStepParallel(ctx context.Context, step *PlanStep) error {
	// TODO: agent.executeStepと同じロジックを実装
	// 現在は簡易実装（実際のツール実行は後で統合）
	return nil
}
