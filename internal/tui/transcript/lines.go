package transcript

import (
	"fmt"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

// NormalizeLine は raw transcript line を表示・レイアウト用に正規化する。
func NormalizeLine(line string) string {
	line = strings.TrimSuffix(line, "\r")
	// VS16 (U+FE0F) を除去して emoji presentation による幅ズレを防ぐ。
	if strings.ContainsRune(line, '\uFE0F') {
		line = strings.ReplaceAll(line, "\uFE0F", "")
	}
	return line
}

// NormalizeLines は複数の raw transcript line を正規化する。
func NormalizeLines(lines []string) []string {
	normalized := make([]string, len(lines))
	for i, line := range lines {
		normalized[i] = NormalizeLine(line)
	}
	return normalized
}

// Message は transcript に積む会話本文の表示モデル。
type Message struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// MessageLines は message role と content から transcript に積む行を生成する。
func MessageLines(role string, content string) []string {
	return Lines(Message{Role: role, Content: content})
}

// Lines は Message から transcript 表示行を生成する。
func Lines(msg Message) []string {
	spec := turnChromeSpecForRole(msg.Role)
	if !spec.Enabled {
		return renderMessageBodyLines(msg.Role, msg.Content)
	}

	timestamp := msg.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	lines := renderMessageBodyLines(msg.Role, msg.Content)
	rendered := make([]string, 0, len(lines)+1)
	rendered = append(rendered, separatorLine(spec, timestamp))
	for _, line := range lines {
		rendered = append(rendered, gutteredLine(spec, line))
	}
	return rendered
}

type turnChromeSpec struct {
	Enabled     bool
	DisplayRole string
	Rule        string
	LinePrefix  string
	HeaderStyle string
}

func turnChromeSpecForRole(role string) turnChromeSpec {
	switch role {
	case "user":
		return turnChromeSpec{
			Enabled:     true,
			DisplayRole: "user",
			Rule:        "━━",
			LinePrefix:  "┃ > ",
			HeaderStyle: theme.Transcript.UserHeader,
		}
	case "assistant":
		return turnChromeSpec{
			Enabled:     true,
			DisplayRole: "assistant",
			Rule:        "──",
			LinePrefix:  "│ ",
			HeaderStyle: theme.Transcript.AssistantHeader,
		}
	case "system_info":
		return turnChromeSpec{
			Enabled:     true,
			DisplayRole: "system",
			Rule:        "┄┄",
			LinePrefix:  "┆ · ",
			HeaderStyle: theme.Transcript.SystemHeader,
		}
	default:
		return turnChromeSpec{}
	}
}

func separatorLine(spec turnChromeSpec, timestamp time.Time) string {
	line := fmt.Sprintf("%s %s · %s · now %s", spec.Rule, spec.DisplayRole, timestamp.Format("15:04"), spec.Rule)
	return styleTranscriptLine(spec.HeaderStyle, line)
}

func gutteredLine(spec turnChromeSpec, line string) string {
	return spec.LinePrefix + line
}

func styleTranscriptLine(style string, line string) string {
	if style == "" {
		return line
	}
	return style + line + theme.Transcript.Reset
}
