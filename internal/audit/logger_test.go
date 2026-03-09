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

func TestInitReturnsEnabledLogger(t *testing.T) {
	tmpHome := testutil.SetupTempHome(t)
	logDir := filepath.Join(tmpHome, ".xelyon", "audit")

	logger, err := Init(true)
	if err != nil {
		t.Fatalf("Init(true) error = %v", err)
	}
	if logger == nil {
		t.Fatal("Init(true) returned nil logger")
	}
	if !logger.enabled {
		t.Fatal("Init(true) should return enabled logger")
	}
	testutil.AssertFileExists(t, logDir)
}

func TestInitReturnsDisabledLogger(t *testing.T) {
	logger, err := Init(false)
	if err != nil {
		t.Fatalf("Init(false) error = %v", err)
	}
	if logger == nil {
		t.Fatal("Init(false) returned nil logger")
	}
	if logger.enabled {
		t.Fatal("Init(false) should return disabled logger")
	}
}

func TestLogToolExecution_Normal(t *testing.T) {
	tmpHome := testutil.SetupTempHome(t)
	logger, err := NewDefaultLogger(true)
	if err != nil {
		t.Fatalf("NewDefaultLogger(true) error = %v", err)
	}

	logger.LogToolExecution(
		"read_file",
		map[string]string{"path": "test.txt"},
		"file content here",
		nil,
		false,
	)

	logFiles, err := filepath.Glob(filepath.Join(tmpHome, ".xelyon", "audit", "audit_*.jsonl"))
	if err != nil || len(logFiles) == 0 {
		t.Fatal("no log file created")
	}

	data, err := os.ReadFile(logFiles[0])
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	var entry LogEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}

	if entry.Tool != "read_file" {
		t.Errorf("Tool = %s, want read_file", entry.Tool)
	}
	if entry.Args["path"] != "test.txt" {
		t.Errorf("Args = %v, want path=test.txt", entry.Args)
	}
	if entry.Output != "file content here" {
		t.Errorf("Output = %q, want %q", entry.Output, "file content here")
	}
	if !entry.Success {
		t.Error("Success should be true")
	}
	if entry.FileChanged {
		t.Error("FileChanged should be false")
	}
}

func TestLogToolExecution_WithError(t *testing.T) {
	testutil.SetupTempHome(t)
	logger, err := NewDefaultLogger(true)
	if err != nil {
		t.Fatalf("NewDefaultLogger(true) error = %v", err)
	}

	logger.LogToolExecution(
		"delete_file",
		map[string]string{"path": "missing.txt"},
		"",
		errors.New("file not found"),
		false,
	)

	logFiles, _ := filepath.Glob(filepath.Join(os.Getenv("HOME"), ".xelyon", "audit", "audit_*.jsonl"))
	data, _ := os.ReadFile(logFiles[0])

	var entry LogEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}

	if entry.Success {
		t.Error("Success should be false on error")
	}
	if entry.Error != "file not found" {
		t.Errorf("Error = %q, want %q", entry.Error, "file not found")
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
			args: map[string]string{"username": "admin", "password": "secret123"},
			expected: map[string]string{
				"username": "admin",
				"password": "[REDACTED]",
			},
		},
		{
			name: "token redaction",
			args: map[string]string{"token": "abc123xyz", "api_key": "key-123", "secret": "my-secret"},
			expected: map[string]string{
				"token":   "[REDACTED]",
				"api_key": "[REDACTED]",
				"secret":  "[REDACTED]",
			},
		},
		{
			name: "long value truncation",
			args: map[string]string{"content": strings.Repeat("a", 250)},
			expected: map[string]string{
				"content": strings.Repeat("a", 200) + "... (truncated)",
			},
		},
		{
			name: "normal values",
			args: map[string]string{"path": "/home/user/file.txt", "mode": "0644"},
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
					t.Errorf("Key %s = %s, want %s", k, result[k], expectedVal)
				}
			}
		})
	}
}

func TestLogToolExecution_OutputTruncation(t *testing.T) {
	testutil.SetupTempHome(t)
	logger, err := NewDefaultLogger(true)
	if err != nil {
		t.Fatalf("NewDefaultLogger(true) error = %v", err)
	}

	longOutput := strings.Repeat("x", 600)
	logger.LogToolExecution("read_file", map[string]string{"path": "large.txt"}, longOutput, nil, false)

	logFiles, _ := filepath.Glob(filepath.Join(os.Getenv("HOME"), ".xelyon", "audit", "audit_*.jsonl"))
	data, _ := os.ReadFile(logFiles[0])

	var entry LogEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}

	expectedOutput := strings.Repeat("x", 500) + "... (truncated)"
	if entry.Output != expectedOutput {
		t.Errorf("Output = %q, want %q", entry.Output, expectedOutput)
	}
}

func TestLogToolExecution_Concurrent(t *testing.T) {
	tmpHome := testutil.SetupTempHome(t)
	logger, err := NewDefaultLogger(true)
	if err != nil {
		t.Fatalf("NewDefaultLogger(true) error = %v", err)
	}

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

	logFiles, _ := filepath.Glob(filepath.Join(tmpHome, ".xelyon", "audit", "audit_*.jsonl"))
	if len(logFiles) == 0 {
		t.Fatal("no log file created")
	}

	file, err := os.Open(logFiles[0])
	if err != nil {
		t.Fatalf("open log file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	if lineCount != 100 {
		t.Errorf("log entry count = %d, want 100", lineCount)
	}
}

func TestLogToolExecution_Disabled(t *testing.T) {
	tmpHome := testutil.SetupTempHome(t)
	logDir := filepath.Join(tmpHome, ".xelyon", "audit")
	logger := NewDisabledLogger()

	logger.LogToolExecution("test_tool", map[string]string{"key": "value"}, "output", nil, false)

	logFiles, _ := filepath.Glob(filepath.Join(logDir, "audit_*.jsonl"))
	if len(logFiles) > 0 {
		data, _ := os.ReadFile(logFiles[0])
		if len(data) > 0 {
			t.Error("disabled logger should not write logs")
		}
	}
}
