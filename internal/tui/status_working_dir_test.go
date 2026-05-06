package tui

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

func TestFormatWorkingDirForStatus_UsesHomeShorthand(t *testing.T) {
	got := formatWorkingDirForStatusWithHome("/home/me/dev/app", "/home/me")
	if got != "~/dev/app" {
		t.Fatalf("formatWorkingDirForStatusWithHome() = %q, want %q", got, "~/dev/app")
	}
}

func TestSanitizeWorkingDirStatusPath_RemovesANSIAndFlattensControls(t *testing.T) {
	got := sanitizeWorkingDirStatusPath("/tmp/repo\nbranch/\033[31mred\tfile")
	if got != "/tmp/repo branch/red file" {
		t.Fatalf("sanitizeWorkingDirStatusPath() = %q, want %q", got, "/tmp/repo branch/red file")
	}
	if strings.ContainsAny(got, "\r\n\t\033") {
		t.Fatalf("sanitized path should not contain control chars, got %q", got)
	}
}

func TestTruncateWorkingDirStatusPath_PreservesTail(t *testing.T) {
	got := truncateWorkingDirStatusPath("/home/me/dev/xelyon-cli", 15)
	if !strings.HasPrefix(got, pathTruncationMarker) {
		t.Fatalf("truncated path should start with marker, got %q", got)
	}
	if !strings.Contains(got, "xelyon-cli") {
		t.Fatalf("truncated path should preserve path tail, got %q", got)
	}
	if width := termtext.PlainTextDisplayWidth(got); width > 15 {
		t.Fatalf("truncated path width = %d, want <= 15; got %q", width, got)
	}
}
