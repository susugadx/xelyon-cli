package transcript

import "strings"

// renderMessageBodyLines は role ごとの本文装飾を適用した transcript 行を生成する。
func renderMessageBodyLines(role string, content string) []string {
	lines := strings.Split(content, "\n")
	if role == "assistant" {
		return renderAssistantBodyLines(lines)
	}
	return lines
}

func renderAssistantBodyLines(lines []string) []string {
	return lines
}
