package app

import (
	"bytes"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/agent"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui"
	"github.com/susugadx/xelyon-cli/internal/tui/lifecycle"
	"github.com/susugadx/xelyon-cli/internal/tuiagent"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

var runTUIProgram = tui.Run
var runTUIProgramWithStartupSubmission = tui.RunWithStartupSubmission
var newTUIAdapter = tuiagent.NewTUIAdapter
var cleanupTUIAgent = func(ag *agent.Agent) {
	ag.Cleanup()
}

type tuiRunOptions struct {
	resumeLastSession bool
	initialImageQuery string
	initialImage      *api.ImageData
}

// RunTUIWithConfig は TUI モードでインタラクティブセッションを起動する。
func RunTUIWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	runTUIWithOptions(model, provider, cfg, autoApprove, tuiRunOptions{})
}

// RunTUIWithResumeWithConfig は前回セッションを再開して TUI モードを起動する。
func RunTUIWithResumeWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	runTUIWithOptions(model, provider, cfg, autoApprove, tuiRunOptions{resumeLastSession: true})
}

// RunTUIWithImageWithConfig は画像付きの初回ターンを実行して TUI モードを起動する。
func RunTUIWithImageWithConfig(query string, model string, provider api.Provider, imagePath string, cfg *config.Config, autoApprove bool) error {
	if !provider.SupportsImages() {
		return fmt.Errorf("provider %q does not support image input", provider.Name())
	}

	image, err := api.LoadImage(imagePath)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}

	if query == "" {
		query = "Please analyze this image."
	}

	runTUIWithOptions(model, provider, cfg, autoApprove, tuiRunOptions{
		initialImageQuery: query,
		initialImage:      image,
	})
	return nil
}

func runTUIWithOptions(model string, provider api.Provider, cfg *config.Config, autoApprove bool, opts tuiRunOptions) {
	// Agent 初期化中の stdout 出力をキャプチャするバッファ。
	// Normal Screen に何も残さず、全てを Alt Screen の viewport に表示する。
	var captureBuf bytes.Buffer
	runtime := agent.NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(bytes.NewReader(nil), &captureBuf, &captureBuf)
	runtime.AutoApprove = autoApprove

	ag := agent.NewInteractiveAgentWithRuntime(runtime, model, provider, autoApprove, commandcatalog.CommandSurfaceTUI)
	defer cleanupTUIAgent(ag)

	// SIGTERM 時に Alt Screen を復旧するフックを登録
	ag.SetExitHook(lifecycle.RestoreTerminal)

	if opts.resumeLastSession {
		ag.LoadLastSessionForInteractive()
	}
	if opts.initialImage != nil {
		ag.PrintLoadedImage(opts.initialImage)
	}

	// ヘッダー + キャプチャした初期化出力を結合して初期コンテンツにする
	initialContent := buildTUIHeader() + captureBuf.String()

	// ツール結果チャネルを作成し、Agent に設定
	toolResultCh := ag.StartToolResultStream(4096)

	// TUIAdapter を作成（sendMsg は後で p.Send 経由で接続）
	adapter := newTUIAdapter(ag, nil)

	startupSubmission := tuiagent.InitialImageStartupSubmission(adapter, opts.initialImageQuery, opts.initialImage)
	if startupSubmission != nil {
		runTUIProgramWithStartupSubmission(adapter, initialContent, startupSubmission, func(p *tea.Program) {
			tuiagent.BindTUIProgram(adapter, ag, toolResultCh, p)
		})
		return
	}
	runTUIProgram(adapter, initialContent, func(p *tea.Program) {
		tuiagent.BindTUIProgram(adapter, ag, toolResultCh, p)
	})
}

// buildTUIHeader は TUI 起動時のグラデーションロゴヘッダーを返す。
func buildTUIHeader() string {
	return agent.BuildInteractiveHeader()
}
