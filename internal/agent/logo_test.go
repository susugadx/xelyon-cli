package agent

import (
	"strings"
	"testing"
)

func TestBuildStartupPanelForWidthIncludesBoxedHeaderWhenWide(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	got := buildStartupPanelForWidth(startupPanelWidth())

	for _, want := range []string{
		"╭─ XELYON ",
		"vdev · code-guided agent runtime",
		"Built to keep agents grounded in your codebase.",
		"Ready · / opens commands · /exit quits",
		"╰",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("buildStartupPanelForWidth() missing %q:\n%s", want, got)
		}
	}

	if strings.Contains(got, "XELYON CLI") || strings.Contains(got, "AI-powered") {
		t.Fatalf("buildStartupPanelForWidth() kept stale startup copy:\n%s", got)
	}
}

func TestBuildStartupPanelForWidthHidesBoxWhenNarrow(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	got := buildStartupPanelForWidth(startupPanelWidth() - 1)

	if strings.ContainsAny(got, "╭╮╰╯│") {
		t.Fatalf("buildStartupPanelForWidth() included the box in narrow mode:\n%s", got)
	}
	for _, want := range []string{
		"XELYON",
		"vdev · code-guided agent runtime",
		"Built to keep agents grounded in your codebase.",
		"Ready · / opens commands · /exit quits",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("buildStartupPanelForWidth() missing compact %q:\n%s", want, got)
		}
	}
}

func TestBuildGradientHeaderNoColorDoesNotEmitANSI(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("COLUMNS", "120")

	got := buildGradientHeader()

	if strings.Contains(got, "\x1b[") {
		t.Fatalf("buildGradientHeader() emitted ANSI despite NO_COLOR: %q", got)
	}
	if !strings.Contains(got, "Ready · / opens commands · /exit quits") {
		t.Fatalf("buildGradientHeader() did not preserve the startup hint:\n%s", got)
	}
}
