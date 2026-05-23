package search

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestJSFamilyLSPReferenceBuilderLoadsEachFileOnce(t *testing.T) {
	absPath := filepath.Join(t.TempDir(), "src", "app.ts")
	loads := 0
	builder := &jsFamilyLSPReferenceBuilder{
		symbol: "buildUser",
		files:  make(map[string]*jsFamilyLSPReferenceFile),
		loadFile: func(gotPath string) *jsFamilyLSPReferenceFile {
			loads++
			if gotPath != absPath {
				t.Fatalf("load path = %q, want %q", gotPath, absPath)
			}
			return &jsFamilyLSPReferenceFile{lines: []string{
				"buildUser('one')",
				"buildUser('two')",
			}}
		},
	}
	defer builder.Close()

	first := builder.Ref(jsFamilyLSPReferenceCandidate{
		displayPath: "src/app.ts",
		absPath:     absPath,
		loc:         navigation.LSPLocation{Line: 1},
	})
	second := builder.Ref(jsFamilyLSPReferenceCandidate{
		displayPath: "src/app.ts",
		absPath:     absPath,
		loc:         navigation.LSPLocation{Line: 2},
	})

	if loads != 1 {
		t.Fatalf("loads = %d, want one load for repeated LSP locations in the same file", loads)
	}
	if first.Snippet != "buildUser('one')" || second.Snippet != "buildUser('two')" {
		t.Fatalf("snippets = (%q, %q), want cached file snippets", first.Snippet, second.Snippet)
	}
}
