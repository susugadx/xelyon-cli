package configgen

import (
	"strings"

	"gopkg.in/yaml.v3"
)

func annotateExampleSectionComments(mapping *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		valueNode := mapping.Content[i+1]

		info, ok := Sections[keyNode.Value]
		if !ok {
			continue
		}

		keyNode.HeadComment = mergeNodeHeadComment(keyNode.HeadComment, buildExampleSectionHeaderComment(info))
		if valueNode.Kind != yaml.MappingNode {
			continue
		}
		annotateExampleFieldComments(valueNode, info.Fields, "")
	}
}

func annotateExampleFieldComments(mapping *yaml.Node, fields map[string]string, prefix string) {
	if mapping == nil || mapping.Kind != yaml.MappingNode || len(fields) == 0 {
		return
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		valueNode := mapping.Content[i+1]
		key := strings.TrimSpace(keyNode.Value)
		if key == "" {
			continue
		}
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		if comment, ok := fields[path]; ok {
			keyNode.HeadComment = mergeNodeHeadComment(keyNode.HeadComment, strings.TrimSpace(comment))
		}
		if valueNode.Kind == yaml.MappingNode {
			annotateExampleFieldComments(valueNode, fields, path)
		}
	}
}

func buildExampleSectionHeaderComment(info SectionInfo) string {
	var lines []string
	if title := strings.TrimSpace(info.Title); title != "" {
		lines = append(lines, "============================================================")
		lines = append(lines, title)
		lines = append(lines, "============================================================")
	}
	for _, comment := range info.Comments {
		comment = strings.TrimSpace(comment)
		if comment == "" {
			continue
		}
		lines = append(lines, comment)
	}
	return strings.Join(lines, "\n")
}

func mergeNodeHeadComment(existing, additional string) string {
	existing = strings.TrimSpace(existing)
	additional = strings.TrimSpace(additional)
	switch {
	case existing == "":
		return additional
	case additional == "":
		return existing
	default:
		return existing + "\n" + additional
	}
}
