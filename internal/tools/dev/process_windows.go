//go:build windows

package dev

import (
	"os/exec"
)

// setSysProcAttr はWindowsでは何もしない（Setpgidはサポートされていない）
func setSysProcAttr(cmd *exec.Cmd) {
	// Windows does not support Setpgid
}

// killProcessGroup はWindowsではプロセスのみを終了
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
