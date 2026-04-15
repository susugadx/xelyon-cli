//go:build windows

package agent

import "os/exec"

func setFinalCheckProcessGroup(cmd *exec.Cmd) {
	// Windows では追加設定なし。
}

func killFinalCheckProcessGroup(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
