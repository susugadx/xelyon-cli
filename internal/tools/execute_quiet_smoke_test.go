package tools

import "testing"

func TestExecuteQuiet_ReturnsResult(t *testing.T) {
	// ExecuteQuietWithContext が正常に結果を返すことを確認する機能テスト。
	tc := &ToolCall{
		Tool: "list_dir",
		Args: map[string]string{"path": "."},
	}
	result, change := ExecuteQuietWithContext(ExecutionContext{}, tc)

	// list_dir は結果を返すはず
	if result == "" {
		t.Error("ExecuteQuietWithContext returned empty result")
	}
	// list_dir は FileChange を返さない
	if change != nil {
		t.Error("ExecuteQuietWithContext returned non-nil change for list_dir")
	}
}
