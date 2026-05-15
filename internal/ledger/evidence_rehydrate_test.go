package ledger

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestEvidencePointers_ReturnDefensiveCopies(t *testing.T) {
	state := RuntimeTaskState{
		Evidence: Evidence{items: []evidenceFact{{
			path:       "internal/ledger/ledger.go",
			startLine:  10,
			endLine:    12,
			source:     "read_file",
			toolCallID: "call_1",
			fileHash:   "sha256:old",
			stale:      true,
			excerpt:    "type Evidence struct",
		}}},
	}

	pointers := state.Evidence.Pointers()
	if len(pointers) != 1 {
		t.Fatalf("Evidence.Pointers() length = %d, want 1", len(pointers))
	}
	got := pointers[0]
	if got.Path != "internal/ledger/ledger.go" ||
		got.StartLine != 10 ||
		got.EndLine != 12 ||
		got.Source != "read_file" ||
		got.ToolCallID != "call_1" ||
		got.FileHash != "sha256:old" ||
		!got.Stale ||
		got.PathBase != EvidencePointerPathBaseRepoRoot {
		t.Fatalf("Evidence.Pointers()[0] = %#v, want exported fields from evidence fact", got)
	}

	pointers[0].Path = "mutated.go"
	if got := state.Evidence.Pointers()[0].Path; got != "internal/ledger/ledger.go" {
		t.Fatalf("Evidence.Pointers() after mutation = %q, want original path", got)
	}

	fromState := EvidencePointersFromState(state)
	fromState[0].StartLine = 99
	if got := EvidencePointersFromState(state)[0].StartLine; got != 10 {
		t.Fatalf("EvidencePointersFromState() after mutation = %d, want 10", got)
	}
}

func TestRehydrateEvidencePointer_ReadsRepoRelativePath(t *testing.T) {
	root := t.TempDir()
	invocationCWD := filepath.Join(root, "pkg")
	writeLedgerTestFile(t, root, "src/file.go", "line 1\nline 2\nline 3\n")

	result, err := RehydrateEvidencePointer(context.Background(), EvidencePointer{
		Path:      "src/file.go",
		StartLine: 2,
		EndLine:   3,
	}, EvidenceRehydrateOptions{RepoRoot: root, InvocationCWD: invocationCWD})
	if err != nil {
		t.Fatalf("RehydrateEvidencePointer() error = %v", err)
	}
	if result.Path != "src/file.go" || result.Content != "line 2\nline 3" {
		t.Fatalf("result = %#v, want src/file.go lines 2-3", result)
	}
	if result.CurrentFileHash == "" || result.Stale {
		t.Fatalf("result hash/stale = hash %q stale %v, want hash and fresh", result.CurrentFileHash, result.Stale)
	}
}

func TestRehydrateEvidencePointer_ReadsLiteralBracketPath(t *testing.T) {
	root := t.TempDir()
	writeLedgerTestFile(t, root, "app/[id]/page.tsx", "export default function Page() {}\n")

	result, err := RehydrateEvidencePointer(context.Background(), EvidencePointer{
		Path:      "app/[id]/page.tsx",
		StartLine: 1,
		EndLine:   1,
	}, EvidenceRehydrateOptions{RepoRoot: root, InvocationCWD: root})
	if err != nil {
		t.Fatalf("RehydrateEvidencePointer() error = %v", err)
	}
	if result.Path != "app/[id]/page.tsx" || result.Content != "export default function Page() {}" {
		t.Fatalf("result = %#v, want literal bracket path content", result)
	}
}

func TestRehydrateEvidencePointer_NormalizesCRLFLineContent(t *testing.T) {
	root := t.TempDir()
	content := "line 1\r\nline 2\r\nline 3\r\n"
	writeLedgerTestFile(t, root, "windows.txt", content)

	result, err := RehydrateEvidencePointer(context.Background(), EvidencePointer{
		Path:      "windows.txt",
		StartLine: 2,
		EndLine:   3,
	}, EvidenceRehydrateOptions{RepoRoot: root, InvocationCWD: root})
	if err != nil {
		t.Fatalf("RehydrateEvidencePointer() error = %v", err)
	}
	if result.Content != "line 2\nline 3" {
		t.Fatalf("result content = %q, want CRLF-normalized lines", result.Content)
	}
	if result.CurrentFileHash != evidenceFileHash([]byte(content)) {
		t.Fatalf("result hash = %q, want hash of raw file bytes", result.CurrentFileHash)
	}
}

func TestRehydrateEvidencePointer_AutoFallsBackToInvocationCWD(t *testing.T) {
	root := t.TempDir()
	invocationCWD := filepath.Join(root, "pkg")
	writeLedgerTestFile(t, invocationCWD, "fallback.go", "pkg line 1\npkg line 2\n")

	result, err := RehydrateEvidencePointer(context.Background(), EvidencePointer{
		Path:      "fallback.go",
		StartLine: 1,
		EndLine:   1,
	}, EvidenceRehydrateOptions{RepoRoot: root, InvocationCWD: invocationCWD})
	if err != nil {
		t.Fatalf("RehydrateEvidencePointer() error = %v", err)
	}
	if result.Path != "pkg/fallback.go" || result.Content != "pkg line 1" {
		t.Fatalf("result = %#v, want pkg/fallback.go line 1", result)
	}
}

func TestRehydrateEvidencePointer_AutoAllowsInvocationCWDParentPathWithinWorkspace(t *testing.T) {
	root := t.TempDir()
	invocationCWD := filepath.Join(root, "pkg")
	makeLedgerTestDir(t, invocationCWD)
	writeLedgerTestFile(t, root, "evidence.go", "root evidence\n")

	result, err := RehydrateEvidencePointer(context.Background(), EvidencePointer{
		Path:      "../evidence.go",
		StartLine: 1,
		EndLine:   1,
	}, EvidenceRehydrateOptions{RepoRoot: root, InvocationCWD: invocationCWD})
	if err != nil {
		t.Fatalf("RehydrateEvidencePointer() error = %v", err)
	}
	if result.Path != "evidence.go" || result.Content != "root evidence" {
		t.Fatalf("result = %#v, want evidence.go line 1", result)
	}
}

func TestRehydrateEvidencePointer_LedgerPointersUseRepoRelativePathOnly(t *testing.T) {
	root := t.TempDir()
	invocationCWD := filepath.Join(root, "pkg")
	writeLedgerTestFile(t, invocationCWD, "evidence.go", "cwd line\n")
	pointer := ledgerEvidencePointerForTest("evidence.go", 1, 1)

	result, err := RehydrateEvidencePointer(context.Background(), pointer, EvidenceRehydrateOptions{
		RepoRoot:      root,
		InvocationCWD: invocationCWD,
	})

	assertEvidenceRehydrateFailure(t, result, err, EvidenceRehydrateReasonMissingFile)
}

func TestRehydrateEvidencePointer_RepoRootPathBaseRejectsParentPaths(t *testing.T) {
	root := t.TempDir()
	invocationCWD := filepath.Join(root, "pkg")
	writeLedgerTestFile(t, root, "evidence.go", "root evidence\n")

	result, err := RehydrateEvidencePointer(context.Background(), EvidencePointer{
		Path:      "../evidence.go",
		StartLine: 1,
		EndLine:   1,
		PathBase:  EvidencePointerPathBaseRepoRoot,
	}, EvidenceRehydrateOptions{RepoRoot: root, InvocationCWD: invocationCWD})

	assertEvidenceRehydrateFailure(t, result, err, EvidenceRehydrateReasonPathEscape)
}

func TestRehydrateEvidencePointer_RejectsUnknownPointerPathBase(t *testing.T) {
	root := t.TempDir()
	writeLedgerTestFile(t, root, "evidence.go", "root line\n")

	result, err := RehydrateEvidencePointer(context.Background(), EvidencePointer{
		Path:      "evidence.go",
		StartLine: 1,
		EndLine:   1,
		PathBase:  EvidencePointerPathBase("invalid"),
	}, EvidenceRehydrateOptions{RepoRoot: root, InvocationCWD: root})

	assertEvidenceRehydrateFailure(t, result, err, EvidenceRehydrateReasonInvalidPath)
}

func TestRehydrateEvidencePointer_LedgerPointersDoNotRequireInvocationCWD(t *testing.T) {
	root := t.TempDir()
	missingInvocationCWD := filepath.Join(root, "removed")
	writeLedgerTestFile(t, root, "evidence.go", "root line\n")
	pointer := ledgerEvidencePointerForTest("evidence.go", 1, 1)

	result, err := RehydrateEvidencePointer(context.Background(), pointer, EvidenceRehydrateOptions{
		RepoRoot:      root,
		InvocationCWD: missingInvocationCWD,
	})
	if err != nil {
		t.Fatalf("RehydrateEvidencePointer() error = %v", err)
	}
	if result.Path != "evidence.go" || result.Content != "root line" {
		t.Fatalf("result = %#v, want root evidence.go content", result)
	}
}

func TestRehydrateEvidencePointer_RejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeLedgerTestFile(t, outside, "outside.go", "outside\n")
	writeLedgerTestFile(t, root, "safe.go", "safe\n")

	if err := os.Symlink(filepath.Join(outside, "outside.go"), filepath.Join(root, "link.go")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	tests := []struct {
		name   string
		path   string
		reason EvidenceRehydrateErrorReason
	}{
		{
			name:   "outside absolute",
			path:   filepath.Join(outside, "outside.go"),
			reason: EvidenceRehydrateReasonPathEscape,
		},
		{
			name:   "parent escape",
			path:   "../outside.go",
			reason: EvidenceRehydrateReasonPathEscape,
		},
		{
			name:   "glob",
			path:   "*.go",
			reason: EvidenceRehydrateReasonInvalidPath,
		},
		{
			name:   "url",
			path:   "https://example.com/file.go",
			reason: EvidenceRehydrateReasonInvalidPath,
		},
		{
			name:   "locator",
			path:   "locator:abc",
			reason: EvidenceRehydrateReasonInvalidPath,
		},
		{
			name:   "locator id",
			path:   "L12",
			reason: EvidenceRehydrateReasonInvalidPath,
		},
		{
			name:   "nul",
			path:   "safe.go\x00",
			reason: EvidenceRehydrateReasonInvalidPath,
		},
		{
			name:   "trailing newline",
			path:   "safe.go\n",
			reason: EvidenceRehydrateReasonInvalidPath,
		},
		{
			name:   "symlink escape",
			path:   "link.go",
			reason: EvidenceRehydrateReasonPathEscape,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RehydrateEvidencePointer(context.Background(), EvidencePointer{
				Path:      tt.path,
				StartLine: 1,
				EndLine:   1,
			}, EvidenceRehydrateOptions{RepoRoot: root, InvocationCWD: root})
			assertEvidenceRehydrateFailure(t, result, err, tt.reason)
		})
	}
}

func TestRehydrateEvidencePointer_ReturnsStructuredFileAndRangeErrors(t *testing.T) {
	root := t.TempDir()
	writeLedgerTestFile(t, root, "one.go", "one\n")
	writeLedgerTestFile(t, root, "empty.go", "")
	writeLedgerTestFile(t, root, "binary.go", "a\x00b\n")
	if err := os.Mkdir(filepath.Join(root, "dir"), 0o755); err != nil {
		t.Fatalf("mkdir dir: %v", err)
	}

	tests := []struct {
		name    string
		pointer EvidencePointer
		reason  EvidenceRehydrateErrorReason
	}{
		{
			name:    "missing file",
			pointer: EvidencePointer{Path: "missing.go", StartLine: 1, EndLine: 1},
			reason:  EvidenceRehydrateReasonMissingFile,
		},
		{
			name:    "zero start line",
			pointer: EvidencePointer{Path: "one.go", StartLine: 0, EndLine: 1},
			reason:  EvidenceRehydrateReasonInvalidRange,
		},
		{
			name:    "end before start",
			pointer: EvidencePointer{Path: "one.go", StartLine: 2, EndLine: 1},
			reason:  EvidenceRehydrateReasonInvalidRange,
		},
		{
			name:    "empty file",
			pointer: EvidencePointer{Path: "empty.go", StartLine: 1, EndLine: 1},
			reason:  EvidenceRehydrateReasonRangeOutOfBounds,
		},
		{
			name:    "range out of bounds",
			pointer: EvidencePointer{Path: "one.go", StartLine: 1, EndLine: 2},
			reason:  EvidenceRehydrateReasonRangeOutOfBounds,
		},
		{
			name:    "directory",
			pointer: EvidencePointer{Path: "dir", StartLine: 1, EndLine: 1},
			reason:  EvidenceRehydrateReasonNotRegularFile,
		},
		{
			name:    "binary file",
			pointer: EvidencePointer{Path: "binary.go", StartLine: 1, EndLine: 1},
			reason:  EvidenceRehydrateReasonBinaryFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RehydrateEvidencePointer(context.Background(), tt.pointer, EvidenceRehydrateOptions{
				RepoRoot:      root,
				InvocationCWD: root,
			})
			assertEvidenceRehydrateFailure(t, result, err, tt.reason)
		})
	}
}

func TestRehydrateEvidencePointer_StaleUsesCurrentFileHash(t *testing.T) {
	root := t.TempDir()
	writeLedgerTestFile(t, root, "hash.go", "line\n")
	opts := EvidenceRehydrateOptions{RepoRoot: root, InvocationCWD: root}
	pointer := EvidencePointer{Path: "hash.go", StartLine: 1, EndLine: 1}

	withoutHash, err := RehydrateEvidencePointer(context.Background(), pointer, opts)
	if err != nil {
		t.Fatalf("RehydrateEvidencePointer(empty hash) error = %v", err)
	}
	if withoutHash.CurrentFileHash == "" || withoutHash.Stale {
		t.Fatalf("empty hash result = %#v, want current hash and stale=false", withoutHash)
	}

	pointer.FileHash = withoutHash.CurrentFileHash
	matchingHash, err := RehydrateEvidencePointer(context.Background(), pointer, opts)
	if err != nil {
		t.Fatalf("RehydrateEvidencePointer(matching hash) error = %v", err)
	}
	if matchingHash.Stale {
		t.Fatalf("matching hash stale = true, want false")
	}

	pointer.FileHash = "sha256:0000"
	mismatchedHash, err := RehydrateEvidencePointer(context.Background(), pointer, opts)
	if err != nil {
		t.Fatalf("RehydrateEvidencePointer(mismatched hash) error = %v", err)
	}
	if !mismatchedHash.Stale {
		t.Fatalf("mismatched hash stale = false, want true")
	}
}

func TestRehydrateEvidencePointer_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := RehydrateEvidencePointer(ctx, EvidencePointer{
		Path:      "file.go",
		StartLine: 1,
		EndLine:   1,
	}, EvidenceRehydrateOptions{RepoRoot: t.TempDir(), InvocationCWD: t.TempDir()})

	assertEvidenceRehydrateFailure(t, result, err, EvidenceRehydrateReasonContextCancelled)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want errors.Is(context.Canceled)", err)
	}
}

func writeLedgerTestFile(t *testing.T, root, relativePath, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func makeLedgerTestDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func assertEvidenceRehydrateReason(t *testing.T, err error, want EvidenceRehydrateErrorReason) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want reason %q", want)
	}
	var rehydrateErr *EvidenceRehydrateError
	if !errors.As(err, &rehydrateErr) {
		t.Fatalf("error = %T %v, want *EvidenceRehydrateError", err, err)
	}
	if rehydrateErr.Reason != want {
		t.Fatalf("error reason = %q, want %q", rehydrateErr.Reason, want)
	}
}

func assertEvidenceRehydrateFailure(t *testing.T, result EvidenceRehydrateResult, err error, want EvidenceRehydrateErrorReason) {
	t.Helper()
	assertEvidenceRehydrateReason(t, err, want)
	if result.Reason != want {
		t.Fatalf("result reason = %q, want %q", result.Reason, want)
	}
}

func ledgerEvidencePointerForTest(path string, startLine, endLine int) EvidencePointer {
	state := RuntimeTaskState{
		Evidence: Evidence{items: []evidenceFact{{
			path:      path,
			startLine: startLine,
			endLine:   endLine,
			source:    "read_file",
			excerpt:   "evidence",
		}}},
	}
	return EvidencePointersFromState(state)[0]
}
