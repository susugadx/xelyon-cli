package agent

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestNewSharedContext(t *testing.T) {
	sc := NewSharedContext()
	if sc == nil {
		t.Fatal("NewSharedContext returned nil")
	}
	if sc.investigationResults == nil {
		t.Error("investigationResults map not initialized")
	}
	if sc.stepResults == nil {
		t.Error("stepResults map not initialized")
	}
	if sc.fileChanges == nil {
		t.Error("fileChanges map not initialized")
	}
	if sc.completedSteps == nil {
		t.Error("completedSteps map not initialized")
	}
}

func TestSharedContext_InvestigationResults(t *testing.T) {
	sc := NewSharedContext()

	// 追加
	sc.AddInvestigationResult("query1", "result1")
	sc.AddInvestigationResult("query2", "result2")

	// 取得
	result, ok := sc.GetInvestigationResult("query1")
	if !ok {
		t.Error("GetInvestigationResult returned false for existing query")
	}
	if result != "result1" {
		t.Errorf("expected 'result1', got '%s'", result)
	}

	// 存在しないクエリ
	_, ok = sc.GetInvestigationResult("nonexistent")
	if ok {
		t.Error("GetInvestigationResult returned true for nonexistent query")
	}

	// 全取得
	all := sc.GetAllInvestigationResults()
	if len(all) != 2 {
		t.Errorf("expected 2 results, got %d", len(all))
	}
}

func TestSharedContext_StepResults(t *testing.T) {
	sc := NewSharedContext()

	// 追加
	sc.AddStepResult(1, "step1 result")
	sc.AddStepResult(2, "step2 result")

	// 取得
	result, ok := sc.GetStepResult(1)
	if !ok {
		t.Error("GetStepResult returned false for existing step")
	}
	if result != "step1 result" {
		t.Errorf("expected 'step1 result', got '%s'", result)
	}

	// 存在しないステップ
	_, ok = sc.GetStepResult(999)
	if ok {
		t.Error("GetStepResult returned true for nonexistent step")
	}
}

func TestSharedContext_FileChanges(t *testing.T) {
	sc := NewSharedContext()

	change1 := tools.FileChange{FilePath: "file1.go", Tool: "write_file"}
	change2 := tools.FileChange{FilePath: "file2.go", Tool: "str_replace"}

	// 追加
	sc.AddFileChange(1, change1)
	sc.AddFileChange(1, change2)

	// 取得
	changes := sc.GetFileChanges(1)
	if len(changes) != 2 {
		t.Errorf("expected 2 changes, got %d", len(changes))
	}

	// 全取得
	allChanges := sc.GetAllFileChanges()
	if len(allChanges) != 2 {
		t.Errorf("expected 2 total changes, got %d", len(allChanges))
	}
}

func TestSharedContext_CompletedSteps(t *testing.T) {
	sc := NewSharedContext()

	// 完了前
	if sc.IsStepCompleted(1) {
		t.Error("step 1 should not be completed")
	}

	// 完了マーク
	sc.MarkStepCompleted(1)
	sc.MarkStepCompleted(3)

	// 完了後
	if !sc.IsStepCompleted(1) {
		t.Error("step 1 should be completed")
	}
	if !sc.IsStepCompleted(3) {
		t.Error("step 3 should be completed")
	}
	if sc.IsStepCompleted(2) {
		t.Error("step 2 should not be completed")
	}

	// 完了ステップ ID 取得
	ids := sc.GetCompletedStepIDs()
	if len(ids) != 2 {
		t.Errorf("expected 2 completed steps, got %d", len(ids))
	}
}

func TestSharedContext_GetContextForStep(t *testing.T) {
	sc := NewSharedContext()

	// 依存ステップの結果を追加
	sc.AddStepResult(1, "Step 1 completed successfully")
	sc.AddStepResult(2, "Step 2 completed successfully")

	// 依存ステップを持つステップ
	step := &plan.PlanStep{
		ID:        3,
		DependsOn: []int{1, 2},
	}

	context := sc.GetContextForStep(step)
	if context == "" {
		t.Error("expected non-empty context")
	}
	if !strings.Contains(context, "Step 1 Result") {
		t.Error("context should contain Step 1 Result")
	}
	if !strings.Contains(context, "Step 2 Result") {
		t.Error("context should contain Step 2 Result")
	}
}

func TestSharedContext_Errors(t *testing.T) {
	sc := NewSharedContext()

	// エラー追加
	sc.AddError(WorkerError{WorkerID: 0, StepID: 1, Error: errors.New("test error 1")})
	sc.AddError(WorkerError{WorkerID: 1, StepID: 2, Error: errors.New("test error 2")})

	// 取得
	errs := sc.GetErrors()
	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errs))
	}
}

func TestSharedContext_Clear(t *testing.T) {
	sc := NewSharedContext()

	// データを追加
	sc.AddInvestigationResult("query", "result")
	sc.AddStepResult(1, "result")
	sc.MarkStepCompleted(1)

	// クリア
	sc.Clear()

	// 確認
	_, ok := sc.GetInvestigationResult("query")
	if ok {
		t.Error("investigation result should be cleared")
	}
	_, ok = sc.GetStepResult(1)
	if ok {
		t.Error("step result should be cleared")
	}
	if sc.IsStepCompleted(1) {
		t.Error("completed step should be cleared")
	}
}

func TestSharedContext_ConcurrentAccess(t *testing.T) {
	sc := NewSharedContext()
	var wg sync.WaitGroup

	// 並行アクセステスト
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			// 調査結果の追加と取得
			sc.AddInvestigationResult("query"+string(rune(n)), "result")
			sc.GetInvestigationResult("query" + string(rune(n)))
			sc.GetAllInvestigationResults()

			// ステップ結果の追加と取得
			sc.AddStepResult(n, "result")
			sc.GetStepResult(n)

			// 完了ステップのマーク
			sc.MarkStepCompleted(n)
			sc.IsStepCompleted(n)
			sc.GetCompletedStepIDs()

			// ファイル変更
			sc.AddFileChange(n, tools.FileChange{FilePath: "test.go"})
			sc.GetFileChanges(n)
			sc.GetAllFileChanges()
		}(i)
	}

	wg.Wait()
	// データ競合がなければテスト成功
}

func TestSharedContext_ImmutableReturns(t *testing.T) {
	sc := NewSharedContext()

	// データを追加
	sc.AddInvestigationResult("query", "original")

	// 取得した結果を変更しても元データに影響しないことを確認
	results := sc.GetAllInvestigationResults()
	results["query"] = "modified"

	// 元データが変更されていないことを確認
	original, _ := sc.GetInvestigationResult("query")
	if original != "original" {
		t.Error("original data should not be modified")
	}
}

func TestSubscribe(t *testing.T) {
	sc := NewSharedContext()
	ch := sc.Subscribe("step_completed", 10)
	if ch == nil {
		t.Fatal("Subscribe returned nil channel")
	}
}

func TestPublish(t *testing.T) {
	sc := NewSharedContext()

	topic := "step_completed"
	raw := sc.Subscribe(topic, 10)

	msg := WorkerMessage{
		FromWorker: 1,
		Topic:      topic,
		Content:    "done",
		StepID:     42,
		Timestamp:  time.Now(),
	}
	sc.Publish(msg)

	select {
	case got := <-raw:
		if got.Content != msg.Content || got.StepID != msg.StepID || got.Topic != msg.Topic {
			t.Fatalf("unexpected message: %+v", got)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for message")
	}
}

func TestPublishWithHistory(t *testing.T) {
	sc := NewSharedContext()

	topic := "file_changed"
	sc.Publish(WorkerMessage{FromWorker: 1, Topic: topic, Content: "a", Timestamp: time.Now()})
	sc.Publish(WorkerMessage{FromWorker: 2, Topic: topic, Content: "b", Timestamp: time.Now()})

	msgs := sc.GetMessages(topic)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Content != "a" || msgs[1].Content != "b" {
		t.Fatalf("unexpected history: %+v", msgs)
	}
}

func TestGetMessages(t *testing.T) {
	sc := NewSharedContext()

	topic := "step_failed"
	sc.Publish(WorkerMessage{FromWorker: 1, Topic: topic, Content: "x", Timestamp: time.Now()})

	msgs := sc.GetMessages(topic)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "x" {
		t.Fatalf("unexpected message: %+v", msgs[0])
	}

	// 返り値が不変（コピー）であること
	msgs[0].Content = "modified"
	msgs2 := sc.GetMessages(topic)
	if msgs2[0].Content != "x" {
		t.Fatal("GetMessages should return a copy")
	}
}

func TestUnsubscribe(t *testing.T) {
	sc := NewSharedContext()

	topic := "step_failed"

	// Subscribe は <-chan を返すため、Unsubscribe 用に元の chan を直接登録する
	ch := make(chan WorkerMessage, 1)
	sc.subMu.Lock()
	sc.subscribers[topic] = append(sc.subscribers[topic], ch)
	sc.subMu.Unlock()

	sc.Unsubscribe(topic, ch)

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected channel to be closed")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}

func TestPublishNoBlock(t *testing.T) {
	sc := NewSharedContext()
	topic := "step_completed"

	// バッファ 1 にして、読まずに Publish を大量に行う
	recv := sc.Subscribe(topic, 1)
	_ = recv

	start := time.Now()
	for i := 0; i < 1000; i++ {
		sc.Publish(WorkerMessage{FromWorker: 1, Topic: topic, Content: "x", StepID: i})
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatal("Publish appears to block when subscriber buffer is full")
	}
}
