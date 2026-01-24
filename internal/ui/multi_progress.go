package ui

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// TaskStatus はタスクの状態を表す
type TaskStatus string

const (
	TaskPending TaskStatus = "pending"
	TaskRunning TaskStatus = "running"
	TaskDone    TaskStatus = "done"
	TaskError   TaskStatus = "error"
)

// Task は個別のタスク情報を保持
type Task struct {
	ID        int
	Message   string
	Status    TaskStatus
	StartTime time.Time
	EndTime   time.Time
	Error     string
}

// MultiProgress は複数タスクの進捗を管理・表示
type MultiProgress struct {
	mu       sync.Mutex
	tasks    []*Task
	active   bool
	stopCh   chan struct{}
	writer   io.Writer
	lastRows int // 前回描画した行数（クリア用）
}

// NewMultiProgress は新しいMultiProgressを作成
func NewMultiProgress() *MultiProgress {
	return NewMultiProgressWithWriter(os.Stdout)
}

// NewMultiProgressWithWriter は出力先を指定してMultiProgressを作成
func NewMultiProgressWithWriter(w io.Writer) *MultiProgress {
	if w == nil {
		w = os.Stdout
	}
	return &MultiProgress{
		tasks:  make([]*Task, 0),
		writer: w,
	}
}

// AddTask はタスクを追加
func (mp *MultiProgress) AddTask(id int, message string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mp.tasks = append(mp.tasks, &Task{
		ID:      id,
		Message: message,
		Status:  TaskPending,
	})
}

// Start はタスクを実行中にマーク
func (mp *MultiProgress) Start(id int) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	for _, t := range mp.tasks {
		if t.ID == id {
			t.Status = TaskRunning
			t.StartTime = time.Now()
			break
		}
	}
}

// Done はタスクを完了にマーク
func (mp *MultiProgress) Done(id int) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	for _, t := range mp.tasks {
		if t.ID == id {
			t.Status = TaskDone
			t.EndTime = time.Now()
			break
		}
	}
}

// Fail はタスクをエラーにマーク
func (mp *MultiProgress) Fail(id int, err string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	for _, t := range mp.tasks {
		if t.ID == id {
			t.Status = TaskError
			t.EndTime = time.Now()
			t.Error = err
			break
		}
	}
}

// StartRendering はバックグラウンドで定期的に再描画を開始
func (mp *MultiProgress) StartRendering() {
	mp.mu.Lock()
	if mp.active {
		mp.mu.Unlock()
		return
	}
	mp.active = true
	mp.stopCh = make(chan struct{})
	mp.mu.Unlock()

	go mp.renderLoop()
}

// StopRendering は再描画を停止し、最終状態を表示
func (mp *MultiProgress) StopRendering() {
	mp.mu.Lock()
	if !mp.active {
		mp.mu.Unlock()
		return
	}
	mp.active = false
	if mp.stopCh != nil {
		close(mp.stopCh)
		mp.stopCh = nil
	}
	mp.mu.Unlock()

	// 最終状態を描画
	mp.Render()
	fmt.Fprintln(mp.writer) // 改行
}

// renderLoop はバックグラウンドで定期的に再描画
func (mp *MultiProgress) renderLoop() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	mp.mu.Lock()
	stopCh := mp.stopCh
	mp.mu.Unlock()

	if stopCh == nil {
		return
	}

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			mp.Render()
		}
	}
}

// Render は全タスクの状態を表示
func (mp *MultiProgress) Render() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	// 前回の描画をクリア
	mp.clearPreviousOutput()

	// 各タスクを描画
	for _, t := range mp.tasks {
		line := mp.formatTask(t)
		fmt.Fprintln(mp.writer, line)
	}

	mp.lastRows = len(mp.tasks)
}

// clearPreviousOutput は前回の出力をクリア（ANSIエスケープ使用）
func (mp *MultiProgress) clearPreviousOutput() {
	if mp.lastRows > 0 {
		// カーソルを上に移動してクリア
		for i := 0; i < mp.lastRows; i++ {
			fmt.Fprint(mp.writer, "\033[A\033[K") // 1行上に移動してクリア
		}
	}
}

// formatTask はタスクを1行の文字列にフォーマット
func (mp *MultiProgress) formatTask(t *Task) string {
	var icon string
	var elapsed string
	var status string

	switch t.Status {
	case TaskPending:
		icon = "[○]"
		status = "Pending"
	case TaskRunning:
		icon = "[⠋]"
		duration := time.Since(t.StartTime)
		elapsed = fmt.Sprintf("(%.1fs)", duration.Seconds())
		status = "Running"
	case TaskDone:
		icon = "[✓]"
		duration := t.EndTime.Sub(t.StartTime)
		elapsed = fmt.Sprintf("(%.1fs)", duration.Seconds())
		status = "Done"
	case TaskError:
		icon = "[✗]"
		duration := t.EndTime.Sub(t.StartTime)
		elapsed = fmt.Sprintf("(%.1fs)", duration.Seconds())
		status = "Error"
		if t.Error != "" {
			status = fmt.Sprintf("Error: %s", truncateString(t.Error, 30))
		}
	}

	if elapsed != "" {
		return fmt.Sprintf("%s Step %d: %s - %s %s", icon, t.ID, truncateString(t.Message, 40), status, elapsed)
	}
	return fmt.Sprintf("%s Step %d: %s - %s", icon, t.ID, truncateString(t.Message, 40), status)
}

// truncateString は文字列を指定長に切り詰める
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// Clear は全タスクをクリア
func (mp *MultiProgress) Clear() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	mp.clearPreviousOutput()
	mp.tasks = make([]*Task, 0)
	mp.lastRows = 0
}

// GetTask はIDでタスクを取得
func (mp *MultiProgress) GetTask(id int) *Task {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	for _, t := range mp.tasks {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// IsAllDone は全タスクが完了したかチェック
func (mp *MultiProgress) IsAllDone() bool {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	for _, t := range mp.tasks {
		if t.Status == TaskPending || t.Status == TaskRunning {
			return false
		}
	}
	return len(mp.tasks) > 0
}
