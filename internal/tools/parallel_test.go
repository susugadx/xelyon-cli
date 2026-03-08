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

func TestIsParallelSafe_StaticTools(t *testing.T) {
	tests := []struct {
		tool     string
		expected bool
	}{
		{"read_file", true},
		{"read_files", true},
		{"list_dir", true},
		{"search_code", true},
		{"web_search", true},
		{"git_status", true},
		{"git_log", true},
		{"git_diff", true},
		{"write_file", false},
		{"str_replace", false},
		{"delete_file", false},
		{"ask_user_question", false},
		{"create_plan", false},
		{"unknown_tool", false},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			tc := &ToolCall{Tool: tt.tool, Args: map[string]string{}}
			got := IsParallelSafe(tc)
			if got != tt.expected {
				t.Errorf("IsParallelSafe(%q) = %v, want %v", tt.tool, got, tt.expected)
			}
		})
	}
}

func TestIsParallelSafe_BashReadOnly(t *testing.T) {
	tests := []struct {
		command  string
		expected bool
	}{
		// parallel-safe bash commands
		{"pwd", true},
		{"ls", true},
		{"ls -la", true},
		{"find . -name '*.go'", true},
		{"rg 'pattern' src/", true},
		{"grep -r 'TODO' .", true},
		{"cat README.md", true},
		{"head -20 file.go", true},
		{"tail -10 file.go", true},
		{"wc -l file.go", true},
		{"git status", true},
		{"git diff", true},
		{"git diff --stat", true},
		{"git log", true},
		{"git log --oneline -10", true},
		{"git branch", true},
		{"git ls-files", true},
		{"git show HEAD", true},
		{"git remote -v", true},

		// NOT parallel-safe bash commands
		{"rm -rf /tmp/foo", false},
		{"echo hello", false},
		{"go build ./...", false},
		{"npm install", false},
		{"make", false},
		{"", false},

		// Pipes/redirects/chains are never parallel-safe
		{"cat file.go | grep TODO", false},
		{"ls > output.txt", false},
		{"ls >> output.txt", false},
		{"ls && pwd", false},
		{"ls; pwd", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			tc := &ToolCall{
				Tool: "bash",
				Args: map[string]string{"command": tt.command},
			}
			got := IsParallelSafe(tc)
			if got != tt.expected {
				t.Errorf("IsParallelSafe(bash %q) = %v, want %v", tt.command, got, tt.expected)
			}
		})
	}
}

func TestIsParallelSafe_BashFromRawArgs(t *testing.T) {
	// Args が未設定で RawArgs のみの場合
	tc := &ToolCall{
		Tool:    "bash",
		RawArgs: map[string]any{"command": "git status"},
		Args:    map[string]string{}, // 空
	}
	if !IsParallelSafe(tc) {
		t.Error("should be parallel-safe when command is in RawArgs")
	}
}

func TestClassifyToolCalls(t *testing.T) {
	toolCalls := []*ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "a.go"}},                     // 0: parallel
		{Tool: "write_file", Args: map[string]string{"path": "b.go"}},                    // 1: sequential
		{Tool: "search_code", Args: map[string]string{"pattern": "foo"}},                 // 2: parallel
		{Tool: "bash", Args: map[string]string{"command": "git status"}},                 // 3: parallel (read-only bash)
		{Tool: "bash", Args: map[string]string{"command": "go build ./..."}},             // 4: sequential (non-read-only bash)
		{Tool: "list_dir", Args: map[string]string{"path": "."}},                         // 5: parallel
		{Tool: "str_replace", Args: map[string]string{"path": "c.go", "old_str": "foo"}}, // 6: sequential
	}

	parallelIdx, seqIdx := ClassifyToolCalls(toolCalls)

	expectedParallel := []int{0, 2, 3, 5}
	expectedSeq := []int{1, 4, 6}

	if len(parallelIdx) != len(expectedParallel) {
		t.Fatalf("parallel count = %d, want %d", len(parallelIdx), len(expectedParallel))
	}
	for i, idx := range parallelIdx {
		if idx != expectedParallel[i] {
			t.Errorf("parallelIdx[%d] = %d, want %d", i, idx, expectedParallel[i])
		}
	}

	if len(seqIdx) != len(expectedSeq) {
		t.Fatalf("sequential count = %d, want %d", len(seqIdx), len(expectedSeq))
	}
	for i, idx := range seqIdx {
		if idx != expectedSeq[i] {
			t.Errorf("seqIdx[%d] = %d, want %d", i, idx, expectedSeq[i])
		}
	}
}

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

func TestExecuteToolCallsParallel_ContextCancel(t *testing.T) {
	toolCalls := []*ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "a.go"}},
		{Tool: "list_dir", Args: map[string]string{"path": "."}},
		{Tool: "search_code", Args: map[string]string{"pattern": "foo"}},
	}

	ctx, cancel := context.WithCancel(context.Background())

	var execCount int32
	execFn := func(ctx context.Context, tc *ToolCall) (string, *FileChange) {
		count := atomic.AddInt32(&execCount, 1)
		// 最初のツール実行後にキャンセル
		if count == 1 {
			cancel()
		}
		// キャンセルされたら早期リターン
		if ctx.Err() != nil {
			return "Error: context cancelled", nil
		}
		time.Sleep(100 * time.Millisecond)
		return "ok", nil
	}

	results := ExecuteToolCallsParallel(ctx, toolCalls, execFn)

	if len(results) != 3 {
		t.Fatalf("results count = %d, want 3", len(results))
	}

	// キャンセル後のツールはエラーを含むはず
	cancelledCount := 0
	for _, r := range results {
		if strings.Contains(r.Result, "cancel") {
			cancelledCount++
		}
	}
	if cancelledCount == 0 {
		t.Error("expected at least one cancelled result")
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

// --- ExecutionContext の goroutine 安全性テスト ---

func TestExecutionContext_ConcurrentReadWrite(t *testing.T) {
	// 複数 goroutine から同時に Set/Get しても panic / race しないことを確認。
	// go test -race で実行すると data race を検出できる。
	var wg sync.WaitGroup
	ctx1 := ExecutionContext{ProviderName: "provider-A", Model: "model-A"}
	ctx2 := ExecutionContext{ProviderName: "provider-B", Model: "model-B"}

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			if i%2 == 0 {
				SetExecutionContext(ctx1)
			} else {
				SetExecutionContext(ctx2)
			}
		}
	}()

	// Reader goroutines
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				got := GetExecutionContext()
				// 値は ctx1 か ctx2 か空のいずれか（Set 前の初期値）
				if got.ProviderName != "" && got.ProviderName != "provider-A" && got.ProviderName != "provider-B" {
					t.Errorf("unexpected ProviderName: %q", got.ProviderName)
				}
			}
		}()
	}
	wg.Wait()

	ClearExecutionContext()
	got := GetExecutionContext()
	if got.ProviderName != "" || got.Model != "" {
		t.Errorf("after Clear, expected empty context, got %+v", got)
	}
}

func TestExecutionContext_ParallelBatchPattern(t *testing.T) {
	// executeToolCallsWithParallel の実際のパターンを模倣:
	// Set → 複数 goroutine で Get（読み取りのみ）→ Clear
	expected := ExecutionContext{ProviderName: "test-provider", Model: "test-model"}
	SetExecutionContext(expected)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got := GetExecutionContext()
			if got.ProviderName != expected.ProviderName {
				t.Errorf("goroutine got ProviderName=%q, want %q", got.ProviderName, expected.ProviderName)
			}
			if got.Model != expected.Model {
				t.Errorf("goroutine got Model=%q, want %q", got.Model, expected.Model)
			}
		}()
	}
	wg.Wait()
	ClearExecutionContext()
}

// --- ExecuteQuiet の stdout 抑制テスト ---

func TestExecuteQuiet_NoHeaderOutput(t *testing.T) {
	// ExecuteQuiet は "🔧 Tool: ..." ヘッダーや折りたたみ表示を出力しない。
	// ここでは関数が正常に呼べることと、結果を返すことを確認する。
	// （stdout のキャプチャは testing では煩雑なため、機能テストのみ）
	tc := &ToolCall{
		Tool: "list_dir",
		Args: map[string]string{"path": "."},
	}
	result, change := ExecuteQuiet(tc)

	// list_dir は結果を返すはず
	if result == "" {
		t.Error("ExecuteQuiet returned empty result")
	}
	// list_dir は FileChange を返さない
	if change != nil {
		t.Error("ExecuteQuiet returned non-nil change for list_dir")
	}
}

func TestIsBashParallelSafe(t *testing.T) {
	tests := []struct {
		command  string
		expected bool
	}{
		{"pwd", true},
		{"ls", true},
		{"ls -la /tmp", true},
		{"find . -name '*.go'", true},
		{"rg pattern src/", true},
		{"grep -rn TODO .", true},
		{"cat file.txt", true},
		{"head -20 main.go", true},
		{"tail -5 log.txt", true},
		{"wc -l file.go", true},
		{"git status", true},
		{"git diff", true},
		{"git diff --stat", true},
		{"git log", true},
		{"git log --oneline -10", true},
		{"git branch", true},
		{"git ls-files", true},
		{"git show HEAD:main.go", true},
		{"git remote -v", true},

		// NOT parallel-safe
		{"echo hello", false},
		{"go build ./...", false},
		{"npm install", false},
		{"rm file.txt", false},
		{"mkdir -p dir", false},
		{"touch file.txt", false},
		{"", false},

		// Pipes/redirects/chains
		{"cat file | grep foo", false},
		{"ls > out.txt", false},
		{"ls >> out.txt", false},
		{"pwd && ls", false},
		{"pwd; ls", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := IsBashParallelSafe(tt.command)
			if got != tt.expected {
				t.Errorf("IsBashParallelSafe(%q) = %v, want %v", tt.command, got, tt.expected)
			}
		})
	}
}
