package applypatch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyPatch_AddFile(t *testing.T) {
	withTempWorkdir(t, func() {
		result, err := ApplyPatch("*** Begin Patch\n*** Add File: hello.txt\n+Hello\n+World\n*** End Patch")
		if err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}
		assertApplyResult(t, result, []string{"hello.txt"}, nil, nil)
		assertFileContent(t, "hello.txt", "Hello\nWorld\n")
	})
}

func TestApplyPatch_AddFileFailsWhenTargetExists(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "hello.txt", "existing\n")

		_, err := ApplyPatch("*** Begin Patch\n*** Add File: hello.txt\n+Hello\n*** End Patch")
		if err == nil {
			t.Fatal("ApplyPatch() should fail when Add File targets an existing file")
		}
		if !strings.Contains(err.Error(), "file already exists") {
			t.Fatalf("unexpected error: %v", err)
		}
		assertFileContent(t, "hello.txt", "existing\n")
	})
}

func TestApplyPatch_DeleteFile(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "delete.txt", "bye\n")

		result, err := ApplyPatch("*** Begin Patch\n*** Delete File: delete.txt\n*** End Patch")
		if err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}
		assertApplyResult(t, result, nil, nil, []string{"delete.txt"})
		if _, err := os.Stat("delete.txt"); !os.IsNotExist(err) {
			t.Fatalf("delete.txt should be removed, stat err = %v", err)
		}
	})
}

func TestApplyPatch_UpdateSingleChunk(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "main.go", "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n")

		patch := "*** Begin Patch\n" +
			"*** Update File: main.go\n" +
			"@@\n" +
			" package main\n" +
			" \n" +
			" func main() {\n" +
			"-\tprintln(\"hello\")\n" +
			"+\tprintln(\"world\")\n" +
			" }\n" +
			"*** End Patch"

		result, err := ApplyPatch(patch)
		if err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}
		assertApplyResult(t, result, nil, []string{"main.go"}, nil)
		assertFileContent(t, "main.go", "package main\n\nfunc main() {\n\tprintln(\"world\")\n}\n")
	})
}

func TestApplyPatch_UpdateMultipleChunks(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "multi.txt", "foo\nbar\nbaz\nqux\n")

		patch := "*** Begin Patch\n" +
			"*** Update File: multi.txt\n" +
			"@@\n" +
			" foo\n" +
			"-bar\n" +
			"+BAR\n" +
			"@@\n" +
			" baz\n" +
			"-qux\n" +
			"+QUX\n" +
			"*** End Patch"

		result, err := ApplyPatch(patch)
		if err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}
		assertApplyResult(t, result, nil, []string{"multi.txt"}, nil)
		assertFileContent(t, "multi.txt", "foo\nBAR\nbaz\nQUX\n")
	})
}

func TestApplyPatch_MultipleFiles(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "update.txt", "before\n")
		writeTestFile(t, "delete.txt", "remove\n")

		patch := "*** Begin Patch\n" +
			"*** Add File: add.txt\n" +
			"+new\n" +
			"*** Update File: update.txt\n" +
			"@@\n" +
			"-before\n" +
			"+after\n" +
			"*** Delete File: delete.txt\n" +
			"*** End Patch"

		result, err := ApplyPatch(patch)
		if err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}
		assertApplyResult(t, result, []string{"add.txt"}, []string{"update.txt"}, []string{"delete.txt"})
		assertFileContent(t, "add.txt", "new\n")
		assertFileContent(t, "update.txt", "after\n")
		if _, err := os.Stat("delete.txt"); !os.IsNotExist(err) {
			t.Fatalf("delete.txt should be removed, stat err = %v", err)
		}
	})
}

func TestApplyPatch_IsAtomicOnLaterChunkFailure(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "good.txt", "before\n")
		writeTestFile(t, "bad.txt", "keep\n")

		patch := "*** Begin Patch\n" +
			"*** Update File: good.txt\n" +
			"@@\n" +
			"-before\n" +
			"+after\n" +
			"*** Update File: bad.txt\n" +
			"@@\n" +
			"-missing\n" +
			"+new\n" +
			"*** End Patch"

		_, err := ApplyPatch(patch)
		if err == nil {
			t.Fatal("ApplyPatch() should fail when a later hunk cannot be matched")
		}

		assertFileContent(t, "good.txt", "before\n")
		assertFileContent(t, "bad.txt", "keep\n")
	})
}

func TestApplyPatch_IsAtomicWhenLaterDeleteFails(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "good.txt", "before\n")

		patch := "*** Begin Patch\n" +
			"*** Update File: good.txt\n" +
			"@@\n" +
			"-before\n" +
			"+after\n" +
			"*** Delete File: missing.txt\n" +
			"*** End Patch"

		_, err := ApplyPatch(patch)
		if err == nil {
			t.Fatal("ApplyPatch() should fail when a later delete target is missing")
		}

		assertFileContent(t, "good.txt", "before\n")
	})
}

func TestApplyPatch_ContextJump(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "jump.go", "package main\n\nfunc keep() {\n\tprintln(\"keep\")\n}\n\nfunc target() {\n\tprintln(\"old\")\n}\n")

		patch := "*** Begin Patch\n" +
			"*** Update File: jump.go\n" +
			"@@ func target() {\n" +
			"-\tprintln(\"old\")\n" +
			"+\tprintln(\"new\")\n" +
			"*** End Patch"

		result, err := ApplyPatch(patch)
		if err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}
		assertApplyResult(t, result, nil, []string{"jump.go"}, nil)
		assertFileContent(t, "jump.go", "package main\n\nfunc keep() {\n\tprintln(\"keep\")\n}\n\nfunc target() {\n\tprintln(\"new\")\n}\n")
	})
}

func TestApplyPatch_MoveFile(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "src.txt", "line\n")

		patch := "*** Begin Patch\n" +
			"*** Update File: src.txt\n" +
			"*** Move to: dst.txt\n" +
			"@@\n" +
			"-line\n" +
			"+line2\n" +
			"*** End Patch"

		result, err := ApplyPatch(patch)
		if err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}
		assertApplyResult(t, result, nil, []string{"dst.txt"}, nil)
		assertFileContent(t, "dst.txt", "line2\n")
		if _, err := os.Stat("src.txt"); !os.IsNotExist(err) {
			t.Fatalf("src.txt should be removed, stat err = %v", err)
		}
	})
}

func TestApplyPatch_MoveToSamePathTreatsAsUpdate(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "src.txt", "line\n")

		patch := "*** Begin Patch\n" +
			"*** Update File: src.txt\n" +
			"*** Move to: src.txt\n" +
			"@@\n" +
			"-line\n" +
			"+line2\n" +
			"*** End Patch"

		result, err := ApplyPatch(patch)
		if err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}
		assertApplyResult(t, result, nil, []string{"src.txt"}, nil)
		assertFileContent(t, "src.txt", "line2\n")
	})
}

func TestApplyPatch_PreservesPermissionsOnUpdate(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "script.sh", "#!/bin/sh\necho old\n")
		if err := os.Chmod("script.sh", 0o755); err != nil {
			t.Fatalf("os.Chmod(script.sh) error = %v", err)
		}

		patch := "*** Begin Patch\n" +
			"*** Update File: script.sh\n" +
			"@@\n" +
			" #!/bin/sh\n" +
			"-echo old\n" +
			"+echo new\n" +
			"*** End Patch"

		if _, err := ApplyPatch(patch); err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}

		assertPerm(t, "script.sh", 0o755)
	})
}

func TestApplyPatch_PreservesPermissionsOnMove(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "src.sh", "#!/bin/sh\necho old\n")
		writeTestFile(t, "dst.sh", "#!/bin/sh\necho stale\n")
		if err := os.Chmod("src.sh", 0o755); err != nil {
			t.Fatalf("os.Chmod(src.sh) error = %v", err)
		}
		if err := os.Chmod("dst.sh", 0o644); err != nil {
			t.Fatalf("os.Chmod(dst.sh) error = %v", err)
		}

		patch := "*** Begin Patch\n" +
			"*** Update File: src.sh\n" +
			"*** Move to: dst.sh\n" +
			"@@\n" +
			" #!/bin/sh\n" +
			"-echo old\n" +
			"+echo new\n" +
			"*** End Patch"

		if _, err := ApplyPatch(patch); err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}

		assertPerm(t, "dst.sh", 0o755)
		if _, err := os.Stat("src.sh"); !os.IsNotExist(err) {
			t.Fatalf("src.sh should be removed, stat err = %v", err)
		}
	})
}

func TestApplyPatch_WhitespaceTolerance(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "whitespace.go", "func main() {\n        value := 1   \n}\n")

		patch := "*** Begin Patch\n" +
			"*** Update File: whitespace.go\n" +
			"@@\n" +
			" func main() {\n" +
			"-    value := 1\n" +
			"+    value := 2\n" +
			" }\n" +
			"*** End Patch"

		result, err := ApplyPatch(patch)
		if err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}
		assertApplyResult(t, result, nil, []string{"whitespace.go"}, nil)
		assertFileContent(t, "whitespace.go", "func main() {\n    value := 2\n}\n")
	})
}

func TestApplyPatch_InvalidPatch(t *testing.T) {
	withTempWorkdir(t, func() {
		_, err := ApplyPatch("bad")
		if err == nil {
			t.Fatal("ApplyPatch() should fail for invalid patch")
		}
		if !strings.Contains(err.Error(), "The first line of the patch must be '*** Begin Patch'") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestApplyPatch_EndOfFile(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "eof.txt", "foo\nbar\nbaz\n")

		patch := "*** Begin Patch\n" +
			"*** Update File: eof.txt\n" +
			"@@\n" +
			"+quux\n" +
			"*** End of File\n" +
			"*** End Patch"

		result, err := ApplyPatch(patch)
		if err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}
		assertApplyResult(t, result, nil, []string{"eof.txt"}, nil)
		assertFileContent(t, "eof.txt", "foo\nbar\nbaz\nquux\n")
	})
}

func TestApplyPatch_HeredocWrapper(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "wrapped.txt", "before\n")

		patch := "<<EOF\n" +
			"*** Begin Patch\n" +
			"*** Update File: wrapped.txt\n" +
			"@@\n" +
			"-before\n" +
			"+after\n" +
			"*** End Patch\n" +
			"EOF\n"

		result, err := ApplyPatch(patch)
		if err != nil {
			t.Fatalf("ApplyPatch() error = %v", err)
		}
		assertApplyResult(t, result, nil, []string{"wrapped.txt"}, nil)
		assertFileContent(t, "wrapped.txt", "after\n")
	})
}

func withTempWorkdir(t *testing.T, fn func()) {
	t.Helper()

	root := t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", root, err)
	}
	defer func() {
		if err := os.Chdir(prev); err != nil {
			t.Fatalf("restore cwd error = %v", err)
		}
	}()

	fn()
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("content mismatch for %s:\n got: %q\nwant: %q", path, string(got), want)
	}
}

func assertPerm(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("perm mismatch for %s: got %o, want %o", path, got, want)
	}
}

func assertApplyResult(t *testing.T, got *ApplyResult, added, modified, deleted []string) {
	t.Helper()

	if got == nil {
		t.Fatal("ApplyResult should not be nil")
	}
	if !sameStrings(got.Added, added) {
		t.Fatalf("Added = %v, want %v", got.Added, added)
	}
	if !sameStrings(got.Modified, modified) {
		t.Fatalf("Modified = %v, want %v", got.Modified, modified)
	}
	if !sameStrings(got.Deleted, deleted) {
		t.Fatalf("Deleted = %v, want %v", got.Deleted, deleted)
	}
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestLooksLikeUnifiedDiffHeader(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"-274,6 +274,32 @@", true},
		{"-43,7 +43,7 @@", true},
		{"-1,3 +1,5", true},
		{"func BuildProjectMap", false},
		{"class Config", false},
		{`case "openrouter":`, false},
		{"", false},
		{"def greet():", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeUnifiedDiffHeader(tt.input)
			if got != tt.want {
				t.Errorf("looksLikeUnifiedDiffHeader(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestApplyPatch_UnifiedDiffHeaderError(t *testing.T) {
	withTempWorkdir(t, func() {
		writeTestFile(t, "target.go", "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n")

		patch := "*** Begin Patch\n" +
			"*** Update File: target.go\n" +
			"@@ -1,3 +1,5 @@\n" +
			" package main\n" +
			"+import \"os\"\n" +
			"*** End Patch\n"

		_, err := ApplyPatch(patch)
		if err == nil {
			t.Fatal("expected error for unified diff header")
		}
		if !strings.Contains(err.Error(), "not unified diff line numbers") {
			t.Errorf("error should mention unified diff line numbers, got: %v", err)
		}
	})
}
