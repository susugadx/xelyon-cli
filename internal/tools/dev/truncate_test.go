package dev

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestTruncateWithFile_Empty(t *testing.T) {
	result := TruncateWithFile("")
	if result != "" {
		t.Errorf("TruncateWithFile() should return empty string for empty input, got %q", result)
	}
}

func TestTruncateWithFile_OverLimit_Format(t *testing.T) {
	// テスト用に cwd を一時ディレクトリに変更
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// 50行のテストデータを生成（各行"line_NNN\n"）
	var sb strings.Builder
	totalLines := 200
	for i := 1; i <= totalLines; i++ {
		fmt.Fprintf(&sb, "line_%03d\n", i)
	}
	// OutputTruncateLen を超えるようにパディング
	padding := strings.Repeat("x", config.OutputTruncateLen)
	input := sb.String() + padding

	result := TruncateWithFile(input)

	// "Output saved:" で始まること
	if !strings.HasPrefix(result, "Output saved: .xelyon/artifacts/output_") {
		t.Errorf("should start with 'Output saved:', got: %s", result[:80])
	}

	// ファイルメタデータが含まれること
	if !strings.Contains(result, "lines,") {
		t.Error("should contain line count")
	}
	if !strings.Contains(result, "KB)") {
		t.Error("should contain size in KB")
	}

	// "Last 30 lines:" が含まれること
	if !strings.Contains(result, "Last 30 lines:") {
		t.Error("should contain 'Last 30 lines:'")
	}

	// read_file ヒントが含まれること
	if !strings.Contains(result, "Use `read_file .xelyon/artifacts/output_") {
		t.Error("should contain read_file hint")
	}
	if !strings.Contains(result, ":1-50` to read specific sections.") {
		t.Error("should contain read_file range hint")
	}
}

func TestTruncateWithFile_FileSaved(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	input := "HEAD" + strings.Repeat("m", config.OutputTruncateLen+4992) + "TAIL"
	result := TruncateWithFile(input)

	// ファイルパスを抽出
	prefix := ".xelyon/artifacts/output_"
	idx := strings.Index(result, prefix)
	if idx == -1 {
		t.Fatal("should contain artifact file path")
	}
	end := strings.Index(result[idx:], " ")
	if end == -1 {
		t.Fatal("could not find end of file path")
	}
	relPath := result[idx : idx+end]

	// ファイルが存在し内容が一致すること
	data, err := os.ReadFile(relPath)
	if err != nil {
		t.Fatalf("saved file should be readable: %v", err)
	}
	if string(data) != input {
		t.Error("saved file content should match original input")
	}

	// パーミッションが0600であること
	info, err := os.Stat(relPath)
	if err != nil {
		t.Fatalf("could not stat file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("file permission should be 0600, got %v", info.Mode().Perm())
	}
}

func TestTruncateWithFile_Last30Lines(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// 100行 + パディングで閾値超過
	var sb strings.Builder
	for i := 1; i <= 100; i++ {
		fmt.Fprintf(&sb, "LINE_%03d\n", i)
	}
	sb.WriteString(strings.Repeat("P", config.OutputTruncateLen))
	input := sb.String()

	result := TruncateWithFile(input)

	// 末尾にはパディングの最後の部分が含まれる
	// "Last 30 lines:" セクションが存在すること
	if !strings.Contains(result, "Last 30 lines:") {
		t.Fatal("should contain 'Last 30 lines:'")
	}
}

func TestLastNLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		n     int
		want  string
	}{
		{"fewer than n", "a\nb\nc", 5, "a\nb\nc"},
		{"exact n", "a\nb\nc", 3, "a\nb\nc"},
		{"more than n", "a\nb\nc\nd\ne", 3, "c\nd\ne"},
		{"single line", "hello", 3, "hello"},
		{"empty", "", 3, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lastNLines(tt.input, tt.n)
			if got != tt.want {
				t.Errorf("lastNLines(%q, %d) = %q, want %q", tt.input, tt.n, got, tt.want)
			}
		})
	}
}

func TestCleanupArtifacts(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	dir := filepath.Join(tmpDir, artifactsDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	// 古いファイル（25時間前）
	oldFile := filepath.Join(dir, "output_old.txt")
	if err := os.WriteFile(oldFile, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-25 * time.Hour)
	os.Chtimes(oldFile, oldTime, oldTime)

	// 新しいファイル（1時間前）
	newFile := filepath.Join(dir, "output_new.txt")
	if err := os.WriteFile(newFile, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	newTime := time.Now().Add(-1 * time.Hour)
	os.Chtimes(newFile, newTime, newTime)

	CleanupArtifacts()

	// 古いファイルは削除されていること
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("old artifact should be deleted")
	}

	// 新しいファイルは残っていること
	if _, err := os.Stat(newFile); err != nil {
		t.Error("new artifact should still exist")
	}
}

func TestCleanupArtifacts_NoDir(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// ディレクトリが存在しなくてもパニックしないこと
	CleanupArtifacts()
}

func TestCheckGitignore_Present(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// .xelyon/ を含む .gitignore
	os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("node_modules/\n.xelyon/\n"), 0644)

	// パニックしないこと（log出力はなし）
	checkGitignore()
}

func TestCheckGitignore_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// .gitignore なし → 警告が出るがパニックしないこと
	checkGitignore()
}

func TestCheckGitignore_WithoutXelyon(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// .xelyon/ を含まない .gitignore
	os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("node_modules/\n"), 0644)

	// パニックしないこと（log出力あり）
	checkGitignore()
}
