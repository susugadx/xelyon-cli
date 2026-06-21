package readtool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatOutline_Signatures(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("package main\n\nimport \"fmt\"\n\n")
	for i := 5; i <= 30; i++ {
		fmt.Fprintf(&sb, "// line %d\n", i)
	}
	sb.WriteString("func Alpha() {\n")
	for i := 0; i < 5; i++ {
		sb.WriteString("\tfmt.Println()\n")
	}
	sb.WriteString("}\n\n")
	sb.WriteString("func Beta() {\n")
	for i := 0; i < 5; i++ {
		sb.WriteString("\tfmt.Println()\n")
	}
	sb.WriteString("}\n")

	lines := strings.Split(sb.String(), "\n")
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(goFile, []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}

	out := formatOutline(goFile, lines, len(lines))

	if !strings.Contains(out, "package main") {
		t.Errorf("expected head to contain package declaration")
	}
	if !strings.Contains(out, "func Alpha") {
		t.Errorf("expected signature for Alpha, got:\n%s", out)
	}
	if !strings.Contains(out, "func Beta") {
		t.Errorf("expected signature for Beta, got:\n%s", out)
	}
	if !strings.Contains(out, "L") {
		t.Errorf("expected line numbers in signatures, got:\n%s", out)
	}
	if !strings.Contains(out, fmt.Sprintf("(%d lines total. For specific sections: paths=[", len(lines))) {
		t.Errorf("expected total-lines footer")
	}
}

func TestFormatOutline_SmallFile(t *testing.T) {
	var sb strings.Builder
	for i := 1; i <= 35; i++ {
		fmt.Fprintf(&sb, "line %d\n", i)
	}
	lines := strings.Split(strings.TrimRight(sb.String(), "\n"), "\n")
	tmpDir := t.TempDir()
	txtFile := filepath.Join(tmpDir, "small.txt")
	if err := os.WriteFile(txtFile, []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}

	out := formatOutline(txtFile, lines, len(lines))

	if !strings.Contains(out, "line 1") {
		t.Errorf("expected first line")
	}
	if !strings.Contains(out, "line 35") {
		t.Errorf("expected last line, got:\n%s", out)
	}
	if !strings.Contains(out, "35 lines total") {
		t.Errorf("expected total line count, got:\n%s", out)
	}
}
