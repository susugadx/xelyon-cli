package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/atotto/clipboard"
)

var (
	clipboardWriteAll    = writeClipboardText
	clipboardLibWriteAll = clipboard.WriteAll
	execLookPath         = exec.LookPath
	wslProcVersionPath   = "/proc/version"
	runCommandWithInput  = func(ctx context.Context, name string, args []string, input string) error {
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Stdin = strings.NewReader(input)
		return cmd.Run()
	}
)

func writeClipboardText(text string) error {
	if isWSLEnvironment() {
		if err := writeClipboardTextWSL(text); err == nil {
			return nil
		}
	}
	return clipboardLibWriteAll(text)
}

func isWSLEnvironment() bool {
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}

	data, err := os.ReadFile(wslProcVersionPath)
	if err != nil {
		return false
	}
	version := strings.ToLower(string(data))
	return strings.Contains(version, "microsoft") || strings.Contains(version, "wsl")
}

func writeClipboardTextWSL(text string) error {
	powershellPath, err := execLookPath("powershell.exe")
	if err != nil {
		return err
	}

	const script = "[Console]::InputEncoding=[System.Text.UTF8Encoding]::new($false); $text=[Console]::In.ReadToEnd(); Set-Clipboard -Value $text"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := runCommandWithInput(ctx, powershellPath, []string{
		"-NoProfile",
		"-NonInteractive",
		"-Command",
		script,
	}, text); err != nil {
		return fmt.Errorf("powershell clipboard write failed: %w", err)
	}
	return nil
}
