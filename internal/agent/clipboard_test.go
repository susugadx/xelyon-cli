package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsWSLEnvironment_UsesEnv(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")
	t.Setenv("WSL_INTEROP", "")

	if !isWSLEnvironment() {
		t.Fatal("isWSLEnvironment() = false, want true")
	}
}

func TestIsWSLEnvironment_UsesProcVersion(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "")
	t.Setenv("WSL_INTEROP", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "version")
	if err := os.WriteFile(path, []byte("Linux version 5.15.0-microsoft-standard-WSL2"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	oldPath := wslProcVersionPath
	wslProcVersionPath = path
	t.Cleanup(func() {
		wslProcVersionPath = oldPath
	})

	if !isWSLEnvironment() {
		t.Fatal("isWSLEnvironment() = false, want true")
	}
}

func TestWriteClipboardTextWSL_UsesPowerShellUTF8Pipe(t *testing.T) {
	oldLookPath := execLookPath
	oldRun := runCommandWithInput
	execLookPath = func(file string) (string, error) {
		if file != "powershell.exe" {
			t.Fatalf("LookPath file = %q, want powershell.exe", file)
		}
		return "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe", nil
	}

	var gotName string
	var gotArgs []string
	var gotInput string
	runCommandWithInput = func(ctx context.Context, name string, args []string, input string) error {
		gotName = name
		gotArgs = append([]string(nil), args...)
		gotInput = input
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("context should have deadline")
		}
		return nil
	}

	t.Cleanup(func() {
		execLookPath = oldLookPath
		runCommandWithInput = oldRun
	})

	input := "こんにちは世界"
	if err := writeClipboardTextWSL(input); err != nil {
		t.Fatalf("writeClipboardTextWSL() error = %v", err)
	}

	if !strings.HasSuffix(gotName, "powershell.exe") {
		t.Fatalf("command name = %q, want powershell.exe", gotName)
	}
	if gotInput != input {
		t.Fatalf("stdin input = %q, want %q", gotInput, input)
	}
	joined := strings.Join(gotArgs, " ")
	if !strings.Contains(joined, "Set-Clipboard") {
		t.Fatalf("args should contain Set-Clipboard, got %q", joined)
	}
	if !strings.Contains(joined, "UTF8Encoding") {
		t.Fatalf("args should configure UTF-8 input, got %q", joined)
	}
}

func TestWriteClipboardText_PrefersWSLWriter(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")

	oldWSLPath := wslProcVersionPath
	oldLookPath := execLookPath
	oldRun := runCommandWithInput
	oldClipboard := clipboardLibWriteAll

	wslProcVersionPath = filepath.Join(t.TempDir(), "missing")
	execLookPath = func(file string) (string, error) { return "powershell.exe", nil }

	var wslCalls int
	runCommandWithInput = func(context.Context, string, []string, string) error {
		wslCalls++
		return nil
	}

	var fallbackCalls int
	clipboardLibWriteAll = func(string) error {
		fallbackCalls++
		return nil
	}

	t.Cleanup(func() {
		wslProcVersionPath = oldWSLPath
		execLookPath = oldLookPath
		runCommandWithInput = oldRun
		clipboardLibWriteAll = oldClipboard
	})

	if err := writeClipboardText("日本語"); err != nil {
		t.Fatalf("writeClipboardText() error = %v", err)
	}

	if wslCalls != 1 {
		t.Fatalf("WSL writer calls = %d, want 1", wslCalls)
	}
	if fallbackCalls != 0 {
		t.Fatalf("fallback calls = %d, want 0", fallbackCalls)
	}
}
