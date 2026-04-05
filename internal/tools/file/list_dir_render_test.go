package file

import (
	"errors"
	"strings"
	"testing"
)

func TestRenderListDirSummary_RendersStructuredSections(t *testing.T) {
	section := &listDirSection{
		totalDirs:  2,
		totalFiles: 1,
		dirs:       []string{"a/", "b/"},
		files:      []listDirFileSummary{{name: "root.txt", size: 4}},
		subtrees: []*listDirSection{
			{
				relPath:    "a/",
				totalDirs:  1,
				totalFiles: 1,
				dirs:       []string{"nested/"},
				files:      []listDirFileSummary{{name: "child.txt", size: 5}},
			},
		},
	}

	rendered := strings.Join(renderListDirSummary("/tmp/project", 2, section), "\n")
	if !strings.Contains(rendered, "summary: depth=2, dirs=2, files=1") {
		t.Fatalf("expected summary header, got: %s", rendered)
	}
	if !strings.Contains(rendered, "dirs: a/, b/") {
		t.Fatalf("expected root dirs, got: %s", rendered)
	}
	if !strings.Contains(rendered, "files: root.txt (4 bytes)") {
		t.Fatalf("expected root files, got: %s", rendered)
	}
	if !strings.Contains(rendered, "subtrees: 1 shown") {
		t.Fatalf("expected subtree summary, got: %s", rendered)
	}
	if !strings.Contains(rendered, "- a/ -> dirs=1, files=1") {
		t.Fatalf("expected child section, got: %s", rendered)
	}
}

func TestRenderListDirSummary_ReadError(t *testing.T) {
	section := &listDirSection{readErr: errors.New("boom")}

	rendered := strings.Join(renderListDirSummary("/tmp/project", 1, section), "\n")
	if !strings.Contains(rendered, "Error: failed to read directory") {
		t.Fatalf("expected read error line, got: %s", rendered)
	}
}
