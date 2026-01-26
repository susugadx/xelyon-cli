//go:build windows

package ui

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// enableVirtualTerminalProcessing enables ANSI escape sequence processing on Windows
// This is required for bracketed paste mode to work correctly in CMD.EXE and PowerShell
// Windows Terminal has this enabled by default, but we enable it anyway for consistency
func enableVirtualTerminalProcessing() {
	debug := os.Getenv("XELYON_DEBUG_PASTE") == "1"

	stdout := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(stdout, &mode); err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Windows: GetConsoleMode(stdout) failed: %v\n", err)
		}
		return
	}

	oldMode := mode
	// ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
	mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	if err := windows.SetConsoleMode(stdout, mode); err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Windows: SetConsoleMode(stdout) failed: %v\n", err)
		}
	} else if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Windows: stdout VT enabled (0x%04x -> 0x%04x)\n", oldMode, mode)
	}

	// Also enable for stdin to properly receive bracketed paste sequences
	stdin := windows.Handle(os.Stdin.Fd())
	if err := windows.GetConsoleMode(stdin, &mode); err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Windows: GetConsoleMode(stdin) failed: %v\n", err)
		}
		return
	}

	oldMode = mode
	// ENABLE_VIRTUAL_TERMINAL_INPUT = 0x0200
	mode |= 0x0200
	if err := windows.SetConsoleMode(stdin, mode); err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Windows: SetConsoleMode(stdin) failed: %v\n", err)
		}
	} else if debug {
		fmt.Fprintf(os.Stderr, "[DEBUG] Windows: stdin VT enabled (0x%04x -> 0x%04x)\n", oldMode, mode)
	}
}

func init() {
	enableVirtualTerminalProcessing()
	// 常にログ出力（デバッグ時に init() が呼ばれたか確認）
	if os.Getenv("XELYON_DEBUG_PASTE") == "1" {
		fmt.Fprintln(os.Stderr, "[DEBUG] terminal_windows.go init() completed")
	}
}
