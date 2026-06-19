package prompt

import (
	"regexp"
	"strings"
)

const (
	ProjectMapStartMarker = "<!-- PROJECT_MAP_START -->"
	ProjectMapEndMarker   = "<!-- PROJECT_MAP_END -->"

	projectMapDataStartTag = "<project_map_data>"
	projectMapDataEndTag   = "</project_map_data>"
)

var projectMapSectionRe = regexp.MustCompile(`(?s)\n?<!-- PROJECT_MAP_START -->\n?(.*?)\n?<!-- PROJECT_MAP_END -->\n?`)

var projectMapDelimiterReplacer = strings.NewReplacer(
	ProjectMapStartMarker, "&lt;!-- PROJECT_MAP_START --&gt;",
	ProjectMapEndMarker, "&lt;!-- PROJECT_MAP_END --&gt;",
	projectMapDataStartTag, "&lt;project_map_data&gt;",
	projectMapDataEndTag, "&lt;/project_map_data&gt;",
)

// BuildProjectMapSection は marker 付き Project Map セクションを構築する。
func BuildProjectMapSection(section string) string {
	section = strings.Trim(section, "\n")
	if strings.TrimSpace(section) == "" {
		return ""
	}
	section = projectMapDelimiterReplacer.Replace(section)
	return ProjectMapStartMarker + "\n" + projectMapDataStartTag + "\n" + section + "\n" + projectMapDataEndTag + "\n" + ProjectMapEndMarker
}

// InjectProjectMapSection は既存 Project Map セクションを置換して末尾に注入する。
func InjectProjectMapSection(systemPrompt, section string) string {
	base := StripProjectMapSection(systemPrompt)
	block := BuildProjectMapSection(section)
	if block == "" {
		return strings.TrimRight(base, "\n")
	}
	base = strings.TrimRight(base, "\n")
	if base == "" {
		return block
	}
	return base + "\n\n" + block
}

// ExtractProjectMapSection は Project Map セクション本文のみを返す。
// marker ブロックのみを解釈する（legacy fallback は行わない）。
func ExtractProjectMapSection(systemPrompt string) string {
	matches := projectMapSectionRe.FindAllStringSubmatch(systemPrompt, -1)
	if len(matches) > 0 {
		last := matches[len(matches)-1]
		if len(last) >= 2 {
			return unwrapProjectMapDataBlock(last[1])
		}
	}
	return ""
}

func unwrapProjectMapDataBlock(section string) string {
	section = strings.Trim(section, "\n")
	if strings.HasPrefix(section, projectMapDataStartTag) && strings.HasSuffix(section, projectMapDataEndTag) {
		section = strings.TrimPrefix(section, projectMapDataStartTag)
		section = strings.TrimSuffix(section, projectMapDataEndTag)
	}
	return strings.Trim(section, "\n")
}

// StripProjectMapSection は Project Map セクションを除去する。
// marker ブロックのみを除去する（legacy fallback は行わない）。
func StripProjectMapSection(systemPrompt string) string {
	stripped := projectMapSectionRe.ReplaceAllString(systemPrompt, "")
	return strings.TrimRight(stripped, "\n")
}

// ExtractProjectMapSectionCompat は marker を優先し、なければ legacy 見出し形式を解釈する。
func ExtractProjectMapSectionCompat(systemPrompt string) string {
	section := ExtractProjectMapSection(systemPrompt)
	if section != "" {
		return section
	}
	return extractLegacyProjectMapSection(systemPrompt)
}

// StripProjectMapSectionCompat は marker を優先し、なければ legacy 見出し形式を除去する。
func StripProjectMapSectionCompat(systemPrompt string) string {
	stripped := StripProjectMapSection(systemPrompt)
	if strings.Contains(stripped, ProjectMapStartMarker) || strings.Contains(stripped, ProjectMapEndMarker) {
		return stripped
	}
	legacySection := extractLegacyProjectMapSection(stripped)
	if legacySection == "" {
		return stripped
	}
	idx := strings.LastIndex(stripped, legacySection)
	if idx < 0 {
		return stripped
	}
	head := strings.TrimRight(stripped[:idx], "\n")
	if idx+len(legacySection) < len(stripped) {
		head += stripped[idx+len(legacySection):]
	}
	return strings.TrimRight(head, "\n")
}

func extractLegacyProjectMapSection(systemPrompt string) string {
	const marker = "## Project Map\n"

	idx := strings.LastIndex(systemPrompt, marker)
	if idx < 0 {
		return ""
	}

	section := systemPrompt[idx:]
	nextSection := strings.Index(section[len(marker):], "\n## ")
	if nextSection >= 0 {
		section = section[:len(marker)+nextSection]
	}

	return strings.TrimRight(section, "\n")
}
