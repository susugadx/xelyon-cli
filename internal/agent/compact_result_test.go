package agent

import (
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestCompactBash_VerifySuccess(t *testing.T) {
	// verify + 成功 + 長い出力 → 1行サマリー
	longOutput := strings.Repeat("ok  github.com/foo/bar  0.3s\n", 30)
	tc := &tools.ToolCall{Tool: "bash", Args: map[string]string{"command": "go test ./..."}}
	got := compactBash(tc.Args["command"], longOutput)
	if got != "OK: go test ./... (exit 0)" {
		t.Errorf("expected OK summary, got: %s", got)
	}
}

func TestCompactBash_VerifySuccessShort(t *testing.T) {
	// verify + 成功 + 短い出力 → そのまま
	got := compactBash("go vet ./...", "(no output)")
	if got != "(no output)" {
		t.Errorf("expected original output, got: %s", got)
	}
}

func TestCompactBash_NonVerifySuccess(t *testing.T) {
	// 非verify + 成功 → そのまま
	longOutput := strings.Repeat("On branch main\n", 50)
	got := compactBash("git status", longOutput)
	if got != longOutput {
		t.Error("non-verify success should return original output")
	}
}

func TestCompactBash_Failure(t *testing.T) {
	// 失敗 → tail 30行
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "ok  pkg/x  0.1s")
	}
	lines = append(lines, "--- FAIL: TestFoo")
	lines = append(lines, "    error here")
	longOutput := strings.Join(lines, "\n")

	got := compactBash("go test ./...", "Error: exit status 1\nOutput: "+longOutput)
	if !strings.Contains(got, "FAIL: TestFoo") {
		t.Error("failure output should contain error details")
	}
	if !strings.Contains(got, "lines omitted") {
		t.Error("failure output should indicate omitted lines")
	}
}

func TestCompactBash_FailureShort(t *testing.T) {
	// 短い失敗出力 → そのまま
	result := "Error: exit status 1\nOutput: compilation failed"
	got := compactBash("go build ./...", result)
	if got != result {
		t.Errorf("short failure should return as-is, got: %s", got)
	}
}

func TestCompactBash_CommandInterrupted(t *testing.T) {
	result := "Command interrupted.\nPartial output:\nsome output"
	got := compactBash("go test ./...", result)
	if got != result {
		t.Errorf("interrupted should return as-is for short output, got: %s", got)
	}
}

func TestCompactBash_VerifyChainSuccess(t *testing.T) {
	// verifyチェーンコマンド + 長い出力
	longOutput := strings.Repeat("checking...\n", 100)
	got := compactBash("go fmt ./... && go build ./... && go test ./...", longOutput)
	if !strings.HasPrefix(got, "OK:") {
		t.Errorf("verify chain should be compressed, got: %s", got)
	}
}

func TestCompactBash_TruncateCommand(t *testing.T) {
	got := truncateCommand("short", 120)
	if got != "short" {
		t.Errorf("short command should not be truncated, got: %s", got)
	}

	longCmd := strings.Repeat("a", 200)
	got = truncateCommand(longCmd, 120)
	if len(got) != 120 {
		t.Errorf("truncated command should be 120 chars, got: %d", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Error("truncated command should end with ...")
	}
}

func TestTailLines(t *testing.T) {
	// 短い入力 → そのまま
	got := tailLines("line1\nline2", 30)
	if got != "line1\nline2" {
		t.Errorf("short input should return as-is, got: %s", got)
	}

	// 長い入力 → tail N行 + omission notice
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "line")
	}
	input := strings.Join(lines, "\n")
	got = tailLines(input, 5)
	if !strings.HasPrefix(got, "[... 45 lines omitted ...]") {
		t.Errorf("should start with omission notice, got: %s", got[:50])
	}
	// 末尾5行が含まれていることを確認
	gotLines := strings.Split(got, "\n")
	// 最初の行は omission notice
	if len(gotLines) != 6 { // 1 notice + 5 tail lines
		t.Errorf("expected 6 lines (1 notice + 5 tail), got: %d", len(gotLines))
	}
}

func TestIsBashError(t *testing.T) {
	tests := []struct {
		result string
		want   bool
	}{
		{"Error: exit status 1\nOutput: failed", true},
		{"Error: command not found", true},
		{"Command interrupted.\nPartial output:", true},
		{"ok  github.com/foo/bar  0.3s", false},
		{"(no output)", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.result, func(t *testing.T) {
			if got := isBashError(tt.result); got != tt.want {
				t.Errorf("isBashError(%q) = %v, want %v", tt.result, got, tt.want)
			}
		})
	}
}

func TestCompactToolResult_OtherTools(t *testing.T) {
	// read_file, search_code 等 → そのまま
	got := compactBash("", "file contents...")
	// bash with empty command and no error → not verify, returns as-is
	if got != "file contents..." {
		t.Errorf("non-bash tools should return as-is, got: %s", got)
	}
}

// ── ログ系コマンド ──

func TestCompactBash_LogCommand(t *testing.T) {
	var lines []string
	for i := range 200 {
		lines = append(lines, fmt.Sprintf("2026-03-10 10:00:%02d log line %d", i%60, i))
	}
	longLog := strings.Join(lines, "\n")

	got := compactBash("kubectl logs my-pod", longLog)
	if !strings.Contains(got, "200 lines total") {
		t.Errorf("should contain total line count, got: %s", got[:80])
	}
	if !strings.Contains(got, "showing last 50") {
		t.Error("should indicate tail lines")
	}
	if !strings.Contains(got, "log line 199") {
		t.Error("should contain last log line")
	}
}

func TestCompactBash_LogCommandShort(t *testing.T) {
	shortLog := "2026-03-10 log line 1\n2026-03-10 log line 2"
	got := compactBash("docker logs my-container", shortLog)
	if got != shortLog {
		t.Error("short log should return as-is")
	}
}

func TestIsLogCommand(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{"kubectl logs my-pod", true},
		{"docker logs my-container", true},
		{"docker compose logs", true},
		{"journalctl -u nginx", true},
		{"heroku logs --tail", true},
		{"aws logs get-log-events --log-group /ecs/app", true},
		{"stern my-pod", true},

		// 非ログ
		{"kubectl get pods", false},
		{"docker ps", false},
		{"git log", false}, // git logはPhase 2で別処理
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			if got := isLogCommand(tt.command); got != tt.want {
				t.Errorf("isLogCommand(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestCompactLogOutput(t *testing.T) {
	// 短い → そのまま
	short := strings.Repeat("line\n", 30)
	if got := compactLogOutput(short); got != short {
		t.Error("short log should return as-is")
	}

	// 長い → tail 50行
	var lines []string
	for i := range 100 {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	long := strings.Join(lines, "\n")
	got := compactLogOutput(long)
	if !strings.Contains(got, "100 lines total") {
		t.Errorf("should contain total count, got: %s", got[:60])
	}
	if !strings.Contains(got, "line 99") {
		t.Error("should contain last line")
	}
}
