package agent

import (
	"bytes"
	"strings"
	"sync"
	"sync/atomic"

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
	return handleSpecialCommand(cmd, a.agent)
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

// tuiCaptureWriter は agent の出力をキャプチャし TUI に送信する io.Writer
type tuiCaptureWriter struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	sendFn func(string)
}

func newTUICaptureWriter(sendFn func(string)) *tuiCaptureWriter {
	return &tuiCaptureWriter{sendFn: sendFn}
}

// Write は書き込まれた内容を改行区切りでフラッシュする。
func (w *tuiCaptureWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n, err := w.buf.Write(p)

	// 改行を含む場合、行ごとにフラッシュ
	for {
		line, readErr := w.buf.ReadString('\n')
		if readErr != nil {
			// 改行が見つからなかった → 残りをバッファに戻す
			if line != "" {
				w.buf.WriteString(line)
			}
			break
		}
		if w.sendFn != nil {
			w.sendFn(strings.TrimRight(line, "\n"))
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
