package agent

import (
	"fmt"
	"strings"
	"sync"

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

// Clear はコンテキストをクリア
func (sc *SharedContext) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.investigationResults = make(map[string]string)
	sc.stepResults = make(map[int]string)
	sc.fileChanges = make(map[int][]tools.FileChange)
	sc.completedSteps = make(map[int]bool)
	sc.errors = []WorkerError{}
}
