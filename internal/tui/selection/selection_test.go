package selection

import "testing"

func TestNormalize(t *testing.T) {
	r, ok := Normalize(5, 3, 2, 1)
	if !ok {
		t.Fatal("Normalize() ok = false, want true")
	}
	if r != (Range{StartLine: 2, StartCol: 1, EndLine: 5, EndCol: 3}) {
		t.Fatalf("Normalize() = %#v", r)
	}

	if _, ok := Normalize(-1, -1, -1, -1); ok {
		t.Fatal("Normalize(invalid) ok = true, want false")
	}
}

func TestColumnsForLine(t *testing.T) {
	r := Range{StartLine: 1, StartCol: 2, EndLine: 3, EndCol: 4}
	tests := []struct {
		line      int
		wantStart int
		wantEnd   int
		wantOK    bool
	}{
		{line: 0, wantOK: false},
		{line: 1, wantStart: 2, wantEnd: 9999, wantOK: true},
		{line: 2, wantStart: 0, wantEnd: 9999, wantOK: true},
		{line: 3, wantStart: 0, wantEnd: 5, wantOK: true},
		{line: 4, wantOK: false},
	}

	for _, tt := range tests {
		start, end, ok := ColumnsForLine(r, tt.line, 10)
		if ok != tt.wantOK || start != tt.wantStart || end != tt.wantEnd {
			t.Fatalf("ColumnsForLine(line=%d) = (%d, %d, %v), want (%d, %d, %v)", tt.line, start, end, ok, tt.wantStart, tt.wantEnd, tt.wantOK)
		}
	}
}

func TestANSIPlainText(t *testing.T) {
	lines := []string{"zero", "\033[31malpha\033[0m", "日本語tail"}
	got, lineCount := ANSIPlainText(lines, Range{StartLine: 1, StartCol: 1, EndLine: 2, EndCol: 4})
	if got != "lpha\n日本語" {
		t.Fatalf("ANSIPlainText() = %q, want %q", got, "lpha\n日本語")
	}
	if lineCount != 2 {
		t.Fatalf("lineCount = %d, want 2", lineCount)
	}
}
