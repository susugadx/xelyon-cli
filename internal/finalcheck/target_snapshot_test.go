package finalcheck

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFingerprintTargetFiles_ChangesWhenFileContentChanges(t *testing.T) {
	workDir := t.TempDir()
	target := filepath.Join(workDir, "main.go")

	if err := os.WriteFile(target, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}
	first := FingerprintTargetFiles([]string{target})
	if first == "" {
		t.Fatal("expected non-empty fingerprint for existing file")
	}

	if err := os.WriteFile(target, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to update file: %v", err)
	}
	second := FingerprintTargetFiles([]string{target})
	if second == "" {
		t.Fatal("expected non-empty fingerprint after content update")
	}
	if second == first {
		t.Fatal("expected fingerprint to change when file content changes")
	}
}

func TestFingerprintTargetFiles_IsOrderIndependent(t *testing.T) {
	workDir := t.TempDir()
	a := filepath.Join(workDir, "a.go")
	b := filepath.Join(workDir, "b.go")

	if err := os.WriteFile(a, []byte("a"), 0o644); err != nil {
		t.Fatalf("failed to write a.go: %v", err)
	}
	if err := os.WriteFile(b, []byte("b"), 0o644); err != nil {
		t.Fatalf("failed to write b.go: %v", err)
	}

	first := FingerprintTargetFiles([]string{a, b})
	second := FingerprintTargetFiles([]string{b, a})
	if first == "" || second == "" {
		t.Fatalf("expected non-empty fingerprints: first=%q second=%q", first, second)
	}
	if first != second {
		t.Fatalf("expected order-independent fingerprint, got first=%q second=%q", first, second)
	}
}

func TestFingerprintTargetFiles_UsesStableMarkerForMissingPath(t *testing.T) {
	target := filepath.Join(t.TempDir(), "missing.go")
	first := FingerprintTargetFiles([]string{target})
	second := FingerprintTargetFiles([]string{target})
	if first == "" || second == "" {
		t.Fatalf("expected non-empty fingerprints for missing path: first=%q second=%q", first, second)
	}
	if first != second {
		t.Fatalf("expected stable fingerprint for missing path: first=%q second=%q", first, second)
	}
}

func TestFingerprintTargetFiles_UsesStableMarkerForUnreadablePath(t *testing.T) {
	workDir := t.TempDir()
	first := FingerprintTargetFiles([]string{workDir})
	second := FingerprintTargetFiles([]string{workDir})
	if first == "" || second == "" {
		t.Fatalf("expected non-empty fingerprints for unreadable path: first=%q second=%q", first, second)
	}
	if first != second {
		t.Fatalf("expected stable fingerprint for unreadable path: first=%q second=%q", first, second)
	}
}

func TestBuildTargetSnapshot_FallsBackToMutationFingerprintWhenFilesUnknown(t *testing.T) {
	snapshot := BuildTargetSnapshot(TargetInput{ProgressFingerprint: "turn-progress"})
	if snapshot.ProgressFingerprint != "turn-progress" {
		t.Fatalf("ProgressFingerprint = %q, want fallback", snapshot.ProgressFingerprint)
	}
}
