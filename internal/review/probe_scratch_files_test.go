package review

import (
	"errors"
	"testing"
)

func TestValidateAndBuildScratchFiles_AllowsRelativePaths(t *testing.T) {
	scratchDir := t.TempDir()

	files, err := validateAndBuildScratchFiles(scratchDir, []ReviewProbeFile{
		{Path: "check.py", Content: "print('ok')\n"},
		{Path: "scripts/check.go", Content: "package main\n"},
	})
	if err != nil {
		t.Fatalf("validateAndBuildScratchFiles() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
}

func TestValidateAndBuildScratchFiles_BlocksAbsolutePath(t *testing.T) {
	scratchDir := t.TempDir()

	_, err := validateAndBuildScratchFiles(scratchDir, []ReviewProbeFile{
		{Path: "/tmp/escape.py", Content: "print('x')\n"},
	})
	if err == nil {
		t.Fatal("validateAndBuildScratchFiles() error = nil")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("validateAndBuildScratchFiles() error = %v, want ErrHostReadOnlyBlocked", err)
	}
}

func TestValidateAndBuildScratchFiles_BlocksEscapePath(t *testing.T) {
	scratchDir := t.TempDir()

	_, err := validateAndBuildScratchFiles(scratchDir, []ReviewProbeFile{
		{Path: "../escape.py", Content: "print('x')\n"},
	})
	if err == nil {
		t.Fatal("validateAndBuildScratchFiles() error = nil")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("validateAndBuildScratchFiles() error = %v, want ErrHostReadOnlyBlocked", err)
	}
}

func TestValidateAndBuildScratchFiles_BlocksDuplicatePaths(t *testing.T) {
	scratchDir := t.TempDir()

	_, err := validateAndBuildScratchFiles(scratchDir, []ReviewProbeFile{
		{Path: "check.py", Content: "print('a')\n"},
		{Path: "check.py", Content: "print('b')\n"},
	})
	if err == nil {
		t.Fatal("validateAndBuildScratchFiles() error = nil")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("validateAndBuildScratchFiles() error = %v, want ErrHostReadOnlyBlocked", err)
	}
}
