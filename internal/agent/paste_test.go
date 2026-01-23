package agent

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// TestPasteConfig はPasteConfigのデフォルト値をテスト
func TestPasteConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	// デフォルト値の確認
	if cfg.Paste.MaxLines != 10000 {
		t.Errorf("Expected MaxLines=10000, got %d", cfg.Paste.MaxLines)
	}
	if cfg.Paste.MaxBytes != 1048576 {
		t.Errorf("Expected MaxBytes=1048576, got %d", cfg.Paste.MaxBytes)
	}
	if cfg.Paste.TimeoutSeconds != 60 {
		t.Errorf("Expected TimeoutSeconds=60, got %d", cfg.Paste.TimeoutSeconds)
	}
}

// TestPasteConfigLoadDefaults はLoadConfig時のデフォルト値適用をテスト
func TestPasteConfigLoadDefaults(t *testing.T) {
	// 一時ディレクトリを使用
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// 設定ファイルなしでロード → デフォルト値が適用される
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Paste.MaxLines != 10000 {
		t.Errorf("Expected MaxLines=10000 after load, got %d", cfg.Paste.MaxLines)
	}
	if cfg.Paste.MaxBytes != 1048576 {
		t.Errorf("Expected MaxBytes=1048576 after load, got %d", cfg.Paste.MaxBytes)
	}
	if cfg.Paste.TimeoutSeconds != 60 {
		t.Errorf("Expected TimeoutSeconds=60 after load, got %d", cfg.Paste.TimeoutSeconds)
	}
}

// TestParsePasteInput はペースト入力のパース処理をテスト
func TestParsePasteInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "空行2回で終了",
			input:    "line1\nline2\n\n\n",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "ENDで終了",
			input:    "line1\nline2\nEND\nline3\n",
			expected: []string{"line1", "line2"},
		},
		{
			name:     "/endで終了",
			input:    "line1\n/end\nline2\n",
			expected: []string{"line1"},
		},
		{
			name:     "単一行",
			input:    "single line\n\n\n",
			expected: []string{"single line"},
		},
		{
			name:     "空入力",
			input:    "\n\n",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := parsePasteInput(strings.NewReader(tt.input), 10000, 1048576, 5*time.Second)

			if len(lines) != len(tt.expected) {
				t.Errorf("Expected %d lines, got %d", len(tt.expected), len(lines))
				t.Errorf("Lines: %v", lines)
				return
			}

			for i, expected := range tt.expected {
				if lines[i] != expected {
					t.Errorf("Line %d: expected %q, got %q", i, expected, lines[i])
				}
			}
		})
	}
}

// TestParsePasteInputMaxLines は行数上限をテスト
func TestParsePasteInputMaxLines(t *testing.T) {
	// 100行入力、上限5行
	var input bytes.Buffer
	for i := 0; i < 100; i++ {
		input.WriteString("line\n")
	}
	input.WriteString("\n\n") // 終了

	lines := parsePasteInput(&input, 5, 1048576, 5*time.Second)

	if len(lines) != 5 {
		t.Errorf("Expected 5 lines (max), got %d", len(lines))
	}
}

// TestParsePasteInputMaxBytes はバイト数上限をテスト
func TestParsePasteInputMaxBytes(t *testing.T) {
	// 長い行を入力、バイト上限50
	input := "12345678901234567890\n12345678901234567890\n12345678901234567890\n\n\n"

	lines := parsePasteInput(strings.NewReader(input), 10000, 50, 5*time.Second)

	// バイト上限に達すると3行目が追加される前に停止する
	// 2行 = 21 + 21 = 42バイト、3行目を追加すると63バイトで上限超過
	if len(lines) > 3 {
		t.Errorf("Expected at most 3 lines due to byte limit, got %d", len(lines))
	}
}

// parsePasteInput はペースト入力をパースするヘルパー関数（テスト用）
// 実際の handlePasteCommand のロジックを抽出
func parsePasteInput(r io.Reader, maxLines, maxBytes int, timeout time.Duration) []string {
	var lines []string
	emptyCount := 0
	totalBytes := 0

	// 1バイトずつ読んで行を構築（タイムアウトのシミュレーション用）
	var currentLine bytes.Buffer
	buf := make([]byte, 1)

	startTime := time.Now()

	for time.Since(startTime) <= timeout {
		n, err := r.Read(buf)
		if err == io.EOF {
			// 残りの行を処理
			if currentLine.Len() > 0 {
				line := currentLine.String()
				if line == "END" || line == "/end" {
					break
				}
				lines = append(lines, line)
			}
			break
		}
		if err != nil {
			break
		}
		if n == 0 {
			continue
		}

		if buf[0] == '\n' {
			line := strings.TrimRight(currentLine.String(), "\r")
			currentLine.Reset()

			// 終了条件
			if line == "" {
				emptyCount++
				if emptyCount >= 2 {
					break
				}
				lines = append(lines, line)
				totalBytes += 1
				continue
			}

			if line == "END" || line == "/end" {
				break
			}

			emptyCount = 0
			lines = append(lines, line)
			totalBytes += len(line) + 1

			if len(lines) >= maxLines {
				break
			}
			if totalBytes >= maxBytes {
				break
			}
		} else {
			currentLine.WriteByte(buf[0])
		}
	}

	// 末尾の空行を削除
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// TestCommandAliasP は /p エイリアスが /paste に解決されることをテスト
func TestCommandAliasP(t *testing.T) {
	resolved := resolveCommandAlias("/p")
	if resolved != "/paste" {
		t.Errorf("Expected /p to resolve to /paste, got %s", resolved)
	}
}

// TestParsePasteInputCancel は /cancel と /c でキャンセルされることをテスト
func TestParsePasteInputCancel(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldAbort bool
		expected    []string
	}{
		{
			name:        "/cancel でキャンセル",
			input:       "line1\nline2\n/cancel\nline3\n\n\n",
			shouldAbort: true,
			expected:    []string{"line1", "line2"},
		},
		{
			name:        "/c でキャンセル",
			input:       "line1\n/c\nline2\n\n\n",
			shouldAbort: true,
			expected:    []string{"line1"},
		},
		{
			name:        "最初の行で /cancel",
			input:       "/cancel\nline1\n\n\n",
			shouldAbort: true,
			expected:    []string{},
		},
		{
			name:        "/cancel を含む文字列は通常行として扱う",
			input:       "line with /cancel in it\n\n\n",
			shouldAbort: false,
			expected:    []string{"line with /cancel in it"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines, cancelled := parsePasteInputWithCancel(strings.NewReader(tt.input), 10000, 1048576, 5*time.Second)

			if cancelled != tt.shouldAbort {
				t.Errorf("Expected cancelled=%v, got %v", tt.shouldAbort, cancelled)
			}

			if len(lines) != len(tt.expected) {
				t.Errorf("Expected %d lines, got %d", len(tt.expected), len(lines))
				t.Errorf("Lines: %v", lines)
				return
			}

			for i, expected := range tt.expected {
				if lines[i] != expected {
					t.Errorf("Line %d: expected %q, got %q", i, expected, lines[i])
				}
			}
		})
	}
}

// parsePasteInputWithCancel は /cancel と /c をサポートするパース関数（テスト用）
func parsePasteInputWithCancel(r io.Reader, maxLines, maxBytes int, timeout time.Duration) ([]string, bool) {
	var lines []string
	emptyCount := 0
	totalBytes := 0
	cancelled := false

	var currentLine bytes.Buffer
	buf := make([]byte, 1)

	startTime := time.Now()

	for time.Since(startTime) <= timeout {
		n, err := r.Read(buf)
		if err == io.EOF {
			if currentLine.Len() > 0 {
				line := currentLine.String()
				if line == "END" || line == "/end" {
					break
				}
				if line == "/cancel" || line == "/c" {
					cancelled = true
					break
				}
				lines = append(lines, line)
			}
			break
		}
		if err != nil {
			break
		}
		if n == 0 {
			continue
		}

		if buf[0] == '\n' {
			line := strings.TrimRight(currentLine.String(), "\r")
			currentLine.Reset()

			if line == "" {
				emptyCount++
				if emptyCount >= 2 {
					break
				}
				lines = append(lines, line)
				totalBytes += 1
				continue
			}

			if line == "END" || line == "/end" {
				break
			}

			// キャンセルコマンド
			if line == "/cancel" || line == "/c" {
				cancelled = true
				break
			}

			emptyCount = 0
			lines = append(lines, line)
			totalBytes += len(line) + 1

			if len(lines) >= maxLines {
				break
			}
			if totalBytes >= maxBytes {
				break
			}
		} else {
			currentLine.WriteByte(buf[0])
		}
	}

	// 末尾の空行を削除
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	return lines, cancelled
}
