package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/version"
)

const (
	logoForegroundReset = "\033[39m"
)

type logoColorSegment struct {
	start int
	end   int
	color string
}

var logoColorSegments = []logoColorSegment{
	{start: 0, end: 8, color: "\033[38;2;0;109;255m"},
	{start: 8, end: 16, color: "\033[38;2;0;128;255m"},
	{start: 16, end: 24, color: "\033[38;2;0;146;255m"},
	{start: 24, end: 34, color: "\033[38;2;0;160;245m"},
	{start: 34, end: 42, color: "\033[38;2;0;169;238m"},
	{start: 42, end: 52, color: "\033[38;2;0;178;232m"},
}

// logoLines は oh-my-logo で生成した XELYON ロゴ。
// 色は buildLogo で左側ロゴ色に寄せた6文字分のグラデーションを付与する。
//
// 再生成コマンド:
//
//	NO_COLOR=1 npx oh-my-logo "XELYON" --filled --letter-spacing 0 -d horizontal
var logoLines = []string{
	"██╗  ██╗███████╗██╗     ██╗   ██╗ ██████╗ ███╗   ██╗",
	"╚██╗██╔╝██╔════╝██║     ╚██╗ ██╔╝██╔═══██╗████╗  ██║",
	" ╚███╔╝ █████╗  ██║      ╚████╔╝ ██║   ██║██╔██╗ ██║",
	" ██╔██╗ ██╔══╝  ██║       ╚██╔╝  ██║   ██║██║╚██╗██║",
	"██╔╝ ██╗███████╗███████╗   ██║   ╚██████╔╝██║ ╚████║",
	"╚═╝  ╚═╝╚══════╝╚══════╝   ╚═╝    ╚═════╝ ╚═╝  ╚═══╝",
}

func buildLogo() string {
	colored := make([]string, 0, len(logoLines))
	for _, line := range logoLines {
		colored = append(colored, colorizeLogoLine(line))
	}
	return strings.Join(colored, "\n")
}

func colorizeLogoLine(line string) string {
	runes := []rune(line)
	var b strings.Builder
	cursor := 0
	for _, segment := range logoColorSegments {
		if segment.start >= len(runes) {
			continue
		}
		if segment.start > cursor {
			b.WriteString(string(runes[cursor:segment.start]))
		}
		end := min(segment.end, len(runes))
		b.WriteString(segment.color)
		b.WriteString(string(runes[segment.start:end]))
		cursor = end
	}
	if cursor < len(runes) {
		b.WriteString(string(runes[cursor:]))
	}
	b.WriteString(logoForegroundReset)
	return b.String()
}

// buildGradientHeader はロゴ + サブテキストのヘッダーを構築する。
func buildGradientHeader() string {
	return "\n" + buildLogo() + "\n" +
		fmt.Sprintf("  \033[38;5;245mv%s · AI-powered coding agent\033[0m\n", version.GetVersion()) +
		"  \033[38;5;228mType / for commands, /exit to quit\033[0m\n\n"
}
