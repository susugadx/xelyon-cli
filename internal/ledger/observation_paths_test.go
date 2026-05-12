package ledger

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

type symlinkedLedgerWorkspace struct {
	realRoot string
	linkRoot string
}

func newSymlinkedLedgerWorkspace(t *testing.T) symlinkedLedgerWorkspace {
	t.Helper()

	realRoot := t.TempDir()
	linkParent := t.TempDir()
	linkRoot := filepath.Join(linkParent, "checkout")
	if err := os.Symlink(realRoot, linkRoot); err != nil {
		t.Skipf("symlink is not available: %v", err)
	}
	return symlinkedLedgerWorkspace{
		realRoot: realRoot,
		linkRoot: linkRoot,
	}
}

func writeSymlinkedWorkspaceFile(t *testing.T, workspace symlinkedLedgerWorkspace, relativePath string, content []byte) string {
	t.Helper()

	targetPath := filepath.Join(workspace.realRoot, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := os.WriteFile(targetPath, content, 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	return targetPath
}

func TestRecorder_RecordToolObservation_NormalizesChangedAndTouchedFiles(t *testing.T) {
	root := t.TempDir()
	store := NewStoreWithRoot(root)
	recorder := store.Recorder()

	recorder.RecordToolObservation(ToolObservation{
		ToolName: "apply_patch",
		Change: &tools.FileChange{
			FilePath: filepath.Join(root, "fallback.go"),
			Details: []tools.FileChangeDetail{
				{FilePath: filepath.Join(root, "src/a.go")},
				{FilePath: ""},
				{FilePath: "src/a.go"},
				{FilePath: filepath.Join(root, "src/b.go")},
				{FilePath: filepath.Join(root, "app/[id]/page.tsx")},
				{FilePath: "app/[slug]/loading.tsx"},
				{FilePath: filepath.Join(root, "..", "outside.go")},
				{FilePath: "https://example.com/src/c.go"},
				{FilePath: "src/*.go"},
			},
		},
	})
	recorder.RecordChangedFile(filepath.Join(root, "src/c.go"))
	recorder.RecordTouchedFile(filepath.Join(root, "src/read.go"))

	snapshot := store.Snapshot()
	wantChanged := []string{"src/a.go", "src/b.go", "app/[id]/page.tsx", "app/[slug]/loading.tsx", "src/c.go"}
	if got := snapshot.ChangedFiles.Paths(); !reflect.DeepEqual(got, wantChanged) {
		t.Fatalf("ChangedFiles.Paths() = %v, want %v", got, wantChanged)
	}
	wantTouched := []string{"src/a.go", "src/b.go", "app/[id]/page.tsx", "app/[slug]/loading.tsx", "src/read.go"}
	if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, wantTouched) {
		t.Fatalf("TouchedFiles.Paths() = %v, want %v", got, wantTouched)
	}
}

func TestRecorder_RecordToolObservation_NormalizesRealPathsFromSymlinkedWorkspace(t *testing.T) {
	workspace := newSymlinkedLedgerWorkspace(t)
	targetPath := writeSymlinkedWorkspaceFile(t, workspace, "src/main.go", []byte("package main\n"))
	store := NewStoreWithRoot(workspace.linkRoot)

	store.Recorder().RecordToolObservation(ToolObservation{
		ToolName: "str_replace",
		Change: &tools.FileChange{
			FilePath: targetPath,
		},
	})

	snapshot := store.Snapshot()
	if got := snapshot.ChangedFiles.Paths(); !reflect.DeepEqual(got, []string{"src/main.go"}) {
		t.Fatalf("ChangedFiles.Paths() = %v, want [src/main.go]", got)
	}
	if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, []string{"src/main.go"}) {
		t.Fatalf("TouchedFiles.Paths() = %v, want [src/main.go]", got)
	}
}

func TestRecorder_RecordToolObservation_NormalizesDeletedRealPathsFromSymlinkedWorkspace(t *testing.T) {
	workspace := newSymlinkedLedgerWorkspace(t)
	targetPath := writeSymlinkedWorkspaceFile(t, workspace, "src/deleted.go", []byte("package main\n"))
	if err := os.Remove(targetPath); err != nil {
		t.Fatalf("remove target file: %v", err)
	}
	store := NewStoreWithRoot(workspace.linkRoot)

	store.Recorder().RecordToolObservation(ToolObservation{
		ToolName: "delete_file",
		Change: &tools.FileChange{
			FilePath: targetPath,
			Details: []tools.FileChangeDetail{{
				FilePath: targetPath,
				Action:   "deleted",
			}},
		},
	})

	snapshot := store.Snapshot()
	if got := snapshot.ChangedFiles.Paths(); !reflect.DeepEqual(got, []string{"src/deleted.go"}) {
		t.Fatalf("ChangedFiles.Paths() = %v, want [src/deleted.go]", got)
	}
	if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, []string{"src/deleted.go"}) {
		t.Fatalf("TouchedFiles.Paths() = %v, want [src/deleted.go]", got)
	}
}

func TestRecorder_RecordToolObservation_ResolvesRelativePathsFromInvocationCWD(t *testing.T) {
	root := t.TempDir()
	invocationCWD := filepath.Join(root, "pkg")
	if err := os.MkdirAll(invocationCWD, 0o755); err != nil {
		t.Fatalf("mkdir invocation cwd: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(invocationCWD, "pkg"), 0o755); err != nil {
		t.Fatalf("mkdir nested invocation cwd: %v", err)
	}
	for _, path := range []string{
		filepath.Join(root, "foo.go"),
		filepath.Join(invocationCWD, "foo.go"),
		filepath.Join(invocationCWD, "repo.go"),
		filepath.Join(invocationCWD, "pkg", "cwd.go"),
		filepath.Join(invocationCWD, "pkg", "next.go"),
		filepath.Join(invocationCWD, "pkg", "repo.go"),
		filepath.Join(invocationCWD, "next.go"),
	} {
		if err := os.WriteFile(path, []byte("package pkg\n"), 0o644); err != nil {
			t.Fatalf("write test file %s: %v", path, err)
		}
	}
	store := NewStoreWithWorkspace(root, invocationCWD)
	recorder := store.Recorder()

	recorder.RecordChangedFile("changed.go")
	recorder.RecordToolObservation(ToolObservation{
		ToolName: "read_file",
		Result: strings.Join([]string{
			"📄 File: foo.go",
			"1: package pkg",
		}, "\n"),
	})
	recorder.RecordToolObservation(ToolObservation{
		ToolName: "read_file",
		Result: strings.Join([]string{
			"📄 File: pkg/cwd.go",
			"1: package pkg",
		}, "\n"),
	})
	recorder.RecordToolObservation(ToolObservation{
		ToolName: "bash",
		Args:     map[string]string{"command": "go test ./..."},
		Result:   "ok\nfoo_test.go:3: ok",
	})
	recorder.RecordToolObservation(ToolObservation{
		ToolName:   "search_code",
		ToolCallID: "search-repo-relative",
		Args:       map[string]string{"path": root},
		Result: strings.Join([]string{
			"Found 1 match(es) in 1 file(s)",
			"",
			"📄 pkg/repo.go (1 match(es)) [L1]",
			"  [ref]     >   2 │ func repo() {}",
			"",
			"Recommended reads:",
			"  - pkg/next.go:1 | repo-relative follow up",
		}, "\n"),
	})

	snapshot := store.Snapshot()
	if got := snapshot.ChangedFiles.Paths(); !reflect.DeepEqual(got, []string{"pkg/changed.go"}) {
		t.Fatalf("ChangedFiles.Paths() = %v, want [pkg/changed.go]", got)
	}
	wantTouched := []string{"pkg/foo.go", "pkg/pkg/cwd.go", "pkg/foo_test.go", "pkg/repo.go"}
	if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, wantTouched) {
		t.Fatalf("TouchedFiles.Paths() = %v, want %v", got, wantTouched)
	}
	evidence := snapshot.Evidence.Items()
	if len(evidence) != 3 || evidence[0].Path() != "pkg/foo.go" || evidence[1].Path() != "pkg/pkg/cwd.go" || evidence[2].Path() != "pkg/repo.go" {
		t.Fatalf("Evidence = %#v, want paths [pkg/foo.go pkg/pkg/cwd.go pkg/repo.go]", evidence)
	}
	reads := snapshot.RecommendedReads.Items()
	if len(reads) != 1 || reads[0].Path() != "pkg/next.go" {
		t.Fatalf("RecommendedReads = %#v, want path pkg/next.go", reads)
	}
}

func TestRecorder_RecordToolObservation_SearchCodeAbsoluteScopeUsesRepoRelativeHeaders(t *testing.T) {
	root := t.TempDir()
	invocationCWD := filepath.Join(root, "other")
	for _, path := range []string{
		filepath.Join(root, "pkg", "scoped.go"),
		filepath.Join(invocationCWD, "pkg", "scoped.go"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir test file dir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("package scoped\n"), 0o644); err != nil {
			t.Fatalf("write test file %s: %v", path, err)
		}
	}
	store := NewStoreWithWorkspace(root, invocationCWD)
	recorder := store.Recorder()

	recorder.RecordToolObservation(ToolObservation{
		ToolName:   "search_code",
		ToolCallID: "search-absolute-scope",
		Args:       map[string]string{"path": filepath.Join(root, "pkg")},
		Result: strings.Join([]string{
			"Found 1 match(es) in 1 file(s)",
			"",
			"📄 pkg/scoped.go (1 match(es)) [L1]",
			"  [ref]     >   2 │ func scoped() {}",
		}, "\n"),
	})

	snapshot := store.Snapshot()
	if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, []string{"pkg/scoped.go"}) {
		t.Fatalf("TouchedFiles.Paths() = %v, want [pkg/scoped.go]", got)
	}
	evidence := snapshot.Evidence.Items()
	if len(evidence) != 1 || evidence[0].Path() != "pkg/scoped.go" {
		t.Fatalf("Evidence = %#v, want one item for pkg/scoped.go", evidence)
	}
}

func TestRecorder_RecordToolObservation_SymbolPathsUseRenderedPathBase(t *testing.T) {
	t.Run("bare path uses invocation cwd", func(t *testing.T) {
		for _, toolName := range []string{"search_code", "gather_context"} {
			t.Run(toolName, func(t *testing.T) {
				root := t.TempDir()
				invocationCWD := filepath.Join(root, "pkg")
				for _, path := range []string{
					filepath.Join(root, "target.py"),
					filepath.Join(invocationCWD, "target.py"),
				} {
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						t.Fatalf("mkdir test file dir %s: %v", path, err)
					}
					if err := os.WriteFile(path, []byte("def Foo():\n    pass\n"), 0o644); err != nil {
						t.Fatalf("write test file %s: %v", path, err)
					}
				}
				store := NewStoreWithWorkspace(root, invocationCWD)
				recorder := store.Recorder()

				recorder.RecordToolObservation(ToolObservation{
					ToolName:   toolName,
					ToolCallID: "search-symbol-bare",
					Result: strings.Join([]string{
						"── function Foo (L1) in target.py ──",
						"Definition:",
						"  1: def Foo():",
						"",
						"Callers (1):",
						"  - target.py:4 in UseFoo | Foo()",
					}, "\n"),
				})

				snapshot := store.Snapshot()
				if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, []string{"pkg/target.py"}) {
					t.Fatalf("TouchedFiles.Paths() = %v, want [pkg/target.py]", got)
				}
				evidence := snapshot.Evidence.Items()
				if len(evidence) != 3 {
					t.Fatalf("Evidence len = %d, want 3: %#v", len(evidence), evidence)
				}
				for i, fact := range evidence {
					if fact.Path() != "pkg/target.py" {
						t.Fatalf("Evidence[%d].Path() = %q, want pkg/target.py", i, fact.Path())
					}
				}
			})
		}
	})

	t.Run("qualified path stays repo relative", func(t *testing.T) {
		root := t.TempDir()
		invocationCWD := filepath.Join(root, "pkg")
		for _, path := range []string{
			filepath.Join(root, "pkg", "run.go"),
			filepath.Join(invocationCWD, "pkg", "run.go"),
		} {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("mkdir test file dir %s: %v", path, err)
			}
			if err := os.WriteFile(path, []byte("package pkg\n"), 0o644); err != nil {
				t.Fatalf("write test file %s: %v", path, err)
			}
		}
		store := NewStoreWithWorkspace(root, invocationCWD)
		recorder := store.Recorder()

		recorder.RecordToolObservation(ToolObservation{
			ToolName:   "search_code",
			ToolCallID: "search-symbol-qualified",
			Result: strings.Join([]string{
				"── function Run (L3) in pkg/run.go ──",
				"Definition:",
				"  3: func Run() {}",
			}, "\n"),
		})

		snapshot := store.Snapshot()
		if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, []string{"pkg/run.go"}) {
			t.Fatalf("TouchedFiles.Paths() = %v, want [pkg/run.go]", got)
		}
	})
}

func TestRecorder_RecordToolObservation_AmbiguousSymbolCandidatesAreEvidence(t *testing.T) {
	t.Run("generic multiple definitions record dot paths", func(t *testing.T) {
		root := t.TempDir()
		for _, path := range []string{
			filepath.Join(root, "target.py"),
			filepath.Join(root, "pkg", "target.py"),
		} {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("mkdir test file dir %s: %v", path, err)
			}
			if err := os.WriteFile(path, []byte("def Foo():\n    pass\n"), 0o644); err != nil {
				t.Fatalf("write test file %s: %v", path, err)
			}
		}
		store := NewStoreWithWorkspace(root, root)
		recorder := store.Recorder()

		recorder.RecordToolObservation(ToolObservation{
			ToolName:   "search_code",
			ToolCallID: "search-ambiguous-generic",
			Result: strings.Join([]string{
				`Multiple definitions found for "Foo":`,
				`  1. function Foo (L1) in ./target.py [L1]`,
				`  2. function Foo (L3) in ./pkg/target.py [L2]`,
				"",
				`Refine with path to disambiguate (e.g. path="src/models/").`,
			}, "\n"),
		})

		snapshot := store.Snapshot()
		wantTouched := []string{"target.py", "pkg/target.py"}
		if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, wantTouched) {
			t.Fatalf("TouchedFiles.Paths() = %v, want %v", got, wantTouched)
		}
		evidence := snapshot.Evidence.Items()
		if len(evidence) != 2 {
			t.Fatalf("Evidence len = %d, want 2: %#v", len(evidence), evidence)
		}
		if evidence[0].Path() != "target.py" || evidence[0].StartLine() != 1 {
			t.Fatalf("Evidence[0] = %#v, want target.py L1", evidence[0])
		}
		if evidence[1].Path() != "pkg/target.py" || evidence[1].StartLine() != 3 {
			t.Fatalf("Evidence[1] = %#v, want pkg/target.py L3", evidence[1])
		}
	})

	t.Run("generic bare path uses invocation cwd", func(t *testing.T) {
		root := t.TempDir()
		invocationCWD := filepath.Join(root, "pkg")
		for _, path := range []string{
			filepath.Join(root, "target.py"),
			filepath.Join(invocationCWD, "target.py"),
		} {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("mkdir test file dir %s: %v", path, err)
			}
			if err := os.WriteFile(path, []byte("def Foo():\n    pass\n"), 0o644); err != nil {
				t.Fatalf("write test file %s: %v", path, err)
			}
		}
		store := NewStoreWithWorkspace(root, invocationCWD)
		recorder := store.Recorder()

		recorder.RecordToolObservation(ToolObservation{
			ToolName:   "search_code",
			ToolCallID: "search-ambiguous-bare",
			Result: strings.Join([]string{
				`Multiple definitions found for "Foo":`,
				`  1. function Foo (L1) in target.py [L1]`,
				"",
				`Refine with path to disambiguate (e.g. path="src/models/").`,
			}, "\n"),
		})

		snapshot := store.Snapshot()
		if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, []string{"pkg/target.py"}) {
			t.Fatalf("TouchedFiles.Paths() = %v, want [pkg/target.py]", got)
		}
		evidence := snapshot.Evidence.Items()
		if len(evidence) != 1 || evidence[0].Path() != "pkg/target.py" || evidence[0].StartLine() != 1 {
			t.Fatalf("Evidence = %#v, want one item for pkg/target.py L1", evidence)
		}
	})

	t.Run("go multiple symbols prefer repo relative file column", func(t *testing.T) {
		root := t.TempDir()
		invocationCWD := filepath.Join(root, "pkg")
		for _, path := range []string{
			filepath.Join(root, "build.go"),
			filepath.Join(invocationCWD, "build.go"),
		} {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("mkdir test file dir %s: %v", path, err)
			}
			if err := os.WriteFile(path, []byte("package example\n"), 0o644); err != nil {
				t.Fatalf("write test file %s: %v", path, err)
			}
		}
		store := NewStoreWithWorkspace(root, invocationCWD)
		recorder := store.Recorder()

		recorder.RecordToolObservation(ToolObservation{
			ToolName:   "search_code",
			ToolCallID: "search-ambiguous-go",
			Result: strings.Join([]string{
				`Multiple symbols matched "Build":`,
				`  1. build.go                                 function Build (L3-L3)`,
				"",
				`Refine with path or receiver-qualified symbol to disambiguate.`,
			}, "\n"),
		})

		snapshot := store.Snapshot()
		if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, []string{"build.go"}) {
			t.Fatalf("TouchedFiles.Paths() = %v, want [build.go]", got)
		}
		evidence := snapshot.Evidence.Items()
		if len(evidence) != 1 || evidence[0].Path() != "build.go" || evidence[0].StartLine() != 3 || evidence[0].EndLine() != 3 {
			t.Fatalf("Evidence = %#v, want one item for build.go L3-L3", evidence)
		}
	})
}
