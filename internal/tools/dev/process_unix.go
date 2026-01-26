//go:build !windows

package dev

import (
	"os/exec"
	"syscall"
)

// setSysProcAttr はUnix系でプロセスグループを設定
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup はプロセスグループ全体を終了
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
