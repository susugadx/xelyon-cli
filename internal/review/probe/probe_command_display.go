package probe

import (
	"strconv"
	"strings"
	"unicode"
)

// FormatProbeCommand は command と args を人間向け表示文字列へ整形する。
func FormatProbeCommand(command string, args []string) string {
	command = strings.TrimSpace(command)
	if command == "" && len(args) == 0 {
		return ""
	}
	parts := make([]string, 0, 1+len(args))
	if command != "" {
		parts = append(parts, command)
	}
	for _, arg := range args {
		parts = append(parts, formatProbeCommandArg(arg))
	}
	return strings.Join(parts, " ")
}

func formatProbeCommand(command string, args []string) string {
	return FormatProbeCommand(command, args)
}

func formatProbeCommandArg(arg string) string {
	if !probeCommandArgNeedsShellQuote(arg) {
		return arg
	}
	return strconv.Quote(arg)
}

func probeCommandArgNeedsShellQuote(arg string) bool {
	if arg == "" {
		return true
	}
	for _, r := range arg {
		if unicode.IsSpace(r) {
			return true
		}
		switch r {
		case '\n', '\r', ';', '|', '&', '<', '>', '`':
			return true
		}
	}
	return strings.Contains(arg, "$(")
}
