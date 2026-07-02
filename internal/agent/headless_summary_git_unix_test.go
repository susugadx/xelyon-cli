//go:build !windows

package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
)

func TestHeadlessGitChangedFileStateForPath_DoesNotOpenFIFO(t *testing.T) {
	dir := t.TempDir()
	fifoPath := filepath.Join(dir, "pipe")
	if err := syscall.Mkfifo(fifoPath, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}

	state := headlessGitChangedFileStateForPath(dir, "pipe", "??")

	if !state.exists {
		t.Fatal("state.exists = false, want true for FIFO")
	}
	if state.mode&os.ModeNamedPipe == 0 {
		t.Fatalf("state.mode = %v, want named pipe mode", state.mode)
	}
	if state.hash != "" {
		t.Fatalf("state.hash = %q, want no hash for FIFO", state.hash)
	}
}

func TestRunHeadlessWithConfig_FinalChecksIncludePreexistingDirtyFilePermissionChangedDuringRun(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	dir := testSubDir(t)
	runHeadlessSummaryGit(t, dir, "init")
	runHeadlessSummaryGit(t, dir, "config", "user.email", "test@example.com")
	runHeadlessSummaryGit(t, dir, "config", "user.name", "Test User")
	writeTestFile(t, filepath.Join(dir, "script.sh"), "#!/bin/sh\necho clean\n")
	if err := os.Chmod(filepath.Join(dir, "script.sh"), 0o644); err != nil {
		t.Fatalf("Chmod(clean) error = %v", err)
	}
	runHeadlessSummaryGit(t, dir, "add", "script.sh")
	runHeadlessSummaryGit(t, dir, "commit", "-m", "initial")
	writeTestFile(t, filepath.Join(dir, "script.sh"), "#!/bin/sh\necho dirty before run\n")
	if err := os.Chmod(filepath.Join(dir, "script.sh"), 0o644); err != nil {
		t.Fatalf("Chmod(dirty) error = %v", err)
	}

	cfg := newProjectMapDisabledConfig()
	cfg.FinalChecks.Commands = []string{`case " $XELYON_CHANGED_FILES " in *" script.sh "* ) ;; *) exit 2;; esac`}
	cfg.FinalChecks.Timeout = 10

	provider := &sequenceMockProvider{
		name: "test-provider",
		responses: []string{
			fmt.Sprintf(`{"tool":"bash","args":{"command":%q}}`, "chmod +x script.sh"),
			"done",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "chmod dirty file via bash", "test-model", provider, cfg)

	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	if result.Summary == nil {
		t.Fatal("Summary = nil, want changed_files and final_checks")
	}
	if !slices.Contains(result.Summary.ChangedFiles, "script.sh") {
		t.Fatalf("Summary.ChangedFiles = %v, want script.sh after chmod", result.Summary.ChangedFiles)
	}
	if len(result.Summary.FinalChecks) != 1 || result.Summary.FinalChecks[0].Status != headlessSummaryStatusPassed {
		t.Fatalf("Summary.FinalChecks = %+v, want one passed final check", result.Summary.FinalChecks)
	}
}
