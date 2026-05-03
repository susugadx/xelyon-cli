package review

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestValidateAndBuildScratchFiles_BlocksOverwriteExistingFile(t *testing.T) {
	scratchDir := t.TempDir()
	writeTestFile(t, filepath.Join(scratchDir, "check.py"), "existing")

	_, err := validateAndBuildScratchFiles(scratchDir, []ReviewProbeFile{
		{Path: "check.py", Content: "print('x')\n"},
	})
	if err == nil {
		t.Fatal("validateAndBuildScratchFiles() error = nil")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("validateAndBuildScratchFiles() error = %v, want ErrHostReadOnlyBlocked", err)
	}
	if !strings.Contains(err.Error(), "would overwrite existing file") {
		t.Fatalf("error = %q, want overwrite block", err.Error())
	}
}

func TestValidateAndBuildScratchFiles_BlocksSymlinkParentEscape(t *testing.T) {
	scratchDir := t.TempDir()
	if err := os.Symlink(t.TempDir(), filepath.Join(scratchDir, "outside-link")); err != nil {
		t.Skipf("symlink is not supported: %v", err)
	}

	_, err := validateAndBuildScratchFiles(scratchDir, []ReviewProbeFile{
		{Path: "outside-link/generated.txt", Content: "x\n"},
	})
	if err == nil {
		t.Fatal("validateAndBuildScratchFiles() error = nil")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("validateAndBuildScratchFiles() error = %v, want ErrHostReadOnlyBlocked", err)
	}
	if !strings.Contains(err.Error(), "escapes scratch directory") {
		t.Fatalf("error = %q, want symlink escape block", err.Error())
	}
}

func TestValidateAndBuildScratchFiles_BlocksWhenExceedingMaxFiles(t *testing.T) {
	scratchDir := t.TempDir()

	files := make([]ReviewProbeFile, 0, defaultScratchOnlyMaxFiles+1)
	for i := 0; i < defaultScratchOnlyMaxFiles+1; i++ {
		files = append(files, ReviewProbeFile{
			Path:    fmt.Sprintf("file-%d.txt", i),
			Content: "ok",
		})
	}

	_, err := validateAndBuildScratchFiles(scratchDir, files)
	if err == nil {
		t.Fatal("validateAndBuildScratchFiles() error = nil")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("validateAndBuildScratchFiles() error = %v, want ErrHostReadOnlyBlocked", err)
	}
}

func TestValidateAndBuildScratchFiles_BlocksWhenExceedingSingleFileBytes(t *testing.T) {
	scratchDir := t.TempDir()

	_, err := validateAndBuildScratchFiles(scratchDir, []ReviewProbeFile{
		{
			Path:    "large.py",
			Content: strings.Repeat("x", defaultScratchOnlyMaxFileBytes+1),
		},
	})
	if err == nil {
		t.Fatal("validateAndBuildScratchFiles() error = nil")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("validateAndBuildScratchFiles() error = %v, want ErrHostReadOnlyBlocked", err)
	}
}

func TestValidateAndBuildScratchFiles_BlocksWhenExceedingTotalFileBytes(t *testing.T) {
	scratchDir := t.TempDir()

	content := strings.Repeat("x", defaultScratchOnlyMaxFileBytes)
	files := []ReviewProbeFile{
		{Path: "a.txt", Content: content},
		{Path: "b.txt", Content: content},
		{Path: "c.txt", Content: content},
		{Path: "d.txt", Content: content},
		{Path: "e.txt", Content: content},
	}
	_, err := validateAndBuildScratchFiles(scratchDir, files)
	if err == nil {
		t.Fatal("validateAndBuildScratchFiles() error = nil")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("validateAndBuildScratchFiles() error = %v, want ErrHostReadOnlyBlocked", err)
	}
}
