package tools

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

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
