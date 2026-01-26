//go:build !windows

package ui

// enableVirtualTerminalProcessing is a no-op on Unix systems
// ANSI escape sequences work natively on Unix terminals
func enableVirtualTerminalProcessing() {
	// No-op on Unix
}

func init() {
	enableVirtualTerminalProcessing()
}
