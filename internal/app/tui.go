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
var runTUIProgramWithStartupOptions = tui.RunWithStartupOptions
var newTUIAdapter = tuiagent.NewTUIAdapter
var cleanupTUIAgent = func(ag *agent.Agent) {
	ag.Cleanup()
}

type tuiRunOptions struct {
	resumeLastSession bool
	resumeSessionID   string
	resumePicker      bool
	resumeAllSessions bool
	initialImageQuery string
	initialImage      *api.ImageData
}

// RunTUIWithConfig は TUI モードでインタラクティブセッションを起動する。
func RunTUIWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	_ = runTUIWithOptions(model, provider, cfg, autoApprove, tuiRunOptions{})
}

// RunTUIWithResumeWithConfig は前回セッションを再開して TUI モードを起動する。
func RunTUIWithResumeWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) error {
	return runTUIWithOptions(model, provider, cfg, autoApprove, tuiRunOptions{resumeLastSession: true})
}

// RunTUIWithResumePickerWithConfig は session picker を開いた状態で TUI モードを起動する。
func RunTUIWithResumePickerWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool, all bool) {
	_ = runTUIWithOptions(model, provider, cfg, autoApprove, tuiRunOptions{
		resumePicker:      true,
		resumeAllSessions: all,
	})
}

// RunTUIWithResumeSessionWithConfig は指定 session を再開して TUI モードを起動する。
func RunTUIWithResumeSessionWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool, sessionID string) error {
	return runTUIWithOptions(model, provider, cfg, autoApprove, tuiRunOptions{resumeSessionID: sessionID})
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

	return runTUIWithOptions(model, provider, cfg, autoApprove, tuiRunOptions{
		initialImageQuery: query,
		initialImage:      image,
	})
}

func runTUIWithOptions(model string, provider api.Provider, cfg *config.Config, autoApprove bool, opts tuiRunOptions) error {
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
		if err := ag.LoadLastSessionForInteractive(); err != nil {
			return fmt.Errorf("failed to resume session: %w", err)
		}
	}
	if opts.resumeSessionID != "" {
		if _, err := ag.ResumeStartupSession(opts.resumeSessionID); err != nil {
			return fmt.Errorf("failed to resume session: %w", err)
		}
		fmt.Fprintf(runtime.UI.Output(), "Resumed session %s\n", opts.resumeSessionID)
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
		return nil
	}
	if opts.resumePicker {
		runTUIProgramWithStartupOptions(adapter, initialContent, tui.StartupOptions{
			SessionPicker: &tui.StartupSessionPicker{All: opts.resumeAllSessions},
		}, func(p *tea.Program) {
			tuiagent.BindTUIProgram(adapter, ag, toolResultCh, p)
		})
		return nil
	}
	runTUIProgram(adapter, initialContent, func(p *tea.Program) {
		tuiagent.BindTUIProgram(adapter, ag, toolResultCh, p)
	})
	return nil
}

// buildTUIHeader は TUI 起動時のグラデーションロゴヘッダーを返す。
func buildTUIHeader() string {
	return agent.BuildInteractiveHeader()
}
