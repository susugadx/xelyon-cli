package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/review"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

// InteractiveStatusSnapshot は interactive surface が表示する現在状態を表す。
type InteractiveStatusSnapshot struct {
	Provider   string
	Model      string
	Mode       string
	Tokens     string
	Cost       string
	LegacyLine string
}

// ReviewRunUsageSummary は /review 1 回分の token/cost summary を表す。
type ReviewRunUsageSummary struct {
	Tokens string
	Cost   string
}

// NewInteractiveAgentWithRuntime は interactive surface 用の Agent を初期化する。
func NewInteractiveAgentWithRuntime(runtime *AgentRuntime, model string, provider api.Provider, autoApprove bool, commandSurface commandcatalog.CommandSurface) *Agent {
	return initInteractiveAgentWithRuntime(runtime, model, provider, autoApprove, commandSurface)
}

// Output は Agent の標準出力 Writer を返す。
func (a *Agent) Output() io.Writer {
	return a.output()
}

// RuntimeUI は Agent が保持する UI runtime を返す。
func (a *Agent) RuntimeUI() *uiruntime.Runtime {
	if a == nil {
		return uiruntime.DefaultRuntime()
	}
	return a.ui()
}

// SetExitHook は Agent 終了前に呼ぶ hook を設定する。
func (a *Agent) SetExitHook(fn func()) {
	if a == nil {
		return
	}
	a.exitHook = fn
}

// StartToolResultStream は構造化ツール結果の event stream を開始する。
func (a *Agent) StartToolResultStream(buffer int) <-chan tools.ToolResultInfo {
	if a == nil {
		return nil
	}
	if buffer < 0 {
		buffer = 0
	}
	ch := make(chan tools.ToolResultInfo, buffer)
	a.tuiToolResultCh = ch
	a.tuiToolResultClosed.Store(false)
	return ch
}

// MarkToolResultStreamClosed は interactive surface 側の tool result stream 終了を記録する。
func (a *Agent) MarkToolResultStreamClosed() {
	if a == nil {
		return
	}
	a.tuiToolResultClosed.Store(true)
}

// ToolResultStreamClosed は tool result stream が終了済みかを返す。
func (a *Agent) ToolResultStreamClosed() bool {
	if a == nil {
		return true
	}
	return a.tuiToolResultClosed.Load()
}

// Chat は interactive session の通常 chat turn を実行する。
func (a *Agent) Chat(input string) error {
	return a.chat(input)
}

// ChatWithImage は interactive session の画像付き chat turn を実行する。
func (a *Agent) ChatWithImage(input string, image *api.ImageData) error {
	return a.chatWithImage(input, image)
}

// ParseImageInput は image: 入力を Agent の出力先へ診断しながら解析する。
func (a *Agent) ParseImageInput(input string) (string, *api.ImageData) {
	return parseImageInputWithWriter(a.output(), input)
}

// HandleCommandForSurface は指定 command surface の slash command を処理する。
func (a *Agent) HandleCommandForSurface(input string, surface commandcatalog.CommandSurface) bool {
	return handleSpecialCommandForSurface(input, a, surface)
}

// CancelActiveRequest は現在の API request をキャンセルする。
func (a *Agent) CancelActiveRequest(reason string) {
	a.cancelActiveRequest(reason)
}

// InteractiveStatusSnapshot は interactive surface 用の構造化状態を返す。
func (a *Agent) InteractiveStatusSnapshot() InteractiveStatusSnapshot {
	if a == nil {
		return InteractiveStatusSnapshot{}
	}
	modeText := planModeStatusText(a.PlanModeEnabled)

	tokens := "0"
	var estimate cost.CostEstimate
	if a.Stats != nil {
		a.statsMu.Lock()
		tokens = FormatTokens(a.Stats.TotalTokens())
		estimate = a.Stats.EstimatedCostEstimateForConfig(a.cfg())
		a.statsMu.Unlock()
	}

	if manager := a.subAgentManager(); manager != nil {
		summary := manager.GetSummary()
		estimate.Cost += summary.TotalCost
		if summary.PricingUnavailable {
			estimate.PricingUnavailable = true
		}
	}

	costText := formatCompactCostEstimate(estimate)
	if strings.EqualFold(a.ProviderName, "ollama") && estimate.Cost == 0 && !estimate.PricingUnavailable {
		costText = ""
	}

	return InteractiveStatusSnapshot{
		Provider:   a.ProviderName,
		Model:      a.CurrentModel,
		Mode:       modeText,
		Tokens:     tokens,
		Cost:       costText,
		LegacyLine: a.FormatStatusLine(),
	}
}

// RunReviewWithProgress は review runner を progress sink 付きで実行し、usage summary を返す。
func (a *Agent) RunReviewWithProgress(ctx context.Context, req review.ReviewRequest, sink review.ReviewProgressSink) (reviewreport.ReviewReport, ReviewRunUsageSummary, error) {
	startStats := a.reviewStatsSnapshot()
	report, err := a.runReview(ctx, req, reviewRunOptions{
		ProgressSink: sink,
	})
	summary := a.reviewRunUsageSummarySince(startStats)
	return report, summary, err
}

// CopyText は指定テキストを clipboard にコピーする。
func CopyText(text string) error {
	return clipboardWriteAll(text)
}

// CopyLastOutput は直近の AI 出力を clipboard にコピーする。
func (a *Agent) CopyLastOutput() (string, error) {
	if a == nil {
		return "", fmt.Errorf("no AI output to copy yet")
	}
	a.historyMu.Lock()
	if len(a.lastOutputs) == 0 {
		a.historyMu.Unlock()
		return "", fmt.Errorf("no AI output to copy yet")
	}
	output := a.lastOutputs[len(a.lastOutputs)-1]
	a.historyMu.Unlock()

	if err := clipboardWriteAll(output); err != nil {
		return "", err
	}
	lines := strings.Count(output, "\n") + 1
	return fmt.Sprintf("Copied %d lines", lines), nil
}

// LoadLastSessionForInteractive は最後の session を読み込み、結果を Agent の出力へ表示する。
func (a *Agent) LoadLastSessionForInteractive() error {
	out := a.output()
	if a.storage == nil {
		red.Fprintln(out, "History storage not available")
		return fmt.Errorf("history storage not available")
	}

	session, err := a.ResumeStartupLastSession(history.ResumeListOptions{})
	if err != nil {
		if errors.Is(err, history.ErrNoResumeSessions) {
			yellow.Fprintln(out, "No previous session found, starting new session")
			return nil
		}
		return err
	}

	green.Fprintf(out, "📂 Resumed session %s (%d messages)\n", session.ID, len(session.ToAPIMessages()))
	return nil
}

// PrintLoadedImage は読み込み済み画像の情報を Agent の出力へ表示する。
func (a *Agent) PrintLoadedImage(image *api.ImageData) {
	if image == nil {
		return
	}
	green.Fprintf(a.output(), "🖼️  Image loaded: %s (%s)\n", image.Path, api.FormatImageSize(image.Size))
}

// SwitchProviderModelWithOutput は provider/model を切り替え、結果を Agent の出力へ表示する。
func (a *Agent) SwitchProviderModelWithOutput(providerName, requestedModel string) error {
	return switchProviderModelWithOutput(a, providerName, requestedModel)
}

// SwitchModelForCurrentProviderWithOutput は current provider の model を切り替え、結果を Agent の出力へ表示する。
func (a *Agent) SwitchModelForCurrentProviderWithOutput(model string) error {
	return switchModelForCurrentProviderWithOutput(a, model)
}

// BuildInteractiveHeader は interactive surface の起動ヘッダーを返す。
func BuildInteractiveHeader() string {
	return buildGradientHeader()
}

func (a *Agent) reviewStatsSnapshot() SessionStats {
	if a == nil || a.Stats == nil {
		return SessionStats{}
	}
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	return *a.Stats
}

func (a *Agent) reviewRunUsageSummarySince(start SessionStats) ReviewRunUsageSummary {
	usage, estimate, ok := a.recordReviewRunUsageSince(start)
	if !ok {
		return ReviewRunUsageSummary{}
	}
	return formatReviewRunUsageSummary(usage, estimate)
}

func (a *Agent) recordReviewRunUsageSince(start SessionStats) (api.Usage, cost.CostEstimate, bool) {
	if a == nil || a.Stats == nil {
		return api.Usage{}, cost.CostEstimate{}, false
	}

	a.statsMu.Lock()
	defer a.statsMu.Unlock()

	turnUsage := a.Stats.UsageDeltaSince(start)
	cfg := a.cfg()
	endEstimate := a.Stats.EstimatedCostEstimateForConfig(cfg)
	startEstimate := start.EstimatedCostEstimateForConfig(cfg)
	costDiff := endEstimate.Cost - startEstimate.Cost
	costUnknown := a.Stats.CostUnknownEvents > start.CostUnknownEvents

	if !turnUsage.HasTokenOrWebSearchObservation() && costDiff == 0 && !costUnknown {
		return api.Usage{}, cost.CostEstimate{}, false
	}

	estimate := cost.CostEstimate{
		Cost:               costDiff,
		PricingUnavailable: costUnknown,
	}
	a.Stats.RecordReviewRunUsage(turnUsage, estimate)
	return turnUsage, estimate, true
}

func formatReviewRunUsageSummary(usage api.Usage, estimate cost.CostEstimate) ReviewRunUsageSummary {
	totalTokens := usage.InputTokens + usage.OutputTokens + usage.ThinkingTokens
	summary := ReviewRunUsageSummary{}
	if totalTokens > 0 {
		summary.Tokens = FormatTokens(totalTokens) + " tok"
	}

	if estimate.PricingUnavailable || estimate.Cost > 0 {
		summary.Cost = formatCompactCostEstimate(estimate)
	}
	return summary
}
