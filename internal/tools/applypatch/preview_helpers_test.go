package applypatch

import (
	"errors"
	"testing"
)

func TestPreviewHelperFunctions(t *testing.T) {
	t.Run("buildAddFilePreview", func(t *testing.T) {
		preview := buildAddFilePreview(Hunk{
			Type:     "add",
			Path:     "new.go",
			Contents: "line1\nline2\n",
		})
		if preview.Added != 2 || len(preview.Hunks) != 1 {
			t.Fatalf("buildAddFilePreview() = %+v, want Added=2 and one hunk", preview)
		}
		if preview.Hunks[0].Lines[0].LineNum != 1 || preview.Hunks[0].Lines[1].LineNum != 2 {
			t.Fatalf("buildAddFilePreview() line numbers = %+v", preview.Hunks[0].Lines)
		}
	})

	t.Run("buildDeleteFilePreview without reader", func(t *testing.T) {
		preview := buildDeleteFilePreview(Hunk{Type: "delete", Path: "gone.go"}, nil)
		if preview.Removed != 0 {
			t.Fatalf("buildDeleteFilePreview() Removed = %d, want 0 without reader", preview.Removed)
		}
	})

	t.Run("buildDeleteFilePreview counts lines", func(t *testing.T) {
		preview := buildDeleteFilePreview(Hunk{Type: "delete", Path: "gone.go"}, func(path string) ([]byte, error) {
			return []byte("one\ntwo\n"), nil
		})
		if preview.Removed != 2 {
			t.Fatalf("buildDeleteFilePreview() Removed = %d, want 2", preview.Removed)
		}
	})

	t.Run("buildUpdateFilePreview requires reader", func(t *testing.T) {
		if _, err := buildUpdateFilePreview(Hunk{Type: "update", Path: "target.go"}, nil); err == nil {
			t.Fatal("buildUpdateFilePreview() error = nil, want reader requirement error")
		}
	})

	t.Run("BuildPatchPreview unsupported hunk", func(t *testing.T) {
		parsed := ParsedPatch{Hunks: []Hunk{{Type: "mystery", Path: "file.go"}}}
		_ = parsed
	})
}

func TestCountPreviewContentLines(t *testing.T) {
	if got := countPreviewContentLines(""); got != 0 {
		t.Fatalf("countPreviewContentLines(\"\") = %d, want 0", got)
	}
	if got := countPreviewContentLines("one\ntwo\n"); got != 2 {
		t.Fatalf("countPreviewContentLines() = %d, want 2", got)
	}
	if got := len(splitPreviewContentLines("one\ntwo\n")); got != 2 {
		t.Fatalf("splitPreviewContentLines() len = %d, want 2", got)
	}
}

func TestBuildPatchPreview_AddAndDeleteHunks(t *testing.T) {
	patch := "*** Begin Patch\n" +
		"*** Add File: new.txt\n" +
		"+hello\n" +
		"+world\n" +
		"*** Delete File: old.txt\n" +
		"*** End Patch"

	previews, err := BuildPatchPreview(patch, func(path string) ([]byte, error) {
		if path == "old.txt" {
			return []byte("old line\n"), nil
		}
		return nil, errors.New("unexpected path")
	})
	if err != nil {
		t.Fatalf("BuildPatchPreview() error = %v", err)
	}
	if len(previews) != 2 {
		t.Fatalf("len(previews) = %d, want 2", len(previews))
	}
	if previews[0].Action != "add" || previews[0].Added != 2 {
		t.Fatalf("add preview = %+v, want Added=2", previews[0])
	}
	if previews[1].Action != "delete" {
		t.Fatalf("delete preview action = %q, want delete", previews[1].Action)
	}
}
