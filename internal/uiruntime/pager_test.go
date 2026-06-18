package uiruntime

import (
	"bytes"
	"strings"
	"testing"
)

func newPagerTestRuntime(input string) (*Runtime, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return NewRuntime(strings.NewReader(input), out, out), out
}

func TestNewPager(t *testing.T) {
	p := NewPager()

	if p == nil {
		t.Fatal("NewPager() returned nil")
	}
	if p.pageSize != defaultPageSize {
		t.Fatalf("NewPager() pageSize = %d, want %d", p.pageSize, defaultPageSize)
	}
}

func TestPager_Display_ShortContent(t *testing.T) {
	runtime, out := newPagerTestRuntime("")
	p := NewPagerWithRuntime(runtime)

	p.Display(strings.Repeat("line\n", 50))

	output := out.String()
	if !strings.Contains(output, "line") {
		t.Fatal("Display() should output content")
	}
	if strings.Contains(output, "--more--") {
		t.Fatal("Display() should not show paging prompt for short content")
	}
}

func TestPager_Display_ExactlyPageSize(t *testing.T) {
	runtime, out := newPagerTestRuntime("")
	p := NewPagerWithRuntime(runtime)

	p.Display(strings.Repeat("line\n", 100))

	if !strings.Contains(out.String(), "line") {
		t.Fatal("Display() should output content")
	}
}

func TestPager_Display_EmptyContent(t *testing.T) {
	runtime, _ := newPagerTestRuntime("")
	p := NewPagerWithRuntime(runtime)

	p.Display("")
}

func TestPager_Display_SingleLine(t *testing.T) {
	runtime, out := newPagerTestRuntime("")
	p := NewPagerWithRuntime(runtime)

	p.Display("single line")

	if !strings.Contains(out.String(), "single line") {
		t.Fatal("Display() should output single line")
	}
}

func TestPager_Display_MultiplePages_Structure(t *testing.T) {
	runtime, out := newPagerTestRuntime("")
	p := NewPagerWithRuntime(runtime)
	p.pageSize = 10

	p.Display(strings.Repeat("line\n", 25))

	output := out.String()
	if !strings.Contains(output, "line") {
		t.Fatal("Display() should output first page")
	}
	if !strings.Contains(output, "--more--") {
		t.Fatal("Display() should show paging prompt for long content")
	}
}

func TestPager_PageSize(t *testing.T) {
	tests := []struct {
		name      string
		pageSize  int
		lineCount int
		wantPages int
	}{
		{name: "9 lines, pageSize 10", pageSize: 10, lineCount: 9, wantPages: 1},
		{name: "25 lines, pageSize 10", pageSize: 10, lineCount: 25, wantPages: 3},
		{name: "5 lines, pageSize 10", pageSize: 10, lineCount: 5, wantPages: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtime, _ := newPagerTestRuntime("")
			p := NewPagerWithRuntime(runtime)
			p.pageSize = tt.pageSize

			content := strings.Repeat("line\n", tt.lineCount)
			lines := strings.Split(content, "\n")
			needsPaging := len(lines) > p.pageSize

			if tt.lineCount < tt.pageSize && needsPaging {
				t.Fatal("Should not need paging for content within page size")
			}
			if tt.lineCount > tt.pageSize && !needsPaging {
				t.Fatal("Should need paging for content exceeding page size")
			}
		})
	}
}

func TestPager_Display_ContentWithSpecialCharacters(t *testing.T) {
	runtime, out := newPagerTestRuntime("")
	p := NewPagerWithRuntime(runtime)

	content := "Hello 世界\n🌍 emoji\n特殊文字: @#$%\n"
	p.Display(content)

	output := out.String()
	if !strings.Contains(output, "Hello 世界") {
		t.Fatal("Display() should preserve Japanese characters")
	}
	if !strings.Contains(output, "🌍 emoji") {
		t.Fatal("Display() should preserve emoji")
	}
}

func TestPager_Display_NoNewlineAtEnd(t *testing.T) {
	runtime, out := newPagerTestRuntime("")
	p := NewPagerWithRuntime(runtime)

	p.Display("line1\nline2\nline3")

	output := out.String()
	if !strings.Contains(output, "line1") || !strings.Contains(output, "line3") {
		t.Fatal("Display() should output all lines")
	}
}
