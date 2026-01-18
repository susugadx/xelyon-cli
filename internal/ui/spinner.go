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

	// stopChanがnilでないことを確認してからclose（競合対策）
	if s.stopChan != nil {
		close(s.stopChan)
		s.stopChan = nil
	}

	// スピナーの行をクリア
	fmt.Fprintf(s.writer, "\r\033[K")
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
			if elapsed != "" {
				fmt.Fprintf(s.writer, "\r%s %s %s", frame, message, elapsed)
			} else {
				fmt.Fprintf(s.writer, "\r%s %s", frame, message)
			}
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
