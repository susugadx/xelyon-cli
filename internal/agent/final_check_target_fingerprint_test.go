package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFingerprintFinalCheckTargetFiles_ChangesWhenFileContentChanges(t *testing.T) {
	workDir := t.TempDir()
	target := filepath.Join(workDir, "main.go")

	if err := os.WriteFile(target, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	first := fingerprintFinalCheckTargetFiles([]string{target})
	if first == "" {
		t.Fatal("expected non-empty fingerprint for existing file")
	}

	if err := os.WriteFile(target, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to update file: %v", err)
	}
	second := fingerprintFinalCheckTargetFiles([]string{target})
	if second == "" {
		t.Fatal("expected non-empty fingerprint after content update")
	}
	if second == first {
		t.Fatal("expected fingerprint to change when file content changes")
	}
}

func TestFingerprintFinalCheckTargetFiles_IsOrderIndependent(t *testing.T) {
	workDir := t.TempDir()
	a := filepath.Join(workDir, "a.go")
	b := filepath.Join(workDir, "b.go")

	if err := os.WriteFile(a, []byte("a"), 0o644); err != nil {
		t.Fatalf("failed to write a.go: %v", err)
	}
	if err := os.WriteFile(b, []byte("b"), 0o644); err != nil {
		t.Fatalf("failed to write b.go: %v", err)
	}

	first := fingerprintFinalCheckTargetFiles([]string{a, b})
	second := fingerprintFinalCheckTargetFiles([]string{b, a})
	if first == "" || second == "" {
		t.Fatalf("expected non-empty fingerprints: first=%q second=%q", first, second)
	}
	if first != second {
		t.Fatalf("expected order-independent fingerprint, got first=%q second=%q", first, second)
	}
}

func TestFingerprintFinalCheckTargetFiles_UsesStableMarkerForUnreadablePath(t *testing.T) {
	// ディレクトリを渡すと os.ReadFile は失敗し、<unreadable> マーカー経由で安定 fingerprint になる。
	workDir := t.TempDir()
	first := fingerprintFinalCheckTargetFiles([]string{workDir})
	second := fingerprintFinalCheckTargetFiles([]string{workDir})
	if first == "" || second == "" {
		t.Fatalf("expected non-empty fingerprints for unreadable path: first=%q second=%q", first, second)
	}
	if first != second {
		t.Fatalf("expected stable fingerprint for unreadable path: first=%q second=%q", first, second)
	}
}
