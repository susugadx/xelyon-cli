package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecuteToolCallsParallel_AllParallel(t *testing.T) {
	toolCalls := []*ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "a.go"}},
		{Tool: "list_dir", Args: map[string]string{"path": "."}},
		{Tool: "search_code", Args: map[string]string{"pattern": "foo"}},
	}

	var mu sync.Mutex
	var maxConcurrent int32
	var currentConcurrent int32

	execFn := func(_ context.Context, tc *ToolCall) (string, *FileChange) {
		cur := atomic.AddInt32(&currentConcurrent, 1)
		mu.Lock()
		if cur > maxConcurrent {
			maxConcurrent = cur
		}
		mu.Unlock()

		time.Sleep(50 * time.Millisecond) // 並列実行を観測するための遅延
		atomic.AddInt32(&currentConcurrent, -1)

		return fmt.Sprintf("result-%s", tc.Tool), nil
	}

	ctx := context.Background()
	results := ExecuteToolCallsParallel(ctx, toolCalls, execFn)

	// 結果数の確認
	if len(results) != 3 {
		t.Fatalf("results count = %d, want 3", len(results))
	}

	// 結果が元のインデックス順で返ること
	for i, r := range results {
		if r.Index != i {
			t.Errorf("results[%d].Index = %d, want %d", i, r.Index, i)
		}
		expected := fmt.Sprintf("result-%s", toolCalls[i].Tool)
		if r.Result != expected {
			t.Errorf("results[%d].Result = %q, want %q", i, r.Result, expected)
		}
	}

	// 並列実行されたことを確認（maxConcurrent > 1）
	if maxConcurrent <= 1 {
		t.Errorf("maxConcurrent = %d, expected > 1 for parallel execution", maxConcurrent)
	}
}

func TestExecuteToolCallsParallel_Mixed(t *testing.T) {
	toolCalls := []*ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "a.go"}},     // parallel
		{Tool: "write_file", Args: map[string]string{"path": "b.go"}},    // sequential
		{Tool: "search_code", Args: map[string]string{"pattern": "foo"}}, // parallel
		{Tool: "str_replace", Args: map[string]string{"path": "c.go"}},   // sequential
	}

	var executionOrder []string
	var mu sync.Mutex

	execFn := func(_ context.Context, tc *ToolCall) (string, *FileChange) {
		// parallel ツールには短い遅延を入れて、順序が保証されないことをテスト
		if IsParallelSafe(tc) {
			time.Sleep(10 * time.Millisecond)
		}
		mu.Lock()
		executionOrder = append(executionOrder, tc.Tool)
		mu.Unlock()
		return fmt.Sprintf("ok-%s", tc.Tool), nil
	}

	ctx := context.Background()
	results := ExecuteToolCallsParallel(ctx, toolCalls, execFn)

	// 結果が元のインデックス順で返ること（実行順ではない）
	if len(results) != 4 {
		t.Fatalf("results count = %d, want 4", len(results))
	}
	expectedTools := []string{"read_file", "write_file", "search_code", "str_replace"}
	for i, r := range results {
		if r.TC.Tool != expectedTools[i] {
			t.Errorf("results[%d].TC.Tool = %q, want %q", i, r.TC.Tool, expectedTools[i])
		}
		if r.Result != fmt.Sprintf("ok-%s", expectedTools[i]) {
			t.Errorf("results[%d].Result = %q, want %q", i, r.Result, fmt.Sprintf("ok-%s", expectedTools[i]))
		}
	}

	// sequential ツールが parallel ツールの後に実行されること
	mu.Lock()
	defer mu.Unlock()
	writeIdx := -1
	strReplaceIdx := -1
	for i, name := range executionOrder {
		if name == "write_file" {
			writeIdx = i
		}
		if name == "str_replace" {
			strReplaceIdx = i
		}
	}
	// parallel ツールは先に実行されるため、sequential ツールのインデックスは 2 以上
	if writeIdx < 2 {
		t.Errorf("write_file executed at position %d, expected >= 2 (after parallel tools)", writeIdx)
	}
	if strReplaceIdx < 2 {
		t.Errorf("str_replace executed at position %d, expected >= 2 (after parallel tools)", strReplaceIdx)
	}
	// sequential ツール同士は順序が保持される
	if writeIdx >= strReplaceIdx {
		t.Errorf("write_file (pos %d) should execute before str_replace (pos %d)", writeIdx, strReplaceIdx)
	}
}

func TestExecuteToolCallsParallel_FailureContinuation(t *testing.T) {
	toolCalls := []*ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "a.go"}},
		{Tool: "read_file", Args: map[string]string{"path": "nonexistent.go"}},
		{Tool: "list_dir", Args: map[string]string{"path": "."}},
	}

	execFn := func(_ context.Context, tc *ToolCall) (string, *FileChange) {
		if tc.Args["path"] == "nonexistent.go" {
			return "Error: file not found", nil
		}
		return fmt.Sprintf("ok-%s-%s", tc.Tool, tc.Args["path"]), nil
	}

	ctx := context.Background()
	results := ExecuteToolCallsParallel(ctx, toolCalls, execFn)

	// 全ツールの結果が返ること（失敗しても他は継続）
	if len(results) != 3 {
		t.Fatalf("results count = %d, want 3", len(results))
	}

	// 成功したツール
	if !strings.HasPrefix(results[0].Result, "ok-") {
		t.Errorf("results[0] should succeed, got %q", results[0].Result)
	}

	// 失敗したツール
	if !strings.HasPrefix(results[1].Result, "Error:") {
		t.Errorf("results[1] should fail, got %q", results[1].Result)
	}

	// 3つ目のツールは失敗に関係なく実行される
	if !strings.HasPrefix(results[2].Result, "ok-") {
		t.Errorf("results[2] should succeed, got %q", results[2].Result)
	}
}

func TestExecuteToolCallsParallel_AllSequential(t *testing.T) {
	toolCalls := []*ToolCall{
		{Tool: "write_file", Args: map[string]string{"path": "a.go"}},
		{Tool: "str_replace", Args: map[string]string{"path": "b.go"}},
		{Tool: "delete_file", Args: map[string]string{"path": "c.go"}},
	}

	var executionOrder []string
	execFn := func(_ context.Context, tc *ToolCall) (string, *FileChange) {
		executionOrder = append(executionOrder, tc.Tool)
		return "ok", nil
	}

	ctx := context.Background()
	results := ExecuteToolCallsParallel(ctx, toolCalls, execFn)

	if len(results) != 3 {
		t.Fatalf("results count = %d, want 3", len(results))
	}

	// 順序が維持されること
	expected := []string{"write_file", "str_replace", "delete_file"}
	for i, name := range executionOrder {
		if name != expected[i] {
			t.Errorf("executionOrder[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestExecuteToolCallsParallel_SemaphoreLimit(t *testing.T) {
	// MaxParallelTools を超える数のツールを並列実行して、セマフォが機能することを確認
	n := MaxParallelTools + 4
	toolCalls := make([]*ToolCall, n)
	for i := 0; i < n; i++ {
		toolCalls[i] = &ToolCall{
			Tool: "read_file",
			Args: map[string]string{"path": fmt.Sprintf("file_%d.go", i)},
		}
	}

	var maxConcurrent int32
	var currentConcurrent int32

	execFn := func(_ context.Context, tc *ToolCall) (string, *FileChange) {
		cur := atomic.AddInt32(&currentConcurrent, 1)
		// ピーク並列数を記録
		for {
			old := atomic.LoadInt32(&maxConcurrent)
			if cur <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, cur) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt32(&currentConcurrent, -1)
		return "ok", nil
	}

	ctx := context.Background()
	results := ExecuteToolCallsParallel(ctx, toolCalls, execFn)

	if len(results) != n {
		t.Fatalf("results count = %d, want %d", len(results), n)
	}

	// セマフォによる制限が効いていること
	peak := atomic.LoadInt32(&maxConcurrent)
	if peak > int32(MaxParallelTools) {
		t.Errorf("maxConcurrent = %d, expected <= %d (semaphore limit)", peak, MaxParallelTools)
	}
	if peak < 2 {
		t.Errorf("maxConcurrent = %d, expected >= 2 (parallel execution should happen)", peak)
	}
}

func TestExecuteToolCallsParallel_Empty(t *testing.T) {
	results := ExecuteToolCallsParallel(context.Background(), nil, nil)
	if results != nil {
		t.Errorf("expected nil for empty input, got %v", results)
	}
}

// --- ExecuteQuiet の機能テスト ---
