package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Spinner はターミナルでアニメーションを表示
type Spinner struct {
	mu        sync.Mutex
	active    bool
	stopChan  chan struct{}
	writer    io.Writer
	frames    []string
	startTime time.Time // 開始時刻（経過時間表示用）
	status    string    // 追加のステータスメッセージ
}

// NewSpinner は新しいSpinnerを作成
func NewSpinner() *Spinner {
	return NewSpinnerWithWriter(os.Stdout)
}

// NewSpinnerWithWriter は出力先を指定してSpinnerを作成
// run_test などで stdout 出力（コマンド結果）と混ざらないよう、stderr を指定する用途を想定。
func NewSpinnerWithWriter(w io.Writer) *Spinner {
	if w == nil {
		w = os.Stdout
	}
	return &Spinner{
		writer: w,
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
}

// Start はスピナーアニメーションを開始
func (s *Spinner) Start(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return
	}

	s.active = true
	s.stopChan = make(chan struct{})
	s.startTime = time.Now()

	go s.spin(message)
}

// Stop はスピナーを停止して行をクリア
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}

	s.active = false
	s.status = "" // ステータスもクリア

	// stopChanがnilでないことを確認してからclose（競合対策）
	if s.stopChan != nil {
		close(s.stopChan)
		s.stopChan = nil
	}

	// スピナーの行をクリア
	fmt.Fprintf(s.writer, "\r\033[K")
}

// SetStatus はスピナーに追加のステータスメッセージを設定
// 例: "⠋ Thinking (5s) - Analyzing main.go"
func (s *Spinner) SetStatus(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
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
func (s *Spinner) spin(message string) {
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	// stopChanとstartTimeをローカルに保存（競合対策）
	s.mu.Lock()
	stopChan := s.stopChan
	startTime := s.startTime
	s.mu.Unlock()

	if stopChan == nil {
		return
	}

	i := 0
	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			frame := s.frames[i%len(s.frames)]
			elapsed := formatElapsed(time.Since(startTime))
			status := s.GetStatus()

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
			// 行末の残りをクリア（前の出力が長い場合用）
			output += "\033[K"
			fmt.Fprint(s.writer, output)
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

// グローバルスピナー（どこからでも停止可能）
var globalSpinner *Spinner
var globalSpinnerMu sync.Mutex

// SetGlobalSpinner はグローバルスピナーを設定
func SetGlobalSpinner(s *Spinner) {
	globalSpinnerMu.Lock()
	defer globalSpinnerMu.Unlock()
	// 既存のスピナーを先に停止
	if globalSpinner != nil {
		globalSpinner.Stop()
	}
	globalSpinner = s
}

// StopGlobalSpinner はグローバルスピナーを停止
func StopGlobalSpinner() {
	globalSpinnerMu.Lock()
	defer globalSpinnerMu.Unlock()
	if globalSpinner != nil {
		globalSpinner.Stop()
	}
}
