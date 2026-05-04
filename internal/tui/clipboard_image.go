package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const clipboardImageTimeout = 6 * time.Second

var runClipboardCommand = func(ctx context.Context, name string, args []string, env []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	return cmd.CombinedOutput()
}

func saveClipboardImageToTemp() (string, error) {
	if runtime.GOOS != "windows" && !isTUIWSLEnvironment() {
		return "", fmt.Errorf("clipboard image paste is unsupported on this platform")
	}

	dir, err := os.MkdirTemp("", clipboardAttachmentTempDirPrefix)
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	linuxPath := filepath.Join(dir, "clipboard.png")
	targetPath := linuxPath
	if isTUIWSLEnvironment() {
		converted, convErr := convertWSLPathToWindows(linuxPath)
		if convErr != nil {
			_ = os.RemoveAll(dir)
			return "", fmt.Errorf("failed to convert temp path for Windows clipboard: %w", convErr)
		}
		targetPath = converted
	}

	powershellPath, err := findPowerShellCommand()
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	script := strings.Join([]string{
		"Add-Type -AssemblyName System.Windows.Forms;",
		"Add-Type -AssemblyName System.Drawing;",
		"if (-not [System.Windows.Forms.Clipboard]::ContainsImage()) { exit 3 };",
		"$img = [System.Windows.Forms.Clipboard]::GetImage();",
		"if ($null -eq $img) { exit 4 };",
		"$img.Save($env:XELYON_CLIP_IMAGE_PATH, [System.Drawing.Imaging.ImageFormat]::Png);",
	}, " ")

	ctx, cancel := context.WithTimeout(context.Background(), clipboardImageTimeout)
	defer cancel()
	output, runErr := runClipboardCommand(ctx, powershellPath, []string{
		"-NoProfile",
		"-NonInteractive",
		"-Command",
		script,
	}, []string{"XELYON_CLIP_IMAGE_PATH=" + targetPath})
	if runErr != nil {
		_ = os.RemoveAll(dir)
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return "", fmt.Errorf("clipboard image extraction failed: %w", runErr)
		}
		return "", fmt.Errorf("clipboard image extraction failed: %w (%s)", runErr, trimmed)
	}

	info, statErr := os.Stat(linuxPath)
	if statErr != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("clipboard image was not saved: %w", statErr)
	}
	if info.Size() == 0 {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("clipboard image was empty")
	}
	return linuxPath, nil
}

func findPowerShellCommand() (string, error) {
	candidates := []string{"powershell.exe", "pwsh", "powershell"}
	for _, candidate := range candidates {
		if path, err := execLookPathTUI(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("PowerShell was not found")
}
