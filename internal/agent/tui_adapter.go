package agent

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/commandruntime"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui"
)

// TUIAdapter は Agent を tui.AgentInterface に適合させるアダプタ
type TUIAdapter struct {
	agent         *Agent
	sendMsg       func(tui.AppendMessageMsg)
	captureWriter *tuiCaptureWriter
	processing    atomic.Bool
}

// NewTUIAdapter は TUIAdapter を作成する。
func NewTUIAdapter(agent *Agent, sendMsg func(tui.AppendMessageMsg)) *TUIAdapter {
	return &TUIAdapter{
		agent:   agent,
		sendMsg: sendMsg,
	}
}

// SetOutputCapture は agent の出力先を TUI キャプチャ用 Writer に差し替える。
func (a *TUIAdapter) SetOutputCapture() {
	capture := newTUICaptureWriter(func(text string) {
		if a.sendMsg != nil {
			a.sendMsg(tui.AppendMessageMsg{
				Message: tui.ChatMessage{
					Role:    "assistant",
					Content: text,
				},
			})
		}
	})
	a.captureWriter = capture
	runtime := a.agent.ui()
	runtime.SetOutput(capture)
	runtime.SetErrorOutput(capture)
}

// Chat はユーザー入力をAIに送信する。goroutine で呼ぶこと。
func (a *TUIAdapter) Chat(input string) {
	a.processing.Store(true)
	defer a.processing.Store(false)

	// 画像入力チェック
	if strings.Contains(input, "image:") {
		textPart, image := parseImageInputWithWriter(a.agent.output(), input)
		if image != nil {
			a.agent.chatWithImage(textPart, image)
			if a.captureWriter != nil {
				a.captureWriter.Flush()
			}
			return
		}
	}

	a.agent.chat(input)
	if a.captureWriter != nil {
		a.captureWriter.Flush()
	}
}

// HandleCommand は特殊コマンドを処理する。処理した場合 true を返す。
func (a *TUIAdapter) HandleCommand(cmd string) bool {
	invocation, ok := commandruntime.Parse(cmd, commandAliasesFromConfig(a.agent.cfg()))
	if !ok {
		return false
	}
	baseCmd := invocation.Command
	args := invocation.Args

	if cmdInfo, known := commandcatalog.Find(baseCmd); known && !cmdInfo.SupportsSurface(commandcatalog.CommandSurfaceTUI) {
		_, _ = fmt.Fprintf(a.agent.output(), "⚠️  %s is not available in TUI mode.\n", baseCmd)
		return true
	}

	if baseCmd == "/project" {
		return false
	}

	if baseCmd == "/config" {
		if !isNonInteractiveConfigSubcommand(args) {
			_, _ = fmt.Fprintf(a.agent.output(), "⚠️  %s is not available in TUI mode.\n   Use bare /config, /config show, or /config model <name>.\n", cmd)
			return true
		}
		return handleSpecialCommandForSurface(cmd, a.agent, commandcatalog.CommandSurfaceTUI)
	}

	return handleSpecialCommandForSurface(cmd, a.agent, commandcatalog.CommandSurfaceTUI)
}

// GetStatusLine はステータスバーに表示する文字列を返す。
func (a *TUIAdapter) GetStatusLine() string {
	return a.agent.FormatStatusLine()
}

// Cancel は現在のAPI呼び出しをキャンセルする。
func (a *TUIAdapter) Cancel() {
	a.agent.cancelActiveRequest("user cancelled")
}

// Cleanup は終了処理を行う。
func (a *TUIAdapter) Cleanup() {
	a.agent.Cleanup()
}

// IsProcessing はAI処理中かどうかを返す。
func (a *TUIAdapter) IsProcessing() bool {
	return a.processing.Load()
}

// CopyText は指定テキストをクリップボードにコピーする。
func (a *TUIAdapter) CopyText(text string) error {
	return clipboardWriteAll(text)
}

// LoadConfigForEdit は設定ファイルを読み込み、編集用のクローンを返す。
func (a *TUIAdapter) LoadConfigForEdit() (*config.Config, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}
	return config.CloneConfig(cfg), nil
}

// SaveAndSyncConfig は設定をファイルに保存し、runtime に反映する。
func (a *TUIAdapter) SaveAndSyncConfig(cfg *config.Config) error {
	return a.agent.SaveAndSyncConfig(cfg)
}

// LoadProjectForEdit は xelyon.yaml を読み込み、編集用のクローンを返す。
func (a *TUIAdapter) LoadProjectForEdit() (*config.ProjectConfig, error) {
	pc, err := config.LoadProjectConfigWithError()
	if err != nil {
		return nil, err
	}
	return config.CloneProjectConfig(pc), nil
}

// SaveProjectConfig は xelyon.yaml を保存する。
func (a *TUIAdapter) SaveProjectConfig(pc *config.ProjectConfig) error {
	return a.agent.SaveAndSyncProjectConfig(pc)
}

// CreateProjectConfigTemplate は xelyon.yaml template を作成し、編集用に読み込む。
func (a *TUIAdapter) CreateProjectConfigTemplate() (*config.ProjectConfig, error) {
	if err := config.CreateProjectConfigTemplate("", false); err != nil {
		return nil, err
	}
	pc, err := config.LoadProjectConfigWithError()
	if err != nil {
		return nil, err
	}
	if pc == nil {
		return nil, fmt.Errorf("created xelyon.yaml but failed to load it")
	}
	return config.CloneProjectConfig(pc), nil
}

// GetProviderName は現在のプロバイダー名を返す。
func (a *TUIAdapter) GetProviderName() string {
	return a.agent.ProviderName
}

// GetProviderConfigKey は現在セッションが代表する provider_models key を返す。
func (a *TUIAdapter) GetProviderConfigKey() string {
	return a.agent.GetProviderConfigKey()
}

// ResolveAlias はコマンド名を alias 解決する。
func (a *TUIAdapter) ResolveAlias(cmd string) string {
	return resolveCommandAliasWithConfig(cmd, a.agent.cfg())
}

// CopyLastOutput は直近のAI出力をクリップボードにコピーする。
// historyMu でロックし、chat goroutine との data race を防ぐ。
func (a *TUIAdapter) CopyLastOutput() (string, error) {
	a.agent.historyMu.Lock()
	if len(a.agent.lastOutputs) == 0 {
		a.agent.historyMu.Unlock()
		return "", fmt.Errorf("no AI output to copy yet")
	}
	output := a.agent.lastOutputs[len(a.agent.lastOutputs)-1]
	a.agent.historyMu.Unlock()

	if err := clipboardWriteAll(output); err != nil {
		return "", err
	}
	lines := strings.Count(output, "\n") + 1
	return fmt.Sprintf("Copied %d lines", lines), nil
}

// tuiCaptureWriter は agent の出力をキャプチャし TUI に送信する io.Writer
type tuiCaptureWriter struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	sendFn func(string)
}

func newTUICaptureWriter(sendFn func(string)) *tuiCaptureWriter {
	return &tuiCaptureWriter{sendFn: sendFn}
}

// Write は書き込まれた内容をバッファに追加し、改行区切りでバッチフラッシュする。
// \r を含む書き込みはスピナー等の行上書きアニメーションなのでドロップする。
// 複数行が一度に届いた場合は1回の sendFn 呼び出しにまとめて Update+View サイクルを削減する。
func (w *tuiCaptureWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// \r を含む書き込みはスピナー → ドロップ
	if bytes.Contains(p, []byte("\r")) {
		return len(p), nil
	}

	n, err := w.buf.Write(p)

	// 改行を含む場合、全行をまとめて1回の sendFn で送る
	if bytes.Contains(p, []byte("\n")) {
		var batch strings.Builder
		for {
			line, readErr := w.buf.ReadString('\n')
			if readErr != nil {
				if line != "" {
					w.buf.WriteString(line)
				}
				break
			}
			if batch.Len() > 0 {
				batch.WriteByte('\n')
			}
			batch.WriteString(strings.TrimRight(line, "\n"))
		}
		if batch.Len() > 0 && w.sendFn != nil {
			w.sendFn(batch.String())
		}
	}

	return n, err
}

// Flush はバッファに残っている内容をフラッシュする。
func (w *tuiCaptureWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.buf.Len() > 0 {
		if w.sendFn != nil {
			w.sendFn(w.buf.String())
		}
		w.buf.Reset()
	}
}
