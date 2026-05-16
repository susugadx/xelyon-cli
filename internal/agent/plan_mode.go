package agent

import (
	"context"
)

// RunPlanMode は Plan Mode の調査/承認フェーズを実行
// - 調査ツール(SafetyHigh)は即座に実行
// - 調査結果から計画を生成し、承認を取って終了
// 承認済み plan の通常実装ターンへの引き継ぎは chat request 側が担当する。
func (a *Agent) RunPlanMode(ctx context.Context, userRequest string) error {
	_, err := a.runPlanMode(ctx, userRequest)
	return err
}

func (a *Agent) runPlanMode(ctx context.Context, userRequest string) (*planModeImplementationHandoff, error) {
	req := newPlanModeRequest(a, ctx, userRequest)
	if err := req.Run(); err != nil {
		return nil, err
	}
	return req.handoff, nil
}
