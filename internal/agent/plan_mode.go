package agent

import (
	"context"
)

// RunPlanMode は Claude Code 風の Plan Mode を実行
// - 調査ツール(SafetyHigh)は即座に実行
// - 調査結果から計画を生成し、承認を取って通常モードへ handoff して終了
func (a *Agent) RunPlanMode(ctx context.Context, userRequest string) error {
	return newPlanModeRequest(a, ctx, userRequest).Run()
}
