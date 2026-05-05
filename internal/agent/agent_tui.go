package agent

import (
	"bytes"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tui"
	"github.com/susugadx/xelyon-cli/internal/tui/lifecycle"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

var runTUIProgram = tui.Run
var runTUIProgramWithStartupSubmission = tui.RunWithStartupSubmission
var registerTUIOnExit = lifecycle.OnExit

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
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(bytes.NewReader(nil), &captureBuf, &captureBuf)
	runtime.AutoApprove = autoApprove

	ag := initInteractiveAgentWithRuntime(runtime, model, provider, autoApprove)
	defer ag.Cleanup()

	// SIGTERM 時に Alt Screen を復旧するフックを登録
	ag.exitHook = lifecycle.RestoreTerminal

	if opts.resumeLastSession {
		loadLastSessionForTUI(ag)
	}
	if opts.initialImage != nil {
		green.Fprintf(ag.output(), "🖼️  Image loaded: %s (%s)\n", opts.initialImage.Path, api.FormatImageSize(opts.initialImage.Size))
	}

	// ヘッダー + キャプチャした初期化出力を結合して初期コンテンツにする
	initialContent := buildTUIHeader() + captureBuf.String()

	// ツール結果チャネルを作成し、Agent に設定
	toolResultCh := make(chan tools.ToolResultInfo, 4096)
	ag.tuiToolResultCh = toolResultCh

	// TUIAdapter を作成（sendMsg は後で p.Send 経由で接続）
	adapter := NewTUIAdapter(ag, nil)

	startupSubmission := initialImageStartupSubmission(adapter, opts.initialImageQuery, opts.initialImage)
	if startupSubmission != nil {
		runTUIProgramWithStartupSubmission(adapter, initialContent, startupSubmission, func(p *tea.Program) {
			bindTUIProgram(adapter, ag, toolResultCh, p)
		})
		return
	}
	runTUIProgram(adapter, initialContent, func(p *tea.Program) {
		bindTUIProgram(adapter, ag, toolResultCh, p)
	})
}

// defaultToolCollapsed はツール種別に応じたデフォルトの折りたたみ状態を返す。
func defaultToolCollapsed(toolName, result string, isError bool) bool {
	// エラーは常に展開
	if isError {
		return false
	}

	switch toolName {
	case "apply_patch":
		return false // diff は見たい → 展開
	case "bash":
		return true // 成功は折りたたみ
	case "gather_context", "search_code", "read_file", "read_files":
		return true // 結果が長い → 折りたたみ
	case "web_search":
		return true // 結果が長い → 折りたたみ
	default:
		return true // デフォルトは折りたたみ
	}
}

// buildTUIHeader は TUI 起動時のグラデーションロゴヘッダーを返す。
func buildTUIHeader() string {
	return buildGradientHeader()
}
