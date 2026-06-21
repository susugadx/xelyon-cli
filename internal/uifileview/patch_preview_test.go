package uifileview

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestPatchPreviewFormattingHelpers(t *testing.T) {
	if got := formatPatchAction("add"); got != "Creating" {
		t.Fatalf("formatPatchAction(add) = %q, want %q", got, "Creating")
	}
	if got := formatPatchAction("deleted"); got != "Deleting" {
		t.Fatalf("formatPatchAction(deleted) = %q, want %q", got, "Deleting")
	}
	if got := formatPatchAction("edit"); got != "Editing" {
		t.Fatalf("formatPatchAction(edit) = %q, want %q", got, "Editing")
	}

	if got := formatPatchTarget("from.go", "to.go"); got != "from.go -> to.go" {
		t.Fatalf("formatPatchTarget() = %q, want %q", got, "from.go -> to.go")
	}
	if got := formatPatchTarget("same.go", "same.go"); got != "same.go" {
		t.Fatalf("formatPatchTarget() = %q, want %q", got, "same.go")
	}

	tests := []struct {
		name         string
		added        int
		removed      int
		wantFragment string
	}{
		{name: "added and removed", added: 2, removed: 1, wantFragment: "(+2, -1)"},
		{name: "added only", added: 3, removed: 0, wantFragment: "(+3)"},
		{name: "removed only", added: 0, removed: 4, wantFragment: "(-4)"},
		{name: "no counts", added: 0, removed: 0, wantFragment: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatPatchCounts(tt.added, tt.removed); got != tt.wantFragment {
				t.Fatalf("formatPatchCounts(%d, %d) = %q, want %q", tt.added, tt.removed, got, tt.wantFragment)
			}
		})
	}
}

func TestShowPatchPreview_RendersAllActionTypes(t *testing.T) {
	originalNoColor := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = originalNoColor
	})

	previews := []PatchFilePreview{
		{
			Path:   "new.txt",
			Action: "add",
			Added:  2,
			Hunks: []PatchHunkPreview{
				{
					Lines: []PatchPreviewLine{
						{Type: '+', LineNum: 1, Text: "first"},
						{Type: '+', LineNum: 2, Text: "second"},
					},
				},
				{
					Lines: []PatchPreviewLine{
						{Type: '+', LineNum: 3, Text: "ignored"},
					},
				},
			},
		},
		{
			Path:    "old.txt",
			Action:  "delete",
			Removed: 1,
			Hunks: []PatchHunkPreview{
				{
					Lines: []PatchPreviewLine{
						{Type: '-', LineNum: 7, Text: "gone"},
					},
				},
			},
		},
		{
			Path:     "from.go",
			MovePath: "to.go",
			Action:   "edit",
			Added:    1,
			Removed:  1,
			Hunks: []PatchHunkPreview{
				{
					Lines: []PatchPreviewLine{
						{Type: ' ', LineNum: 3, Text: "ctx"},
						{Type: '-', LineNum: 4, Text: "old"},
						{Type: '+', LineNum: 4, Text: "new"},
					},
				},
				{
					Lines: []PatchPreviewLine{
						{Type: ' ', LineNum: 10, Text: "tail"},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	ShowPatchPreview(&buf, previews)
	got := stripANSI(buf.String())

	for _, fragment := range []string{
		"Creating new.txt (+2)",
		"Deleting old.txt (-1)",
		"Editing from.go -> to.go (+1, -1)",
		"first",
		"gone",
		"old",
		"new",
		"tail",
		"     :",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("ShowPatchPreview() output missing %q:\n%s", fragment, got)
		}
	}
	if strings.Contains(got, "ignored") {
		t.Fatalf("ShowPatchPreview() should only render first hunk for add/delete actions:\n%s", got)
	}
}

func TestShowSinglePatchPreview_RendersSingleFile(t *testing.T) {
	originalNoColor := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = originalNoColor
	})

	var buf bytes.Buffer
	ShowSinglePatchPreview(&buf, PatchFilePreview{
		Path:    "only.go",
		Action:  "edit",
		Added:   1,
		Removed: 1,
		Hunks: []PatchHunkPreview{
			{
				Lines: []PatchPreviewLine{
					{Type: '-', LineNum: 10, Text: "before"},
					{Type: '+', LineNum: 10, Text: "after"},
				},
			},
		},
	})

	got := stripANSI(buf.String())
	if !strings.Contains(got, "Editing only.go (+1, -1)") {
		t.Fatalf("ShowSinglePatchPreview() output = %q, want header", got)
	}
	if !strings.Contains(got, "before") || !strings.Contains(got, "after") {
		t.Fatalf("ShowSinglePatchPreview() output = %q, want line preview", got)
	}
}
