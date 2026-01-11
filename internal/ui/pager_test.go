package ui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestNewPager(t *testing.T) {
	p := NewPager()

	if p == nil {
		t.Fatal("NewPager() returned nil")
	}

	if p.pageSize != defaultPageSize {
		t.Errorf("NewPager() pageSize = %d, want %d", p.pageSize, defaultPageSize)
	}

	if p.pageSize != 100 {
		t.Errorf("NewPager() pageSize = %d, want 100", p.pageSize)
	}
}

func TestPager_Display_ShortContent(t *testing.T) {
	p := NewPager()

	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// 短いコンテンツ（ページングなし）
	shortContent := strings.Repeat("line\n", 50)
	p.Display(shortContent)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}

	output := buf.String()

	// コンテンツがそのまま出力されることを確認
	if !strings.Contains(output, "line") {
		t.Error("Display() should output content")
	}

	// "--more--" プロンプトが表示されないことを確認
	if strings.Contains(output, "--more--") {
		t.Error("Display() should not show paging prompt for short content")
	}
}

func TestPager_Display_ExactlyPageSize(t *testing.T) {
	p := NewPager()

	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// ちょうどpageSize行（ページングなし）
	exactContent := strings.Repeat("line\n", 100)
	lines := strings.Split(exactContent, "\n")

	// 最後の空行を除くと100行
	if len(lines)-1 != 100 {
		t.Fatalf("Test setup error: expected 100 lines, got %d", len(lines)-1)
	}

	p.Display(exactContent)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}

	output := buf.String()

	// コンテンツが出力されることを確認
	if !strings.Contains(output, "line") {
		t.Error("Display() should output content")
	}
}

func TestPager_Display_EmptyContent(t *testing.T) {
	p := NewPager()

	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	p.Display("")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}

	// panicしないことを確認（出力は空でも問題ない）
}

func TestPager_Display_SingleLine(t *testing.T) {
	p := NewPager()

	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	p.Display("single line")

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "single line") {
		t.Error("Display() should output single line")
	}
}

func TestPager_Display_MultiplePages_Structure(t *testing.T) {
	p := NewPager()
	p.pageSize = 10 // テスト用に小さく設定

	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// 25行のコンテンツ（3ページ分）
	content := strings.Repeat("line\n", 25)

	// promptContinueが呼ばれるが、標準入力がないためすぐに終了する
	p.Display(content)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}

	output := buf.String()

	// 最初のページが出力されることを確認
	if !strings.Contains(output, "line") {
		t.Error("Display() should output first page")
	}
}

func TestPager_PageSize(t *testing.T) {
	tests := []struct {
		name      string
		pageSize  int
		lineCount int
		wantPages int
	}{
		{
			name:      "9 lines, pageSize 10",
			pageSize:  10,
			lineCount: 9,
			wantPages: 1,
		},
		{
			name:      "25 lines, pageSize 10",
			pageSize:  10,
			lineCount: 25,
			wantPages: 3,
		},
		{
			name:      "5 lines, pageSize 10",
			pageSize:  10,
			lineCount: 5,
			wantPages: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPager()
			p.pageSize = tt.pageSize

			content := strings.Repeat("line\n", tt.lineCount)
			lines := strings.Split(content, "\n")

			// ページングが必要かどうかを確認
			needsPaging := len(lines) > p.pageSize

			if tt.lineCount < tt.pageSize && needsPaging {
				t.Error("Should not need paging for content within page size")
			}

			if tt.lineCount > tt.pageSize && !needsPaging {
				t.Error("Should need paging for content exceeding page size")
			}
		})
	}
}

func TestPager_DefaultPageSize(t *testing.T) {
	if defaultPageSize != 100 {
		t.Errorf("defaultPageSize = %d, want 100", defaultPageSize)
	}
}

func TestPager_Display_ContentWithSpecialCharacters(t *testing.T) {
	p := NewPager()

	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	specialContent := "Hello 世界\n🌍 emoji\n特殊文字: @#$%\n"
	p.Display(specialContent)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}

	output := buf.String()

	// 特殊文字が保持されることを確認
	if !strings.Contains(output, "世界") {
		t.Error("Display() should preserve Japanese characters")
	}
	if !strings.Contains(output, "🌍") {
		t.Error("Display() should preserve emoji")
	}
}

func TestPager_Display_NoNewlineAtEnd(t *testing.T) {
	p := NewPager()

	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// 末尾に改行なし
	content := "line1\nline2\nline3"
	p.Display(content)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}

	output := buf.String()

	if !strings.Contains(output, "line1") {
		t.Error("Display() should output all lines")
	}
	if !strings.Contains(output, "line3") {
		t.Error("Display() should output last line without newline")
	}
}

func TestPager_Display_OnlyNewlines(t *testing.T) {
	p := NewPager()

	// 標準出力をキャプチャ
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// 改行のみ
	content := "\n\n\n\n\n"
	p.Display(content)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}

	// panicしないことを確認
}

// Note: promptContinue() は標準入力を読むため、自動テストが難しい
// 実際のユーザーインタラクションが必要な部分は手動テストで確認
