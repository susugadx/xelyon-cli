package audit

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

// resetGlobalLogger はテスト間でグローバルロガーをリセット
func resetGlobalLogger() {
	globalLogger = nil
	once = sync.Once{}
}

func TestInit(t *testing.T) {
	defer resetGlobalLogger()

	tmpHome := testutil.SetupTempHome(t)
	logDir := filepath.Join(tmpHome, ".xelyon", "audit")

	// Init呼び出し
	err := Init(true)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// ログディレクトリが作成されていることを確認
	testutil.AssertFileExists(t, logDir)

	// ロガーが有効化されていることを確認
	logger := GetLogger()
	if logger == nil {
		t.Fatal("GetLogger returned nil")
	}

	if !logger.enabled {
		t.Error("Logger should be enabled")
	}
}

func TestLogToolExecution_Normal(t *testing.T) {
	defer resetGlobalLogger()

	tmpHome := testutil.SetupTempHome(t)
	err := Init(true)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	logger := GetLogger()

	// ツール実行をログ記録
	logger.LogToolExecution(
		"read_file",
		map[string]string{"path": "test.txt"},
		"file content here",
		nil,
		false,
	)

	// ログファイルを確認
	logFiles, err := filepath.Glob(filepath.Join(tmpHome, ".xelyon", "audit", "audit_*.jsonl"))
	if err != nil || len(logFiles) == 0 {
		t.Fatal("No log file created")
	}

	// ログファイルの内容を読み込み
	data, err := os.ReadFile(logFiles[0])
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// JSON解析
	var entry LogEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("Failed to parse log entry: %v", err)
	}

	// フィールド確認
	if entry.Tool != "read_file" {
		t.Errorf("Tool mismatch: got %s, expected read_file", entry.Tool)
	}

	if entry.Args["path"] != "test.txt" {
		t.Errorf("Args mismatch: got %v", entry.Args)
	}

	if entry.Output != "file content here" {
		t.Errorf("Output mismatch: got %s", entry.Output)
	}

	if !entry.Success {
		t.Error("Success should be true")
	}

	if entry.FileChanged {
		t.Error("FileChanged should be false")
	}
}

func TestLogToolExecution_WithError(t *testing.T) {
	defer resetGlobalLogger()

	testutil.SetupTempHome(t)
	Init(true)
	logger := GetLogger()

	// エラー付きでログ記録
	testErr := errors.New("file not found")
	logger.LogToolExecution(
		"delete_file",
		map[string]string{"path": "missing.txt"},
		"",
		testErr,
		false,
	)

	// ログファイル確認
	logFiles, _ := filepath.Glob(filepath.Join(os.Getenv("HOME"), ".xelyon", "audit", "audit_*.jsonl"))
	data, _ := os.ReadFile(logFiles[0])

	var entry LogEntry
	json.Unmarshal(data, &entry)

	if entry.Success {
		t.Error("Success should be false on error")
	}

	if entry.Error != "file not found" {
		t.Errorf("Error message mismatch: got %s", entry.Error)
	}
}

func TestSanitizeArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]string
		expected map[string]string
	}{
		{
			name: "password redaction",
			args: map[string]string{
				"username": "admin",
				"password": "secret123",
			},
			expected: map[string]string{
				"username": "admin",
				"password": "[REDACTED]",
			},
		},
		{
			name: "token redaction",
			args: map[string]string{
				"token":   "abc123xyz",
				"api_key": "key-123",
				"secret":  "my-secret",
			},
			expected: map[string]string{
				"token":   "[REDACTED]",
				"api_key": "[REDACTED]",
				"secret":  "[REDACTED]",
			},
		},
		{
			name: "long value truncation",
			args: map[string]string{
				"content": strings.Repeat("a", 250),
			},
			expected: map[string]string{
				"content": strings.Repeat("a", 200) + "... (truncated)",
			},
		},
		{
			name: "normal values",
			args: map[string]string{
				"path": "/home/user/file.txt",
				"mode": "0644",
			},
			expected: map[string]string{
				"path": "/home/user/file.txt",
				"mode": "0644",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeArgs(tt.args)

			for k, expectedVal := range tt.expected {
				if result[k] != expectedVal {
					t.Errorf("Key %s: got %s, expected %s", k, result[k], expectedVal)
				}
			}
		})
	}
}

func TestLogToolExecution_OutputTruncation(t *testing.T) {
	defer resetGlobalLogger()

	testutil.SetupTempHome(t)
	Init(true)
	logger := GetLogger()

	// 長い出力（500文字超）
	longOutput := strings.Repeat("x", 600)
	logger.LogToolExecution(
		"read_file",
		map[string]string{"path": "large.txt"},
		longOutput,
		nil,
		false,
	)

	// ログファイル確認
	logFiles, _ := filepath.Glob(filepath.Join(os.Getenv("HOME"), ".xelyon", "audit", "audit_*.jsonl"))
	data, _ := os.ReadFile(logFiles[0])

	var entry LogEntry
	json.Unmarshal(data, &entry)

	// 500文字に切り詰められていることを確認
	expectedOutput := strings.Repeat("x", 500) + "... (truncated)"
	if entry.Output != expectedOutput {
		t.Errorf("Output should be truncated to 500 chars")
	}
}

func TestLogToolExecution_Concurrent(t *testing.T) {
	defer resetGlobalLogger()

	tmpHome := testutil.SetupTempHome(t)
	Init(true)
	logger := GetLogger()

	// 並行でログ記録（100 goroutines）
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			logger.LogToolExecution(
				"test_tool",
				map[string]string{"id": string(rune(id))},
				"output",
				nil,
				false,
			)
		}(i)
	}

	wg.Wait()

	// ログファイルを確認
	logFiles, _ := filepath.Glob(filepath.Join(tmpHome, ".xelyon", "audit", "audit_*.jsonl"))
	if len(logFiles) == 0 {
		t.Fatal("No log file created")
	}

	// ログエントリ数を確認（100行あるべき）
	file, err := os.Open(logFiles[0])
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	if lineCount != 100 {
		t.Errorf("Expected 100 log entries, got %d", lineCount)
	}
}

func TestLogToolExecution_Disabled(t *testing.T) {
	defer resetGlobalLogger()

	tmpHome := testutil.SetupTempHome(t)
	logDir := filepath.Join(tmpHome, ".xelyon", "audit")

	// 無効状態で初期化
	Init(false)
	logger := GetLogger()

	logger.LogToolExecution(
		"test_tool",
		map[string]string{"key": "value"},
		"output",
		nil,
		false,
	)

	// ログファイルが作成されていないことを確認
	logFiles, _ := filepath.Glob(filepath.Join(logDir, "audit_*.jsonl"))
	if len(logFiles) > 0 {
		// ファイルが存在する場合、空であることを確認
		data, _ := os.ReadFile(logFiles[0])
		if len(data) > 0 {
			t.Error("No log should be written when logger is disabled")
		}
	}
}

func TestGetLogger_BeforeInit(t *testing.T) {
	defer resetGlobalLogger()

	// Init呼び出し前にGetLogger
	logger := GetLogger()

	if logger == nil {
		t.Fatal("GetLogger should return non-nil logger even before Init")
	}

	if logger.enabled {
		t.Error("Logger should be disabled before Init")
	}

	// ログ記録を試みる（パニックしないことを確認）
	logger.LogToolExecution("test", nil, "", nil, false)
}
