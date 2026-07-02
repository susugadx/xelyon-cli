//go:build linux

package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRunHeadlessWithConfig_FinalChecksIncludePreexistingLargeDirtyFileChangedWithMtimeRestored(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	dir := testSubDir(t)
	runHeadlessSummaryGit(t, dir, "init")
	runHeadlessSummaryGit(t, dir, "config", "user.email", "test@example.com")
	runHeadlessSummaryGit(t, dir, "config", "user.name", "Test User")
	path := filepath.Join(dir, "large.bin")
	size := headlessGitChangedFileHashLimitBytes + 1
	writeSparseTestFileByte(t, path, size, 0, 'a')
	runHeadlessSummaryGit(t, dir, "add", "large.bin")
	runHeadlessSummaryGit(t, dir, "commit", "-m", "initial")
	writeSparseTestFileByte(t, path, size, 0, 'b')

	cfg := newProjectMapDisabledConfig()
	cfg.FinalChecks.Commands = []string{`case " $XELYON_CHANGED_FILES " in *" large.bin "* ) ;; *) exit 2;; esac`}
	cfg.FinalChecks.Timeout = 10

	command := fmt.Sprintf(
		`stamp=$(stat -c %%y large.bin); printf c | dd of=large.bin bs=1 seek=%d conv=notrunc status=none; touch -d "$stamp" large.bin`,
		size-1,
	)
	provider := &sequenceMockProvider{
		name: "test-provider",
		responses: []string{
			fmt.Sprintf(`{"tool":"bash","args":{"command":%q}}`, command),
			"done",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "mutate large dirty file via bash", "test-model", provider, cfg)

	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	if result.Summary == nil {
		t.Fatal("Summary = nil, want changed_files and final_checks")
	}
	if !slices.Contains(result.Summary.ChangedFiles, "large.bin") {
		t.Fatalf("Summary.ChangedFiles = %v, want large.bin after same-size mutation", result.Summary.ChangedFiles)
	}
	if len(result.Summary.FinalChecks) != 1 || result.Summary.FinalChecks[0].Status != headlessSummaryStatusPassed {
		t.Fatalf("Summary.FinalChecks = %+v, want one passed final check", result.Summary.FinalChecks)
	}
}

func writeSparseTestFileByte(t *testing.T, path string, size int64, offset int64, value byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer file.Close()
	if err := file.Truncate(size); err != nil {
		t.Fatalf("Truncate() error = %v", err)
	}
	if _, err := file.WriteAt([]byte{value}, offset); err != nil {
		t.Fatalf("WriteAt() error = %v", err)
	}
}
