package dev

import (
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestTruncateWithFile_Short(t *testing.T) {
	input := "short output"
	result := TruncateWithFile(input)
	if result != input {
		t.Errorf("TruncateWithFile() should return input as-is for short strings, got %q", result)
	}
}

func TestTruncateWithFile_ExactLimit(t *testing.T) {
	input := strings.Repeat("a", config.OutputTruncateLen)
	result := TruncateWithFile(input)
	if result != input {
		t.Error("TruncateWithFile() should return input as-is when exactly at limit")
	}
}

func TestTruncateWithFile_OverLimit(t *testing.T) {
	input := strings.Repeat("x", config.OutputTruncateLen+5000)
	result := TruncateWithFile(input)

	// 先頭1000文字が含まれること
	head := strings.Repeat("x", 1000)
	if !strings.HasPrefix(result, head) {
		t.Error("TruncateWithFile() should start with first 1000 chars of input")
	}

	// 末尾500文字が含まれること
	tail := strings.Repeat("x", 500)
	if !strings.HasSuffix(result, tail) {
		t.Error("TruncateWithFile() should end with last 500 chars of input")
	}

	// ファイルパスが含まれること
	if !strings.Contains(result, "/tmp/xelyon_output_") {
		t.Error("TruncateWithFile() should contain file path")
	}

	// truncated メッセージが含まれること
	if !strings.Contains(result, "truncated") {
		t.Error("TruncateWithFile() should contain 'truncated' message")
	}

	// 文字数が含まれること
	if !strings.Contains(result, "25000 chars") {
		t.Errorf("TruncateWithFile() should contain total char count, got: %s", result)
	}
}

func TestTruncateWithFile_FileSaved(t *testing.T) {
	// 25000文字のテストデータ（先頭と末尾で区別可能にする）
	input := "HEAD" + strings.Repeat("m", config.OutputTruncateLen+4992) + "TAIL"
	result := TruncateWithFile(input)

	// ファイルパスを抽出
	idx := strings.Index(result, "/tmp/xelyon_output_")
	if idx == -1 {
		t.Fatal("TruncateWithFile() should contain file path")
	}
	// パスの末尾を探す
	end := strings.Index(result[idx:], ")")
	if end == -1 {
		t.Fatal("could not find end of file path")
	}
	filePath := result[idx : idx+end]

	// ファイルが存在すること
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("saved file should be readable: %v", err)
	}
	defer os.Remove(filePath)

	// ファイル内容が元の出力と一致すること
	if string(data) != input {
		t.Error("saved file content should match original input")
	}

	// ファイルパーミッションが0600であること
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("could not stat file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("file permission should be 0600, got %v", info.Mode().Perm())
	}
}

func TestTruncateWithFile_HeadAndTailContent(t *testing.T) {
	// 先頭と末尾が異なるデータで確認
	head := strings.Repeat("H", 1000)
	middle := strings.Repeat("M", config.OutputTruncateLen)
	tail := strings.Repeat("T", 500)
	input := head + middle + tail

	result := TruncateWithFile(input)

	// 先頭1000文字が正しいこと
	if !strings.HasPrefix(result, head) {
		t.Error("TruncateWithFile() head portion should be first 1000 chars")
	}

	// 末尾500文字が正しいこと
	if !strings.HasSuffix(result, tail) {
		t.Error("TruncateWithFile() tail portion should be last 500 chars")
	}

	// 保存されたファイルをクリーンアップ
	idx := strings.Index(result, "/tmp/xelyon_output_")
	if idx != -1 {
		end := strings.Index(result[idx:], ")")
		if end != -1 {
			os.Remove(result[idx : idx+end])
		}
	}
}

func TestTruncateWithFile_Empty(t *testing.T) {
	result := TruncateWithFile("")
	if result != "" {
		t.Errorf("TruncateWithFile() should return empty string for empty input, got %q", result)
	}
}
