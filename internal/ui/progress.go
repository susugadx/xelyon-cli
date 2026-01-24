package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Progress はプログレスバーを表示
type Progress struct {
	mu       sync.Mutex
	total    int
	current  int
	message  string
	active   bool
	stopChan chan struct{}
	writer   io.Writer
	width    int // プログレスバーの幅（文字数）
}

// NewProgress は新しいProgressを作成
// total: 全体の数（0 の場合は不確定モード）
// message: 表示するメッセージ
func NewProgress(total int, message string) *Progress {
	return NewProgressWithWriter(total, message, os.Stdout)
}

// NewProgressWithWriter は出力先を指定してProgressを作成
func NewProgressWithWriter(total int, message string, w io.Writer) *Progress {
	if w == nil {
		w = os.Stdout
	}
	return &Progress{
		total:   total,
		message: message,
		writer:  w,
		width:   20, // デフォルト幅
	}
}

// Start はプログレスバーの表示を開始
func (p *Progress) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.active {
		return
	}

	p.active = true
	p.stopChan = make(chan struct{})

	go p.render()
}

// Stop はプログレスバーを停止して行をクリア
func (p *Progress) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.active {
		return
	}

	p.active = false

	if p.stopChan != nil {
		close(p.stopChan)
		p.stopChan = nil
	}

	// 行をクリア
	fmt.Fprintf(p.writer, "\r\033[K")
}

// Update は現在の進捗を設定
func (p *Progress) Update(current int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = current
}

// Increment は進捗を1増加
func (p *Progress) Increment() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current++
}

// SetMessage はメッセージを更新
func (p *Progress) SetMessage(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.message = message
}

// SetTotal は総数を更新
func (p *Progress) SetTotal(total int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.total = total
}

// GetCurrent は現在の進捗を取得
func (p *Progress) GetCurrent() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.current
}

// render はプログレスバーを描画（goroutine内で実行）
func (p *Progress) render() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	p.mu.Lock()
	stopChan := p.stopChan
	p.mu.Unlock()

	if stopChan == nil {
		return
	}

	// 不確定モード用のアニメーションフレーム
	indeterminateFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frameIdx := 0

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			p.mu.Lock()
			total := p.total
			current := p.current
			message := p.message
			width := p.width
			p.mu.Unlock()

			var output string
			if total <= 0 {
				// 不確定モード: スピナー表示
				frame := indeterminateFrames[frameIdx%len(indeterminateFrames)]
				output = fmt.Sprintf("\r%s %s", frame, message)
				frameIdx++
			} else {
				// 確定モード: プログレスバー表示
				percent := 0
				if total > 0 {
					percent = current * 100 / total
					if percent > 100 {
						percent = 100
					}
				}
				filled := width * percent / 100
				empty := width - filled

				bar := ""
				for i := 0; i < filled; i++ {
					bar += "█"
				}
				for i := 0; i < empty; i++ {
					bar += "░"
				}

				output = fmt.Sprintf("\r[%s] %3d%%  %s", bar, percent, message)
			}

			// 行末の残りをクリア
			output += "\033[K"
			fmt.Fprint(p.writer, output)
		}
	}
}

// Complete はプログレスバーを100%にして停止
func (p *Progress) Complete() {
	p.mu.Lock()
	if p.total > 0 {
		p.current = p.total
	}
	p.mu.Unlock()

	// 最終状態を描画するために少し待つ
	time.Sleep(100 * time.Millisecond)
	p.Stop()
}
