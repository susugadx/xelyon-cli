package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestContainsFailure_GoTestFail(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   bool
	}{
		{
			name:   "Go test FAIL pattern",
			result: "--- FAIL: TestSomething (0.00s)\n    main_test.go:10: expected true, got false",
			want:   true,
		},
		{
			name:   "Go test FAIL tab pattern",
			result: "FAIL\tgithub.com/example/pkg\t0.010s",
			want:   true,
		},
		{
			name:   "Go test pass",
			result: "ok  \tgithub.com/example/pkg\t0.010s",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, _ := plan.ContainsFailure(tt.result)
			if failed != tt.want {
				t.Errorf("ContainsFailure() = %v, want %v", failed, tt.want)
			}
		})
	}
}

func TestContainsFailure_Panic(t *testing.T) {
	result := `goroutine 1 [running]:
panic: runtime error: index out of range [1] with length 1

goroutine 1 [running]:
main.main()
	/home/user/project/main.go:10 +0x45`

	failed, reason := plan.ContainsFailure(result)
	if !failed {
		t.Error("ContainsFailure() should detect panic")
	}
	if reason != "Panic detected" {
		t.Errorf("ContainsFailure() reason = %q, want 'Panic detected'", reason)
	}
}

func TestContainsFailure_NpmError(t *testing.T) {
	result := `npm ERR! code ENOENT
npm ERR! syscall open
npm ERR! path /home/user/project/package.json
npm ERR! errno -2
npm ERR! enoent ENOENT: no such file or directory`

	failed, reason := plan.ContainsFailure(result)
	if !failed {
		t.Error("ContainsFailure() should detect npm error")
	}
	if reason != "npm error" {
		t.Errorf("ContainsFailure() reason = %q, want 'npm error'", reason)
	}
}

func TestContainsFailure_ExitStatus(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   bool
	}{
		{
			name:   "exit status 1",
			result: "Command failed with exit status 1",
			want:   true,
		},
		{
			name:   "exit status 0 (success)",
			result: "Command completed with exit status 0",
			want:   false, // exit status 0 is success, should not be detected as failure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, _ := plan.ContainsFailure(tt.result)
			if failed != tt.want {
				t.Errorf("ContainsFailure() = %v, want %v", failed, tt.want)
			}
		})
	}
}

func TestContainsFailure_NoFailure(t *testing.T) {
	tests := []struct {
		name   string
		result string
	}{
		{
			name:   "success output",
			result: "Build successful\nAll tests passed",
		},
		{
			name:   "empty output",
			result: "",
		},
		{
			name:   "normal command output",
			result: "total 16\ndrwxr-xr-x 5 user user 4096 Jan 15 10:00 .",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, _ := plan.ContainsFailure(tt.result)
			if failed {
				t.Errorf("ContainsFailure() should not detect failure for %q", tt.name)
			}
		})
	}
}

func TestContainsFailure_BuildAndCompileErrors(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   bool
	}{
		{
			name:   "build failed",
			result: "go build: build failed: exit status 1",
			want:   true,
		},
		{
			name:   "compile error",
			result: "compile error: undefined: someFunction",
			want:   true,
		},
		{
			name:   "SyntaxError with colon",
			result: "SyntaxError: Unexpected token",
			want:   true,
		},
		{
			name:   "TypeError with colon",
			result: "TypeError: Cannot read property 'x' of undefined",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, _ := plan.ContainsFailure(tt.result)
			if failed != tt.want {
				t.Errorf("ContainsFailure() = %v, want %v", failed, tt.want)
			}
		})
	}
}

// TestContainsFailure_FalsePositives は誤検知を起こさないことをテスト
// コード検索結果やログ出力に "Error" 文字列が含まれていても失敗と判定しない
func TestContainsFailure_FalsePositives(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   bool
	}{
		{
			name: "t.Errorf in test code",
			result: `internal/agent/plan_test.go:397:	t.Errorf("ContainsFailure() = %v, want %v", failed, tt.want)
internal/agent/plan_test.go:485:	t.Errorf("ContainsFailure() should not detect failure for %q", tt.name)`,
			want: false, // コード検索結果は失敗ではない
		},
		{
			name:   "ErrorHandler function name",
			result: "func ErrorHandler(err error) {\n    log.Printf(\"Error: %v\", err)\n}",
			want:   false, // 関数定義は失敗ではない
		},
		{
			name:   "fmt.Errorf in code",
			result: `return fmt.Errorf("failed to parse: %w", err)`,
			want:   false, // コード内のfmt.Errorfは失敗ではない
		},
		{
			name:   "log with error message",
			result: "2024-01-15 10:00:00 INFO: Processing completed\n2024-01-15 10:00:01 DEBUG: Error count: 0",
			want:   false, // ログ出力（エラーカウント0）は失敗ではない
		},
		{
			name:   "grep result with Error string",
			result: "grep result:\ninternal/api/client.go:50: type ErrorResponse struct {\ninternal/api/client.go:51:     Error string `json:\"error\"`",
			want:   false, // コード検索結果は失敗ではない
		},
		{
			name:   "markdown documentation",
			result: "## Error Handling\nThis section describes how errors are handled.\n\n### ErrorTypes\n- ValidationError\n- NetworkError",
			want:   false, // ドキュメントは失敗ではない
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, reason := plan.ContainsFailure(tt.result)
			if failed != tt.want {
				t.Errorf("ContainsFailure() = %v (reason: %s), want %v", failed, reason, tt.want)
			}
		})
	}
}
