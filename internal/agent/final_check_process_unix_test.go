//go:build !windows

package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestRunFinalCheckCommands_TimeoutKillsBackgroundChildren(t *testing.T) {
	cfg := config.DefaultConfig()
	pidFile := filepath.Join(t.TempDir(), "sleep.pid")
	cfg.FinalChecks.Commands = []string{
		fmt.Sprintf("sleep 30 & echo $! > %q; wait", pidFile),
	}
	cfg.FinalChecks.Timeout = 1

	a := newCompletionTestAgent(cfg)
	start := time.Now()
	result := a.runFinalCheckCommands([]string{"/src/main.go"})
	if !result.needsContinue {
		t.Fatal("expected needsContinue=true for timed-out command")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("timeout handling took too long: %s", elapsed)
	}

	rawPID, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("failed to read child pid: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(rawPID)))
	if err != nil {
		t.Fatalf("failed to parse child pid %q: %v", strings.TrimSpace(string(rawPID)), err)
	}

	deadline := time.Now().Add(500 * time.Millisecond)
	for processExists(pid) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if processExists(pid) {
		t.Fatalf("timed-out final check child process is still running: pid=%d", pid)
	}
}

func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
