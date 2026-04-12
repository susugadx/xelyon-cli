package cmd

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestExecute_HelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_XELYON_ROOT_EXECUTE_HELPER") != "1" {
		return
	}

	resetRootFlagsForTest()
	rootCmd.SetArgs([]string{"--unknown-flag"})
	Execute()
}

func TestExecute_ExitsOnError(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	cmd := exec.Command(exe, "-test.run=TestExecute_HelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_XELYON_ROOT_EXECUTE_HELPER=1")

	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected Execute() helper to exit with non-zero status")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("error = %T, want *exec.ExitError", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("exit code = %d, want 1", exitErr.ExitCode())
	}
	if !strings.Contains(string(output), "unknown flag") {
		t.Fatalf("combined output = %q, want cobra error message", string(output))
	}
}
