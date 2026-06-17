package agent

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

// RunPlanMode は Plan Mode の調査/承認フェーズを実行
// - 調査ツール(SafetyHigh)は即座に実行
// - 調査結果から計画を生成し、承認を取って終了
// 承認済み plan の通常実装ターンへの引き継ぎは chat request 側が担当する。
func (a *Agent) RunPlanMode(ctx context.Context, userRequest string) error {
	_, err := a.runPlanMode(ctx, userRequest)
	return err
}

func (a *Agent) runPlanMode(ctx context.Context, userRequest string) (*plan.ImplementationHandoff, error) {
	return a.runPlanModeWithAutoCompression(ctx, userRequest, nil)
}

func (a *Agent) runPlanModeWithAutoCompression(ctx context.Context, userRequest string, autoCompression *autoCompressionTurnState) (*plan.ImplementationHandoff, error) {
	req := newPlanModeRequestWithOptions(a, ctx, userRequest, planModeRequestOptions{
		autoCompression: autoCompression,
	})
	if err := req.Run(); err != nil {
		return nil, err
	}
	return req.handoff, nil
}
