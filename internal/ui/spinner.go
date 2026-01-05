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
	mu       sync.Mutex
	active   bool
	stopChan chan struct{}
	writer   io.Writer
	frames   []string
}

// NewSpinner は新しいSpinnerを作成
func NewSpinner() *Spinner {
	return &Spinner{
		writer: os.Stdout,
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
	close(s.stopChan)

	// スピナーの行をクリア
	fmt.Fprintf(s.writer, "\r\033[K")
}

// spin はアニメーションループ（goroutine内で実行）
func (s *Spinner) spin(message string) {
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	i := 0
	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			frame := s.frames[i%len(s.frames)]
			fmt.Fprintf(s.writer, "\r%s %s", frame, message)
			i++
		}
	}
}
