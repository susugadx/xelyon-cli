package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// chat はAIと対話する
// PlanModeEnabled に応じて Plan Mode または通常モードで処理
func (a *Agent) chat(input string) error {
	return a.chatInternal(input, nil)
}

// ChatOnce は単一クエリを1ターンだけ実行してエラーを返す（--once 用）
// stdin を読まず、REPL ループに入らない。chatCore を共有する。
func (a *Agent) ChatOnce(input string) error {
	return a.chatCore(input, nil, true)
}

// ChatOnceWithImage は画像付きの単一クエリを1ターンだけ実行してエラーを返す。
func (a *Agent) ChatOnceWithImage(input string, image *api.ImageData) error {
	return a.chatCore(input, image, true)
}

// chatInternal はAIと対話する内部実装（対話モード用ラッパー）
// image が nil でない場合は画像付きメッセージとして処理
func (a *Agent) chatInternal(input string, image *api.ImageData) error {
	if err := a.chatCore(input, image, false); err != nil {
		return err
	}
	return a.chatErrorFromStatus()
}

func (a *Agent) chatErrorFromStatus() error {
	status := a.statusRef().getStatus()
	if status.State != StateAborted {
		return nil
	}
	if reason := strings.TrimSpace(status.ReasonEN); reason != "" {
		return errors.New(reason)
	}
	return errors.New("request aborted")
}

// runNormalMode は通常モードでの処理（Plan Mode OFF 時）を実行する。
func (a *Agent) runNormalMode(ctx context.Context, input string, image *api.ImageData) error {
	return newTurnRunner(a, ctx).RunNormalMode(input, image)
}

// showTaskSummary は現在タスクの changeStack からサマリーを生成して表示
// taskChangeOffset 以降の変更のみを対象とする（Issue #118: タスク分離）
func (a *Agent) showTaskSummary() {
	taskChanges := a.changeStack[a.taskChangeOffset:]
	if len(taskChanges) == 0 {
		return
	}

	ts := ui.NewTaskSummary()
	for _, change := range taskChanges {
		if len(change.Details) > 0 {
			for _, detail := range change.Details {
				action := detail.Action
				if action == "" {
					action = ui.InferAction(change.Tool)
				}
				ts.AddChange(detail.FilePath, action, detail.LinesAdded, detail.LinesRemoved)
			}
			continue
		}
		action := ui.InferAction(change.Tool)
		ts.AddChange(change.FilePath, action, change.LinesAdded, change.LinesRemoved)
	}
	if a.taskTestResult != nil {
		ts.SetTestResult(*a.taskTestResult)
	}
	if a.taskTestCommand != "" {
		ts.SetTestCommand(a.taskTestCommand)
	}

	_, _ = fmt.Fprint(a.output(), ts.Render())
}

// chatWithImage は画像付きメッセージでAIと対話する
func (a *Agent) chatWithImage(input string, image *api.ImageData) error {
	// プロバイダーが画像対応かチェック
	if !a.CurrentProvider.SupportsImages() {
		yellow.Fprintf(a.output(), "Warning: %s does not support images. The image will be ignored.\n", a.CurrentProvider.Name())
		return a.chat(input)
	}

	// 画像情報をログ
	green.Fprintf(a.output(), "🖼️  Sending image: %s (%s)\n", image.Path, api.FormatImageSize(image.Size))

	return a.chatInternal(input, image)
}

// printTaskUsage はタスク全体の usage を表示
func (a *Agent) printTaskUsage(startStats SessionStats) {
	a.statsMu.Lock()
	if a.Stats == nil {
		a.statsMu.Unlock()
		return
	}

	turnUsage := a.Stats.UsageDeltaSince(startStats)
	cfg := a.cfg()
	endEstimate := a.Stats.EstimatedCostEstimateForConfig(cfg)
	startEstimate := startStats.EstimatedCostEstimateForConfig(cfg)
	turnEstimate := cost.EstimateRequestCostWithCacheForConfig(cfg, a.Stats.Provider, a.Stats.Model, turnUsage)
	costDiff := endEstimate.Cost - startEstimate.Cost
	costUnknown := a.Stats.CostUnknownEvents > startStats.CostUnknownEvents ||
		(turnUsage.HasTokenOrWebSearchObservation() && turnEstimate.PricingUnavailable)
	turnCostEstimate := cost.CostEstimate{
		Cost:               costDiff,
		PricingUnavailable: costUnknown,
	}
	a.Stats.LastTurnUsage = &turnUsage
	a.Stats.LastTurnCost = costDiff
	a.Stats.LastTurnCostUnknown = costUnknown
	inDiff := turnUsage.InputTokens
	outDiff := turnUsage.OutputTokens
	a.statsMu.Unlock()

	total := inDiff + outDiff
	if total == 0 {
		return
	}

	// ✓ を緑色で表示、残りはdimまたは通常色
	green.Fprint(a.output(), "✓ ")
	if shouldSuppressLocalCostDisplay(a.ProviderName, turnCostEstimate) {
		_, _ = fmt.Fprintf(a.output(), "In: %s + Out: %s = %s tok\n",
			FormatNumber(inDiff),
			FormatNumber(outDiff),
			FormatNumber(total))
	} else {
		_, _ = fmt.Fprintf(a.output(), "In: %s + Out: %s = %s tok ",
			FormatNumber(inDiff),
			FormatNumber(outDiff),
			FormatNumber(total))
		if costUnknown {
			dim.Fprint(a.output(), "(cost N/A)\n")
		} else {
			dim.Fprintf(a.output(), "(~$%.4f)\n", costDiff)
		}
	}
}
