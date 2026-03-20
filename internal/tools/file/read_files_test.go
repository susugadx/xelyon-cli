package file

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestExecuteReadFiles_Normal(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	testutil.CreateTempFile(t, tmpDir, "a.go", "package main\nfunc main() {}")
	testutil.CreateTempFile(t, tmpDir, "b.go", "package util\nfunc helper() {}")

	paths := []string{
		filepath.Join(tmpDir, "a.go"),
		filepath.Join(tmpDir, "b.go"),
	}

	output := ExecuteReadFiles(paths)

	if !strings.Contains(output, "package main") {
		t.Errorf("Expected output to contain 'package main', got: %s", output)
	}
	if !strings.Contains(output, "package util") {
		t.Errorf("Expected output to contain 'package util', got: %s", output)
	}
	// ファイルヘッダーの確認
	if !strings.Contains(output, "📄 File:") {
		t.Errorf("Expected output to contain file headers, got: %s", output)
	}
}

func TestExecuteReadFiles_ParallelOrder(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	testutil.CreateTempFile(t, tmpDir, "first.txt", "first content")
	testutil.CreateTempFile(t, tmpDir, "second.txt", "second content")
	testutil.CreateTempFile(t, tmpDir, "third.txt", "third content")

	paths := []string{
		filepath.Join(tmpDir, "third.txt"),
		filepath.Join(tmpDir, "first.txt"),
		filepath.Join(tmpDir, "second.txt"),
	}

	output := ExecuteReadFiles(paths)

	prevIndex := -1
	for _, path := range paths {
		header := "📄 File: " + path
		idx := strings.Index(output, header)
		if idx < 0 {
			t.Fatalf("expected output to contain header %q, got: %s", header, output)
		}
		if idx <= prevIndex {
			t.Fatalf("expected headers to follow input order, got: %s", output)
		}
		prevIndex = idx
	}
}

func TestExecuteReadFiles_EmptyPaths(t *testing.T) {
	output := ExecuteReadFiles([]string{})
	if !strings.Contains(output, "Error: paths is empty") {
		t.Errorf("Expected 'paths is empty' error, got: %s", output)
	}
}

func TestExecuteReadFiles_TooManyPaths(t *testing.T) {
	paths := make([]string, 11)
	for i := range paths {
		paths[i] = fmt.Sprintf("file%d.go", i)
	}

	output := ExecuteReadFiles(paths)
	if !strings.Contains(output, "too many paths") {
		t.Errorf("Expected 'too many paths' error, got: %s", output)
	}
}

func TestExecuteReadFiles_NonExistentMixed(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	testutil.CreateTempFile(t, tmpDir, "exists.go", "package exists")

	paths := []string{
		filepath.Join(tmpDir, "exists.go"),
		filepath.Join(tmpDir, "nonexistent.go"),
	}

	output := ExecuteReadFiles(paths)

	if !strings.Contains(output, "package exists") {
		t.Errorf("Expected output to contain existing file content, got: %s", output)
	}
	if !strings.Contains(output, "Error") {
		t.Errorf("Expected output to contain error for nonexistent file, got: %s", output)
	}
}

func TestExecuteReadFiles_LineRange(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	testutil.CreateTempFile(t, tmpDir, "lines.txt", "line1\nline2\nline3\nline4\nline5")

	entry := filepath.Join(tmpDir, "lines.txt") + ":2-4"
	output := ExecuteReadFiles([]string{entry})

	if !strings.Contains(output, "line2") || !strings.Contains(output, "line4") {
		t.Errorf("Expected output to contain lines 2-4, got: %s", output)
	}
	if strings.Contains(output, "1: line1") {
		t.Errorf("Output should NOT contain line 1, got: %s", output)
	}
}

func TestExecuteReadFiles_StartLineOnly(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "start.txt", strings.Join(lines, "\n"))

	entry := filepath.Join(tmpDir, "start.txt") + ":8"
	output := ExecuteReadFiles([]string{entry})

	if !strings.Contains(output, "8: line8") {
		t.Errorf("Expected output to start at line 8, got: %s", output)
	}
	if strings.Contains(output, "7: line7") {
		t.Errorf("Output should NOT contain line 7, got: %s", output)
	}
}

func TestParsePath_NoRange(t *testing.T) {
	path, start, end := parsePath("internal/tools/file/read.go")
	if path != "internal/tools/file/read.go" || start != 0 || end != 0 {
		t.Errorf("Expected (internal/tools/file/read.go, 0, 0), got (%s, %d, %d)", path, start, end)
	}
}

func TestParsePath_StartOnly(t *testing.T) {
	path, start, end := parsePath("file.go:10")
	if path != "file.go" || start != 10 || end != 0 {
		t.Errorf("Expected (file.go, 10, 0), got (%s, %d, %d)", path, start, end)
	}
}

func TestParsePath_StartEnd(t *testing.T) {
	path, start, end := parsePath("file.go:10-20")
	if path != "file.go" || start != 10 || end != 20 {
		t.Errorf("Expected (file.go, 10, 20), got (%s, %d, %d)", path, start, end)
	}
}

func TestParsePath_InvalidSuffix(t *testing.T) {
	// コロン後が数字でない場合は全体をパスとして扱う
	path, start, end := parsePath("file.go:abc")
	if path != "file.go:abc" || start != 0 || end != 0 {
		t.Errorf("Expected (file.go:abc, 0, 0), got (%s, %d, %d)", path, start, end)
	}
}

func TestParsePath_InvalidRange(t *testing.T) {
	path, start, end := parsePath("file.go:abc-def")
	if path != "file.go:abc-def" || start != 0 || end != 0 {
		t.Errorf("Expected (file.go:abc-def, 0, 0), got (%s, %d, %d)", path, start, end)
	}
}

func TestPerFileBudget(t *testing.T) {
	tests := []struct {
		n    int
		want int
	}{
		{1, DefaultFullLines}, // 500/1=500, capped to DefaultFullLines(500)
		{2, 250},              // 500/2=250
		{3, 166},              // 500/3=166, within range
		{5, 100},              // 500/5=100, within range
		{6, 83},               // 500/6=83
		{10, 50},              // 500/10=50
		{20, 30},              // 500/20=25, floored to 30
	}
	for _, tt := range tests {
		got := perFileBudget(tt.n)
		if got != tt.want {
			t.Errorf("perFileBudget(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

// generateLines は指定行数のテスト用コンテンツを生成する
func generateLines(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i+1)
	}
	return strings.Join(lines, "\n")
}

func TestExecuteReadFiles_BudgetOutlineMode(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	// 200行のファイルを6個作成（budget=83, 200 > 83 → outline モード）
	paths := make([]string, 6)
	for i := range paths {
		name := fmt.Sprintf("f%d.txt", i)
		testutil.CreateTempFile(t, tmpDir, name, generateLines(200))
		paths[i] = filepath.Join(tmpDir, name)
	}

	output := ExecuteReadFiles(paths)

	// outline モードが適用される（ガイドメッセージが含まれる）
	if !strings.Contains(output, "Use start_line/end_line") {
		t.Error("Expected outline mode with guide message")
	}
	// 先頭行が含まれる
	if !strings.Contains(output, "1: line1") {
		t.Error("Expected head lines to be present")
	}
	// 末尾セクションが含まれる
	if !strings.Contains(output, "Last lines") {
		t.Error("Expected 'Last lines' section")
	}
}

func TestExecuteReadFiles_SmallFilesFullContent(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	// 50行のファイルを4個作成（budget=125, 50 < 125 → 全文返却）
	paths := make([]string, 4)
	for i := range paths {
		name := fmt.Sprintf("f%d.txt", i)
		testutil.CreateTempFile(t, tmpDir, name, generateLines(50))
		paths[i] = filepath.Join(tmpDir, name)
	}

	output := ExecuteReadFiles(paths)

	// 全文が返却される
	if !strings.Contains(output, "50: line50") {
		t.Error("Expected full content with line 50")
	}
	// outline ガイドメッセージが含まれない
	if strings.Contains(output, "Use start_line/end_line") {
		t.Error("Small files should NOT have outline guide message")
	}
}

func TestExecuteReadFiles_BudgetExplicitRangePriority(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	// 200行のファイルを6個作成（budget=83）
	paths := make([]string, 6)
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("f%d.txt", i)
		testutil.CreateTempFile(t, tmpDir, name, generateLines(200))
		paths[i] = filepath.Join(tmpDir, name)
	}
	// 6番目は明示的に行範囲指定（budget を無視して範囲を返す）
	testutil.CreateTempFile(t, tmpDir, "f5.txt", generateLines(200))
	paths[5] = filepath.Join(tmpDir, "f5.txt") + ":1-120"

	output := ExecuteReadFiles(paths)

	// 明示指定のファイルは120行目まで含まれる
	if !strings.Contains(output, "120: line120") {
		t.Error("Expected explicit range file to include line 120")
	}
}

// ── ExecuteReadFilesWithBudget regression test ──
// 自動 batch merge で DefaultFullLines 指定時、3件以上でも単発 read と同じ内容量を返すことを検証。

func TestExecuteReadFilesWithBudget_FullBudgetPreservesContent(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	// 180行のファイル3個: perFileBudget(3)=166 → outline になる
	// DefaultFullLines=500 なら 180 < 500 で全文返却
	paths := make([]string, 3)
	for i := range paths {
		name := fmt.Sprintf("f%d.go", i)
		testutil.CreateTempFile(t, tmpDir, name, generateLines(180))
		paths[i] = filepath.Join(tmpDir, name)
	}

	// perFileBudget(3) = 166: 180 > 166 → outline
	normalOutput := ExecuteReadFiles(paths)
	if !strings.Contains(normalOutput, "Use start_line/end_line") {
		t.Fatal("Expected normal 3-file read to use outline mode for 180-line files")
	}

	// budgetOverride=DefaultFullLines(500): 180 < 500 → 全文
	fullOutput := ExecuteReadFilesWithBudget(common.DefaultOutput(), paths, DefaultFullLines)
	if strings.Contains(fullOutput, "Use start_line/end_line") {
		t.Error("ExecuteReadFilesWithBudget with DefaultFullLines should NOT use outline for 180-line files")
	}
	// 全文が返却される
	if !strings.Contains(fullOutput, "180: line180") {
		t.Error("Expected full content with line 180")
	}
}

// ── ExecuteReadFilesWithRuntime tests ──

func TestExecuteReadFilesWithRuntime_NilConfigCache(t *testing.T) {
	// nil config/cache は ExecuteReadFilesWithBudget と同等の動作
	setupTestMocks(t)
	tmpDir := t.TempDir()

	testutil.CreateTempFile(t, tmpDir, "a.go", "package main\nfunc main() {}")
	paths := []string{filepath.Join(tmpDir, "a.go")}

	output := ExecuteReadFilesWithRuntime(common.DefaultOutput(), nil, nil, paths, DefaultFullLines)
	if !strings.Contains(output, "package main") {
		t.Errorf("Expected output to contain file content, got: %s", output)
	}
}

func TestExecuteReadFilesWithRuntime_ConsistentWithSingleRead(t *testing.T) {
	// auto-merge 経由の batch read が single read と同等の内容量を返すことを検証
	setupTestMocks(t)
	tmpDir := t.TempDir()

	// 150 行のファイルを 3 個作成
	paths := make([]string, 3)
	for i := range paths {
		name := fmt.Sprintf("f%d.go", i)
		testutil.CreateTempFile(t, tmpDir, name, generateLines(150))
		paths[i] = filepath.Join(tmpDir, name)
	}

	// single read: DefaultFullLines(500) → 150 < 500 → 全文
	singleOutput := ExecuteReadFileWithRuntime(common.DefaultOutput(), nil, nil, paths[0], 0, 0)
	if strings.Contains(singleOutput, "Use start_line/end_line") {
		t.Fatal("single read should show full content for 150-line file")
	}

	// batch read with runtime: DefaultFullLines(500) → 150 < 500 → 全文
	batchOutput := ExecuteReadFilesWithRuntime(common.DefaultOutput(), nil, nil, paths, DefaultFullLines)
	if strings.Contains(batchOutput, "Use start_line/end_line") {
		t.Error("batch read with DefaultFullLines should NOT use outline for 150-line files")
	}
	// 全ファイルの全行が含まれる
	if !strings.Contains(batchOutput, "150: line150") {
		t.Error("Expected full content with line 150")
	}
}
