package transcript

import "strings"

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

// MessageLines は message role と content から transcript に積む行を生成する。
func MessageLines(role string, content string) []string {
	return messageRenderer(role)(content)
}

func messageRenderer(role string) func(string) []string {
	if role == "user" {
		return userMessageLines
	}
	return plainMessageLines
}

func plainMessageLines(content string) []string {
	return strings.Split(content, "\n")
}

func userMessageLines(content string) []string {
	lines := strings.Split(content, "\n")
	rendered := make([]string, 0, len(lines)+2)
	rendered = append(rendered, "")
	for _, line := range lines {
		rendered = append(rendered, "> "+line)
	}
	rendered = append(rendered, "")
	return rendered
}
