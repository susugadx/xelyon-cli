package applypatch

import (
	"os"
	"strings"
	"testing"
)

func TestBuildPatchPreview_LineNumbers(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "target.go", "package main\n\nfunc target() {\n\tprintln(\"old\")\n\tprintln(\"keep\")\n}\n")

		patch := "*** Begin Patch\n" +
			"*** Update File: target.go\n" +
			"@@ func target() {\n" +
			" \tprintln(\"old\")\n" +
			"-\tprintln(\"keep\")\n" +
			"+\tprintln(\"new\")\n" +
			" }\n" +
			"*** End Patch"

		previews, err := BuildPatchPreview(patch, os.ReadFile)
		if err != nil {
			t.Fatalf("BuildPatchPreview() error = %v", err)
		}
		if len(previews) != 1 {
			t.Fatalf("len(previews) = %d, want 1", len(previews))
		}

		lines := previews[0].Hunks[0].Lines
		if len(lines) != 4 {
			t.Fatalf("len(lines) = %d, want 4", len(lines))
		}

		want := []struct {
			lineType rune
			lineNum  int
			text     string
		}{
			{lineType: ' ', lineNum: 4, text: "\tprintln(\"old\")"},
			{lineType: '-', lineNum: 5, text: "\tprintln(\"keep\")"},
			{lineType: '+', lineNum: 5, text: "\tprintln(\"new\")"},
			{lineType: ' ', lineNum: 6, text: "}"},
		}

		for i, got := range lines {
			if got.Type != want[i].lineType || got.LineNum != want[i].lineNum || got.Text != want[i].text {
				t.Fatalf("line %d = %#v, want type=%q line=%d text=%q", i, got, want[i].lineType, want[i].lineNum, want[i].text)
			}
		}
	})
}

func TestBuildPatchPreview_FallbackOnMiss(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "target.go", "package main\n\nfunc target() {}\n")

		patch := "*** Begin Patch\n" +
			"*** Update File: target.go\n" +
			"@@ missing()\n" +
			"-func target() {}\n" +
			"+func target() { println(\"x\") }\n" +
			"*** End Patch"

		_, err := BuildPatchPreview(patch, os.ReadFile)
		if err == nil {
			t.Fatal("BuildPatchPreview() error = nil, want context lookup failure")
		}
		if !strings.Contains(err.Error(), "failed to find context 'missing()'") {
			t.Fatalf("BuildPatchPreview() error = %v, want missing-context message", err)
		}
	})
}
