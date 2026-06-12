package agent

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/version"
	"golang.org/x/term"
)

const (
	logoStyleReset         = "\033[0m"
	logoBorderColor        = "\033[38;5;240m"
	logoDimColor           = "\033[38;5;245m"
	logoTextColor          = "\033[38;5;255m"
	logoReadyColor         = "\033[38;2;0;215;255m"
	logoCommandAccentColor = "\033[38;5;228m"
	startupPanelInnerWidth = 57
)

func buildLogo() string {
	return buildStartupPanelForWidth(detectedTerminalWidth())
}

func buildStartupPanelForWidth(width int) string {
	colorEnabled := logoANSIEnabled()
	if width > 0 && width < startupPanelWidth() {
		return strings.Join(compactStartupPanelLines(colorEnabled), "\n")
	}

	return strings.Join(startupPanelLines(colorEnabled), "\n")
}

func startupPanelLines(colorEnabled bool) []string {
	return []string{
		startupPanelTopLine(colorEnabled),
		startupPanelBodyLine(
			fmt.Sprintf(" v%s · code-guided agent runtime", version.GetVersion()),
			colorize(" ", logoDimColor, colorEnabled)+startupVersionLine(colorEnabled),
			colorEnabled,
		),
		startupPanelBodyLine("", "", colorEnabled),
		startupPanelBodyLine(
			"  Built to keep agents grounded in your codebase.",
			colorize("  Built to keep agents grounded in your codebase.", logoTextColor, colorEnabled),
			colorEnabled,
		),
		startupPanelBodyLine("", "", colorEnabled),
		startupPanelBodyLine(
			"  Ready · / opens commands · /exit quits",
			startupReadyLine("  ", colorEnabled),
			colorEnabled,
		),
		startupPanelBottomLine(colorEnabled),
	}
}

func compactStartupPanelLines(colorEnabled bool) []string {
	return []string{
		startupWordmark(colorEnabled),
		startupVersionLine(colorEnabled),
		colorize("Built to keep agents grounded in your codebase.", logoTextColor, colorEnabled),
		startupReadyLine("", colorEnabled),
	}
}

func startupPanelTopLine(colorEnabled bool) string {
	titlePrefix := "╭─ "
	titleSuffix := " " + strings.Repeat("─", startupPanelInnerWidth-visibleWidth("─ XELYON "))
	return colorize(titlePrefix, logoBorderColor, colorEnabled) +
		startupWordmark(colorEnabled) +
		colorize(titleSuffix+"╮", logoBorderColor, colorEnabled)
}

func startupPanelBottomLine(colorEnabled bool) string {
	return colorize("╰"+strings.Repeat("─", startupPanelInnerWidth)+"╯", logoBorderColor, colorEnabled)
}

func startupPanelBodyLine(plain, styled string, colorEnabled bool) string {
	if styled == "" {
		styled = plain
	}
	padding := startupPanelInnerWidth - visibleWidth(plain)
	if padding < 0 {
		padding = 0
	}
	return colorize("│", logoBorderColor, colorEnabled) +
		styled +
		strings.Repeat(" ", padding) +
		colorize("│", logoBorderColor, colorEnabled)
}

func startupVersionLine(colorEnabled bool) string {
	return colorize(fmt.Sprintf("v%s · ", version.GetVersion()), logoDimColor, colorEnabled) +
		colorize("code-guided agent runtime", logoTextColor, colorEnabled)
}

func startupReadyLine(indent string, colorEnabled bool) string {
	return colorize(indent+"Ready", logoReadyColor, colorEnabled) +
		colorize(" · ", logoDimColor, colorEnabled) +
		colorize("/", logoCommandAccentColor, colorEnabled) +
		colorize(" opens commands · ", logoDimColor, colorEnabled) +
		colorize("/exit", logoCommandAccentColor, colorEnabled) +
		colorize(" quits", logoDimColor, colorEnabled)
}

func startupWordmark(colorEnabled bool) string {
	if !colorEnabled {
		return "XELYON"
	}
	colors := []string{
		"\033[38;2;0;74;255m",
		"\033[38;2;0;103;255m",
		"\033[38;2;0;132;255m",
		"\033[38;2;0;161;255m",
		"\033[38;2;0;194;255m",
		"\033[38;2;0;215;255m",
	}
	letters := []rune("XELYON")
	var b strings.Builder
	for i, r := range letters {
		b.WriteString(colors[i])
		b.WriteRune(r)
	}
	b.WriteString(logoStyleReset)
	return b.String()
}

func colorize(text, color string, enabled bool) string {
	if !enabled || text == "" {
		return text
	}
	return color + text + logoStyleReset
}

func startupPanelWidth() int {
	return startupPanelInnerWidth + 2
}

func visibleWidth(text string) int {
	return len([]rune(text))
}

func logoANSIEnabled() bool {
	return os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
}

func detectedTerminalWidth() int {
	if columns := strings.TrimSpace(os.Getenv("COLUMNS")); columns != "" {
		width, err := strconv.Atoi(columns)
		if err == nil && width > 0 {
			return width
		}
	}
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0
	}
	return width
}

// buildGradientHeader は interactive surface の起動パネルを構築する。
func buildGradientHeader() string {
	return "\n" + buildLogo() + "\n\n"
}
