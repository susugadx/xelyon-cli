package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestRepoMapDiskCache_SaveLoad_Hit(t *testing.T) {
	_ = testutil.SetupTempHome(t)

	projectDir := t.TempDir()

	// Create a file to establish a modTime fingerprint.
	p := filepath.Join(projectDir, "main.go")
	if err := os.WriteFile(p, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	fp, err := ComputeProjectFingerprint(projectDir)
	if err != nil {
		t.Fatalf("ComputeProjectFingerprint: %v", err)
	}

	want := "repomap text"
	if err := SaveRepoMapToDisk(projectDir, fp, want); err != nil {
		t.Fatalf("SaveRepoMapToDisk: %v", err)
	}

	got, err := LoadRepoMapFromDisk(projectDir, fp)
	if err != nil {
		t.Fatalf("LoadRepoMapFromDisk: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected repomap: got=%q want=%q", got, want)
	}
}

func TestRepoMapDiskCache_Load_MissOnFingerprintMismatch(t *testing.T) {
	_ = testutil.SetupTempHome(t)

	projectDir := t.TempDir()
	p := filepath.Join(projectDir, "a.txt")
	if err := os.WriteFile(p, []byte("a"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	fp, err := ComputeProjectFingerprint(projectDir)
	if err != nil {
		t.Fatalf("ComputeProjectFingerprint: %v", err)
	}

	if err := SaveRepoMapToDisk(projectDir, fp, "repomap"); err != nil {
		t.Fatalf("SaveRepoMapToDisk: %v", err)
	}

	// Advance modTime by explicitly setting a future time.
	// Using os.Chtimes ensures the modTime changes regardless of filesystem precision.
	futureTime := time.Now().Add(1 * time.Second)
	if err := os.Chtimes(p, futureTime, futureTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}

	fp2, err := ComputeProjectFingerprint(projectDir)
	if err != nil {
		t.Fatalf("ComputeProjectFingerprint2: %v", err)
	}
	if fp2 == fp {
		t.Fatalf("expected fingerprint to change")
	}

	_, err = LoadRepoMapFromDisk(projectDir, fp2)
	if err == nil {
		t.Fatalf("expected cache miss")
	}
	if err != ErrRepoMapCacheMiss {
		t.Fatalf("expected ErrRepoMapCacheMiss, got %v", err)
	}
}

func TestComputeProjectFingerprint_ExcludesGitDir(t *testing.T) {
	projectDir := t.TempDir()

	// Create a .git dir with a new file; it should be ignored.
	gitDir := filepath.Join(projectDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	gitFile := filepath.Join(gitDir, "index")
	if err := os.WriteFile(gitFile, []byte("x"), 0644); err != nil {
		t.Fatalf("write .git file: %v", err)
	}

	// Create a normal file older than .git file; fingerprint should not depend on .git.
	normal := filepath.Join(projectDir, "file.txt")
	if err := os.WriteFile(normal, []byte("n"), 0644); err != nil {
		t.Fatalf("write normal: %v", err)
	}

	fp, err := ComputeProjectFingerprint(projectDir)
	if err != nil {
		t.Fatalf("ComputeProjectFingerprint: %v", err)
	}
	if fp == 0 {
		t.Fatalf("expected non-zero fingerprint")
	}
}
