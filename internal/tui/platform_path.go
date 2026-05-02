package tui

import (
	"os"
	"os/exec"
	"strings"
)

var (
	execLookPathTUI     = exec.LookPath
	runCommandOutputTUI = func(name string, args ...string) ([]byte, error) {
		return exec.Command(name, args...).Output()
	}
	wslProcVersionPathTUI = "/proc/version"
)

func isTUIWSLEnvironment() bool {
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}
	data, err := os.ReadFile(wslProcVersionPathTUI)
	if err != nil {
		return false
	}
	version := strings.ToLower(string(data))
	return strings.Contains(version, "microsoft") || strings.Contains(version, "wsl")
}

func convertWindowsPathToWSL(path string) (string, error) {
	if _, err := execLookPathTUI("wslpath"); err != nil {
		return "", err
	}
	out, err := runCommandOutputTUI("wslpath", "-u", path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func convertWSLPathToWindows(path string) (string, error) {
	if _, err := execLookPathTUI("wslpath"); err != nil {
		return "", err
	}
	out, err := runCommandOutputTUI("wslpath", "-w", path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
