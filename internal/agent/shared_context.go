package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// SharedContext は Worker 間で共有するコンテキスト
// すべてのメソッドはスレッドセーフ
type SharedContext struct {
	mu sync.RWMutex

	// 調査結果
	investigationResults map[string]string // query -> result

	// ステップ結果
	stepResults map[int]string // stepID -> result

	// ファイル変更
	fileChanges map[int][]tools.FileChange // stepID -> changes

	// 完了ステップ（依存関係解決用）
	completedSteps map[int]bool

	// エラーログ
	errors []WorkerError

	// Pub/Sub 用
	messages    map[string][]WorkerMessage      // topic -> messages（履歴保存）
	subscribers map[string][]chan WorkerMessage // topic -> channels
	subMu       sync.RWMutex                    // subscribers 用の別ロック
}

// WorkerMessage は Worker 間のメッセージ
type WorkerMessage struct {
	FromWorker int
	Topic      string // "step_completed", "step_failed", "file_changed", "escalation"
	Content    string
	StepID     int
	FilePath   string
	Timestamp  time.Time
}

// WorkerError は Worker エラー
type WorkerError struct {
	WorkerID int
	StepID   int
	Error    error
}

// NewSharedContext は新しい SharedContext を作成
func NewSharedContext() *SharedContext {
	return &SharedContext{
		investigationResults: make(map[string]string),
		stepResults:          make(map[int]string),
		fileChanges:          make(map[int][]tools.FileChange),
		completedSteps:       make(map[int]bool),
		errors:               []WorkerError{},
		messages:             make(map[string][]WorkerMessage),
		subscribers:          make(map[string][]chan WorkerMessage),
	}
}

// AddInvestigationResult は調査結果を追加
func (sc *SharedContext) AddInvestigationResult(query, result string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.investigationResults[query] = result
}

// GetInvestigationResult は調査結果を取得
func (sc *SharedContext) GetInvestigationResult(query string) (string, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	result, ok := sc.investigationResults[query]
	return result, ok
}

// GetAllInvestigationResults は全ての調査結果を取得
func (sc *SharedContext) GetAllInvestigationResults() map[string]string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	// コピーを返す
	results := make(map[string]string, len(sc.investigationResults))
	for k, v := range sc.investigationResults {
		results[k] = v
	}
	return results
}

// AddStepResult はステップ結果を追加
func (sc *SharedContext) AddStepResult(stepID int, result string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.stepResults[stepID] = result
}

// GetStepResult はステップ結果を取得
func (sc *SharedContext) GetStepResult(stepID int) (string, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	result, ok := sc.stepResults[stepID]
	return result, ok
}

// AddFileChange はファイル変更を記録
func (sc *SharedContext) AddFileChange(stepID int, change tools.FileChange) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.fileChanges[stepID] = append(sc.fileChanges[stepID], change)
}

// GetFileChanges はステップのファイル変更を取得
func (sc *SharedContext) GetFileChanges(stepID int) []tools.FileChange {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	// コピーを返す
	changes := make([]tools.FileChange, len(sc.fileChanges[stepID]))
	copy(changes, sc.fileChanges[stepID])
	return changes
}

// GetAllFileChanges は全てのファイル変更を取得
func (sc *SharedContext) GetAllFileChanges() []tools.FileChange {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	var all []tools.FileChange
	for _, changes := range sc.fileChanges {
		all = append(all, changes...)
	}
	return all
}

// MarkStepCompleted はステップを完了としてマーク
func (sc *SharedContext) MarkStepCompleted(stepID int) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.completedSteps[stepID] = true
}

// IsStepCompleted はステップが完了しているかチェック
func (sc *SharedContext) IsStepCompleted(stepID int) bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.completedSteps[stepID]
}

// GetCompletedStepIDs は完了したステップ ID を取得
func (sc *SharedContext) GetCompletedStepIDs() []int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	ids := make([]int, 0, len(sc.completedSteps))
	for id := range sc.completedSteps {
		ids = append(ids, id)
	}
	return ids
}

// GetContextForStep はステップ実行に必要なコンテキストを取得
// 依存ステップの結果を含める
func (sc *SharedContext) GetContextForStep(step *plan.PlanStep) string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	var context strings.Builder

	// 依存ステップの結果を含める
	for _, depID := range step.DependsOn {
		if result, ok := sc.stepResults[depID]; ok {
			context.WriteString(fmt.Sprintf("=== Step %d Result ===\n%s\n\n", depID, result))
		}
	}

	return context.String()
}

// AddError はエラーを追加
func (sc *SharedContext) AddError(err WorkerError) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.errors = append(sc.errors, err)
}

// GetErrors は全てのエラーを取得
func (sc *SharedContext) GetErrors() []WorkerError {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	// コピーを返す
	errors := make([]WorkerError, len(sc.errors))
	copy(errors, sc.errors)
	return errors
}

// GetMessages はトピックの過去メッセージを取得
func (sc *SharedContext) GetMessages(topic string) []WorkerMessage {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	msgs := sc.messages[topic]
	copied := make([]WorkerMessage, len(msgs))
	copy(copied, msgs)
	return copied
}

// Subscribe はトピックを購読し、メッセージを受け取るチャンネルを返す
// バッファサイズは maxWorkers * 100
func (sc *SharedContext) Subscribe(topic string, bufferSize int) <-chan WorkerMessage {
	if bufferSize <= 0 {
		bufferSize = 1
	}
	ch := make(chan WorkerMessage, bufferSize)

	sc.subMu.Lock()
	sc.subscribers[topic] = append(sc.subscribers[topic], ch)
	sc.subMu.Unlock()

	return ch
}

// Publish はメッセージを発行（履歴に保存 + 購読者に配信）
func (sc *SharedContext) Publish(msg WorkerMessage) {
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// 履歴保存
	sc.mu.Lock()
	sc.messages[msg.Topic] = append(sc.messages[msg.Topic], msg)
	sc.mu.Unlock()

	// 配信（購読者スライスをコピーしてから送る）
	sc.subMu.RLock()
	subs := sc.subscribers[msg.Topic]
	copied := make([]chan WorkerMessage, len(subs))
	copy(copied, subs)
	sc.subMu.RUnlock()

	for _, ch := range copied {
		// ブロックしないように送信（バッファ溢れはドロップ）
		select {
		case ch <- msg:
		default:
		}
	}
}

// Unsubscribe は購読を解除
func (sc *SharedContext) Unsubscribe(topic string, ch chan WorkerMessage) {
	sc.subMu.Lock()
	defer sc.subMu.Unlock()

	subs := sc.subscribers[topic]
	for i := range subs {
		if subs[i] == ch {
			// remove
			subs[i] = subs[len(subs)-1]
			subs = subs[:len(subs)-1]
			break
		}
	}
	if len(subs) == 0 {
		delete(sc.subscribers, topic)
	} else {
		sc.subscribers[topic] = subs
	}

	// 解除時にチャンネルを閉じる
	close(ch)
}

// Clear はコンテキストをクリア
func (sc *SharedContext) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.investigationResults = make(map[string]string)
	sc.stepResults = make(map[int]string)
	sc.fileChanges = make(map[int][]tools.FileChange)
	sc.completedSteps = make(map[int]bool)
	sc.errors = []WorkerError{}
	sc.messages = make(map[string][]WorkerMessage)

	// subscribers は別ロックでクリア
	sc.subMu.Lock()
	defer sc.subMu.Unlock()
	for _, subs := range sc.subscribers {
		for _, ch := range subs {
			close(ch)
		}
	}
	sc.subscribers = make(map[string][]chan WorkerMessage)
}
