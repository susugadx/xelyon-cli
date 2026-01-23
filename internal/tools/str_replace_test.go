package tools

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteStrReplace_ExactMatch(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "line1\nline2\nline3"

	testutil.CreateTempFile(t, tmpDir, "test.txt", originalContent)

	// 文字列置換（完全一致）
	output, backupPath, err := executeStrReplace(testFile, "line2", "REPLACED", "", "")

	// 検証
	if err != nil {
		t.Fatalf("executeStrReplace failed: %v", err)
	}

	if !strings.Contains(output, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// ファイル内容確認
	expectedContent := "line1\nREPLACED\nline3"
	testutil.AssertFileContent(t, testFile, expectedContent)

	// バックアップ確認
	if backupPath == "" {
		t.Error("Backup path should not be empty")
	}
	testutil.AssertFileExists(t, backupPath)
	testutil.AssertFileContent(t, backupPath, originalContent)
}

func TestExecuteStrReplace_MultipleMatches(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "foo\nbar\nfoo\nbaz"

	testutil.CreateTempFile(t, tmpDir, "test.txt", originalContent)

	// 複数マッチする文字列を置換しようとする
	output, _, err := executeStrReplace(testFile, "foo", "REPLACED", "", "")

	// 検証
	if err != nil {
		t.Fatalf("executeStrReplace should not return error: %v", err)
	}

	// エラーメッセージ確認（2回出現する）
	if !strings.Contains(output, "Error: old_str appears 2 times") {
		t.Errorf("Expected multiple matches error, got: %s", output)
	}

	// 追加要件: Candidates/lines/Next actions/IMPORTANT
	if !strings.Contains(output, "Candidates") {
		t.Errorf("Expected Candidates section, got: %s", output)
	}
	if !strings.Contains(output, "lines") {
		t.Errorf("Expected line ranges in error message, got: %s", output)
	}
	if !strings.Contains(output, "Next actions") {
		t.Errorf("Expected Next actions section, got: %s", output)
	}
	if !strings.Contains(output, "IMPORTANT") {
		t.Errorf("Expected IMPORTANT notice, got: %s", output)
	}

	// ファイルは変更されていないべき
	testutil.AssertFileContent(t, testFile, originalContent)
}

func TestExecuteStrReplace_NotFound(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "line1\nline2\nline3"

	testutil.CreateTempFile(t, tmpDir, "test.txt", originalContent)

	// 存在しない文字列を置換しようとする
	output, _, err := executeStrReplace(testFile, "nonexistent", "REPLACED", "", "")

	// 検証
	if err != nil {
		t.Fatalf("executeStrReplace should not return error: %v", err)
	}

	if !strings.Contains(output, "Error: old_str not found") {
		t.Errorf("Expected 'not found' error, got: %s", output)
	}

	// ファイルは変更されていないべき
	testutil.AssertFileContent(t, testFile, originalContent)
}

func TestExecuteStrReplace_UserCancelled(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, false)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "line1\nline2\nline3"

	testutil.CreateTempFile(t, tmpDir, "test.txt", originalContent)

	// 置換実行（キャンセル）
	output, backupPath, err := executeStrReplace(testFile, "line2", "REPLACED", "", "")

	// 検証
	if err != nil {
		t.Fatalf("executeStrReplace should not error on cancel: %v", err)
	}

	if !strings.Contains(output, "[CANCELLED]") {
		t.Errorf("Expected cancellation message, got: %s", output)
	}

	// ファイルは変更されていないべき
	testutil.AssertFileContent(t, testFile, originalContent)

	// バックアップは作成されていないべき
	if backupPath != "" {
		t.Errorf("Backup path should be empty on cancel, got: %s", backupPath)
	}
}

func TestExecuteStrReplace_EmptyPath(t *testing.T) {
	setupTestMocks(t)

	// 空パス
	output, _, err := executeStrReplace("", "old", "new", "", "")

	// 検証
	if err != nil {
		t.Fatalf("executeStrReplace should not return error: %v", err)
	}

	if !strings.Contains(output, "Error: path is required") {
		t.Errorf("Expected 'path is required' error, got: %s", output)
	}
}

func TestExecuteStrReplace_EmptyOldStr(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	testutil.CreateTempFile(t, tmpDir, "test.txt", "content")

	// 空のold_str（レンジ指定なし）
	output, _, err := executeStrReplace(testFile, "", "new", "", "")

	// 検証
	if err != nil {
		t.Fatalf("executeStrReplace should not return error: %v", err)
	}

	if !strings.Contains(output, "Error: old_str is required") {
		t.Errorf("Expected 'old_str is required' error, got: %s", output)
	}
}

func TestExecuteStrReplace_MultilineReplacement(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "function foo() {\n  return 42;\n}\n\nother code"

	testutil.CreateTempFile(t, tmpDir, "test.txt", originalContent)

	// 複数行の置換
	oldStr := "function foo() {\n  return 42;\n}"
	newStr := "function foo() {\n  return 100;\n}"

	output, backupPath, err := executeStrReplace(testFile, oldStr, newStr, "", "")

	// 検証
	if err != nil {
		t.Fatalf("executeStrReplace failed: %v", err)
	}

	if !strings.Contains(output, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// ファイル内容確認
	expectedContent := "function foo() {\n  return 100;\n}\n\nother code"
	testutil.AssertFileContent(t, testFile, expectedContent)

	// バックアップ確認
	testutil.AssertFileExists(t, backupPath)
	testutil.AssertFileContent(t, backupPath, originalContent)
}

func TestExecuteStrReplace_WhitespaceNormalization(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	// ファイル内にはタブ文字が含まれる
	originalContent := "function test() {\n\treturn true;\n}"

	testutil.CreateTempFile(t, tmpDir, "test.txt", originalContent)

	// old_strはスペースで指定（正規化マッチングが動作するべき）
	oldStr := "function test() {\n    return true;\n}"
	newStr := "function test() {\n    return false;\n}"

	output, _, err := executeStrReplace(testFile, oldStr, newStr, "", "")

	// 検証
	if err != nil {
		t.Fatalf("executeStrReplace failed: %v", err)
	}

	// 正規化マッチングで成功するべき
	if !strings.Contains(output, "Successfully replaced") {
		t.Errorf("Expected success with normalized matching, got: %s", output)
	}

	// ファイルが変更されていることを確認
	content := testutil.ReadFile(t, testFile)
	if content == originalContent {
		t.Error("File should be modified by str_replace")
	}
}

func TestExecuteStrReplace_LineRangeReplacement_Success(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "a\nb\nc\nd\ne"
	testutil.CreateTempFile(t, tmpDir, "test.txt", originalContent)

	// 2-4行目を2行に置換
	newStr := "X\nY"
	output, backupPath, err := executeStrReplace(testFile, "", newStr, "2", "4")
	if err != nil {
		t.Fatalf("executeStrReplace failed: %v", err)
	}

	if !strings.Contains(output, "Successfully replaced lines 2-4") {
		t.Errorf("Expected line-range success message, got: %s", output)
	}

	expected := "a\nX\nY\ne"
	testutil.AssertFileContent(t, testFile, expected)

	if backupPath == "" {
		t.Fatal("Backup path should not be empty")
	}
	testutil.AssertFileExists(t, backupPath)
	testutil.AssertFileContent(t, backupPath, originalContent)
}

func TestExecuteStrReplace_LargeFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// 大きなファイル（100行）を作成
	var lines []string
	for i := 1; i <= 100; i++ {
		if i == 50 {
			lines = append(lines, "TARGET_LINE")
		} else {
			lines = append(lines, "line "+string(rune(i)))
		}
	}
	originalContent := strings.Join(lines, "\n")

	testutil.CreateTempFile(t, tmpDir, "large.txt", originalContent)

	// 50行目を置換
	output, _, err := executeStrReplace(testFile, "TARGET_LINE", "REPLACED_LINE", "", "")

	// 検証
	if err != nil {
		t.Fatalf("executeStrReplace failed: %v", err)
	}

	if !strings.Contains(output, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// 置換されていることを確認
	content := testutil.ReadFile(t, testFile)
	if !strings.Contains(content, "REPLACED_LINE") {
		t.Error("File should contain REPLACED_LINE")
	}
	if strings.Contains(content, "TARGET_LINE") {
		t.Error("File should not contain TARGET_LINE")
	}
}

func TestParseLineRange(t *testing.T) {
	tests := []struct {
		name      string
		startStr  string
		endStr    string
		wantStart int
		wantEnd   int
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid range 1-5",
			startStr:  "1",
			endStr:    "5",
			wantStart: 1,
			wantEnd:   5,
			wantErr:   false,
		},
		{
			name:      "valid range 10-20",
			startStr:  "10",
			endStr:    "20",
			wantStart: 10,
			wantEnd:   20,
			wantErr:   false,
		},
		{
			name:      "valid single line",
			startStr:  "5",
			endStr:    "5",
			wantStart: 5,
			wantEnd:   5,
			wantErr:   false,
		},
		{
			name:      "whitespace in start",
			startStr:  "  3  ",
			endStr:    "7",
			wantStart: 3,
			wantEnd:   7,
			wantErr:   false,
		},
		{
			name:      "whitespace in end",
			startStr:  "1",
			endStr:    "  10  ",
			wantStart: 1,
			wantEnd:   10,
			wantErr:   false,
		},
		{
			name:     "invalid start not a number",
			startStr: "abc",
			endStr:   "5",
			wantErr:  true,
			errMsg:   "invalid start_line",
		},
		{
			name:     "invalid end not a number",
			startStr: "1",
			endStr:   "xyz",
			wantErr:  true,
			errMsg:   "invalid end_line",
		},
		{
			name:     "start line zero",
			startStr: "0",
			endStr:   "5",
			wantErr:  true,
			errMsg:   "start_line must be >= 1",
		},
		{
			name:     "start line negative",
			startStr: "-1",
			endStr:   "5",
			wantErr:  true,
			errMsg:   "start_line must be >= 1",
		},
		{
			name:     "end less than start",
			startStr: "10",
			endStr:   "5",
			wantErr:  true,
			errMsg:   "end_line must be >= start_line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := parseLineRange(tt.startStr, tt.endStr)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseLineRange(%q, %q) expected error containing %q, got nil", tt.startStr, tt.endStr, tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("parseLineRange(%q, %q) error = %q, want error containing %q", tt.startStr, tt.endStr, err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("parseLineRange(%q, %q) unexpected error: %v", tt.startStr, tt.endStr, err)
					return
				}
				if start != tt.wantStart || end != tt.wantEnd {
					t.Errorf("parseLineRange(%q, %q) = (%d, %d), want (%d, %d)", tt.startStr, tt.endStr, start, end, tt.wantStart, tt.wantEnd)
				}
			}
		})
	}
}

func TestFindAllOccurrencesLineRanges(t *testing.T) {
	tests := []struct {
		name    string
		content string
		needle  string
		max     int
		want    int // expected number of occurrences
	}{
		{
			name:    "empty needle",
			content: "line1\nline2\nline3",
			needle:  "",
			max:     10,
			want:    0,
		},
		{
			name:    "max zero",
			content: "line1\nline2\nline3",
			needle:  "line",
			max:     0,
			want:    0,
		},
		{
			name:    "single occurrence",
			content: "foo\nbar\nbaz",
			needle:  "bar",
			max:     10,
			want:    1,
		},
		{
			name:    "multiple occurrences",
			content: "foo\nbar\nfoo\nbaz\nfoo",
			needle:  "foo",
			max:     10,
			want:    3,
		},
		{
			name:    "limited by max",
			content: "foo\nbar\nfoo\nbaz\nfoo",
			needle:  "foo",
			max:     2,
			want:    2,
		},
		{
			name:    "no occurrence",
			content: "foo\nbar\nbaz",
			needle:  "xyz",
			max:     10,
			want:    0,
		},
		{
			name:    "multiline needle",
			content: "line1\nline2\nline3\nline2\nline3",
			needle:  "line2\nline3",
			max:     10,
			want:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findAllOccurrencesLineRanges(tt.content, tt.needle, tt.max)
			if len(result) != tt.want {
				t.Errorf("findAllOccurrencesLineRanges() returned %d occurrences, want %d", len(result), tt.want)
			}
		})
	}
}
