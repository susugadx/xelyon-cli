package ui

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Spinner はターミナルでアニメーションを表示
type Spinner struct {
	mu        sync.Mutex
	active    bool
	animated  bool
	stopChan  chan struct{}
	doneChan  chan struct{} // goroutine終了通知（Stop()の同期待機用）
	writer    io.Writer
	frames    []string
	startTime time.Time // 開始時刻（経過時間表示用）
	message   string
	status    string // 追加のステータスメッセージ
}

// NewSpinner は新しいSpinnerを作成
func NewSpinner() *Spinner {
	return NewSpinnerWithRuntime(DefaultRuntime())
}

// NewSpinnerWithRuntime は UI runtime に紐づく出力先で Spinner を作成する。
func NewSpinnerWithRuntime(runtime *Runtime) *Spinner {
	return NewSpinnerWithWriter(runtimeOrDefault(runtime).Output())
}

// NewSpinnerWithWriter は出力先を指定してSpinnerを作成
// run_test などで stdout 出力（コマンド結果）と混ざらないよう、stderr を指定する用途を想定。
func NewSpinnerWithWriter(w io.Writer) *Spinner {
	if w == nil {
		w = DefaultRuntime().Output()
	}
	return &Spinner{
		writer: w,
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
}

// Start はスピナーアニメーションを開始
func (s *Spinner) Start(message string) {
	s.mu.Lock()

	if s.active {
		s.mu.Unlock()
		return
	}

	s.active = true
	s.startTime = time.Now()
	s.message = message
	s.status = ""
	s.animated = shouldAnimateSpinner(s.writer)

	if !s.animated {
		writer := s.writer
		s.mu.Unlock()
		if message != "" {
			fmt.Fprintln(writer, message)
		}
		return
	}

	stopChan := make(chan struct{})
	doneChan := make(chan struct{})
	startTime := s.startTime
	s.stopChan = stopChan
	s.doneChan = doneChan
	s.mu.Unlock()

	go s.spin(message, stopChan, doneChan, startTime)
}

// IsActive はスピナーが動作中かどうかを返す
func (s *Spinner) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

func (s *Spinner) shouldDetachFromRuntime() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.active && !s.startTime.IsZero()
}

// Stop はスピナーを停止して行をクリア
// goroutineの終了を同期的に待機してからクリアするため、
// Stop()後は確実にターミナル書き込みが完了している
func (s *Spinner) Stop() {
	s.mu.Lock()

	if !s.active {
		s.mu.Unlock()
		return
	}

	s.active = false
	s.status = "" // ステータスもクリア
	s.message = ""

	doneChan := s.doneChan
	animated := s.animated

	// stopChanがnilでないことを確認してからclose（競合対策）
	if animated && s.stopChan != nil {
		close(s.stopChan)
		s.stopChan = nil
	}
	s.doneChan = nil

	// ロックを解放してからgoroutine終了を待機
	// （spin goroutineがGetStatus()でロックを取るためデッドロック防止）
	s.mu.Unlock()

	if !animated {
		return
	}

	// goroutineの終了を待機（最大200msのタイムアウト付き）
	if doneChan != nil {
		select {
		case <-doneChan:
		case <-time.After(200 * time.Millisecond):
		}
	}

	// goroutine停止後にスピナーの行をクリア
	fmt.Fprintf(s.writer, "\r\033[K")
}

// SetStatus はスピナーに追加のステータスメッセージを設定
// 例: "⠋ Thinking (5s) - Analyzing main.go"
func (s *Spinner) SetStatus(status string) {
	s.mu.Lock()
	prevStatus := s.status
	s.status = status
	active := s.active
	animated := s.animated
	message := s.message
	writer := s.writer
	s.mu.Unlock()

	if !active || animated || status == "" || status == prevStatus {
		return
	}

	if message == "" {
		fmt.Fprintln(writer, status)
		return
	}
	fmt.Fprintf(writer, "%s - %s\n", message, status)
}

// ClearStatus はステータスメッセージをクリア
func (s *Spinner) ClearStatus() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = ""
}

// GetStatus は現在のステータスメッセージを取得
func (s *Spinner) GetStatus() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// spin はアニメーションループ（goroutine内で実行）
func (s *Spinner) spin(message string, stopChan <-chan struct{}, doneChan chan<- struct{}, startTime time.Time) {
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	// goroutine終了をStop()に通知
	defer func() {
		if doneChan != nil {
			close(doneChan)
		}
	}()

	if stopChan == nil {
		return
	}

	i := 0
	colors := []int{27, 33, 39, 45, 51} // XELYON ブランドカラー
	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			frame := s.frames[i%len(s.frames)]
			elapsed := formatElapsed(time.Since(startTime))
			status := s.GetStatus()
			colorCode := colors[i%len(colors)]

			// 出力フォーマット: "⠋ Message (Ns) - Status"
			var output string
			if elapsed != "" {
				output = fmt.Sprintf("\r%s %s %s", frame, message, elapsed)
			} else {
				output = fmt.Sprintf("\r%s %s", frame, message)
			}
			if status != "" {
				output += fmt.Sprintf(" - %s", status)
			}

			// 色を適用 (ANSI 256色)
			coloredOutput := fmt.Sprintf("\033[38;5;%dm%s\033[0m", colorCode, output)

			// 行末の残りをクリア
			fmt.Fprint(s.writer, coloredOutput+"\033[K")
			i++
		}
	}
}

// formatElapsed は経過時間を読みやすい形式でフォーマット
func formatElapsed(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds < 1 {
		return ""
	}
	return fmt.Sprintf("(%ds)", seconds)
}

func shouldAnimateSpinner(w io.Writer) bool {
	return isFileTerminal(w)
}

// ResetTerminalState はターミナル状態をリセットする
// FC→テキストモードフォールバック時など、ターミナル状態が不整合になった場合に使用
func ResetTerminalState() {
	DefaultRuntime().ResetTerminalState()
}

// SpinnerMessageForTool はツール名に応じたスピナーメッセージを返します
func SpinnerMessageForTool(toolName string) string {
	switch toolName {
	case "write_file":
		return "Writing file..."
	case "str_replace":
		return "Editing file..."
	default:
		return "Preparing..."
	}
}
