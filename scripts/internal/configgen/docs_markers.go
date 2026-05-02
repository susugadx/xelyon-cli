package configgen

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// ReplaceMarkerContent replaces the content between two markers.
func ReplaceMarkerContent(content, startMarker, endMarker, newContent string) (string, bool) {
	startIdx := strings.Index(content, startMarker)
	if startIdx < 0 {
		return content, false
	}
	searchStart := startIdx + len(startMarker)
	rest := content[searchStart:]
	endRel := strings.Index(rest, endMarker)
	if endRel < 0 {
		return content, false
	}
	endIdx := searchStart + endRel
	replacement := startMarker + "\n" + newContent + "\n" + endMarker
	updated := content[:startIdx] + replacement + content[endIdx+len(endMarker):]
	return updated, true
}

var configExampleHeaderCommentPrefixes = []string{
	"XELYON CLI 設定例",
	"設定ファイルの場所:",
	"初回起動時に自動的に作成されます",
	"詳細は docs/config.md を参照してください",
}

// FormatConfigExample strips the file header and wraps the example in a YAML code block.
func FormatConfigExample(example string) string {
	if formatted, ok := formatConfigExampleFromYAML(example); ok {
		return formatted
	}
	return "```yaml\n" + extractLegacyYAMLLines(example) + "```"
}

func formatConfigExampleFromYAML(example string) (string, bool) {
	var raw yaml.Node
	if err := yaml.Unmarshal([]byte(example), &raw); err != nil {
		return "", false
	}
	if raw.Kind != yaml.DocumentNode || len(raw.Content) == 0 {
		return "", false
	}
	mapping := raw.Content[0]
	if mapping.Kind != yaml.MappingNode || len(mapping.Content) < 2 {
		return "", false
	}
	raw.HeadComment = stripConfigExampleHeaderComment(raw.HeadComment)

	out, err := yaml.Marshal(&raw)
	if err != nil {
		return "", false
	}
	return "```yaml\n" + string(out) + "```", true
}

func stripConfigExampleHeaderComment(comment string) string {
	if strings.TrimSpace(comment) == "" {
		return ""
	}
	lines := strings.Split(comment, "\n")
	index := 0
	for index < len(lines) {
		line := normalizeCommentLine(lines[index])
		if line == "" {
			index++
			continue
		}
		if hasAnyPrefix(line, configExampleHeaderCommentPrefixes) {
			index++
			continue
		}
		break
	}
	for index < len(lines) && strings.TrimSpace(lines[index]) == "" {
		index++
	}
	return strings.Join(lines[index:], "\n")
}

func hasAnyPrefix(line string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func normalizeCommentLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "#")
	return strings.TrimSpace(line)
}

func extractLegacyYAMLLines(example string) string {
	lines := strings.Split(example, "\n")
	var yamlLines []string
	inYAML := false
	for _, line := range lines {
		if !strings.HasPrefix(line, "#") && strings.TrimSpace(line) != "" {
			inYAML = true
		}
		if inYAML {
			yamlLines = append(yamlLines, line)
		}
	}
	return strings.Join(yamlLines, "\n")
}
