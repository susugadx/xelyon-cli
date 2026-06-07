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

		sectionHeader := buildExampleSectionHeaderComment(info)
		keyNode.HeadComment = mergeNodeHeadComment(keyNode.HeadComment, sectionHeader)
		if sectionHeader == "" {
			keyNode.HeadComment = mergeNodeHeadComment(keyNode.HeadComment, topLevelFieldComment(info, keyNode.Value))
		}
		if valueNode.Kind != yaml.MappingNode {
			continue
		}
		annotateExampleFieldComments(valueNode, info.Fields, "")
		annotateExampleCommentedFields(valueNode, info.Fields, info.Example.CommentedFields)
	}
}

func topLevelFieldComment(info SectionInfo, key string) string {
	if len(info.Fields) == 0 {
		return ""
	}
	return strings.TrimSpace(info.Fields[key])
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

func annotateExampleCommentedFields(mapping *yaml.Node, fields map[string]string, commentedFields map[string]CommentedExampleField) {
	if mapping == nil || mapping.Kind != yaml.MappingNode || len(commentedFields) == 0 {
		return
	}
	for path, field := range commentedFields {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if findExampleKeyNodeByPath(mapping, path) != nil {
			continue
		}
		anchorKey := findExampleCommentAnchor(mapping, path, strings.TrimSpace(field.Before))
		if anchorKey == nil {
			continue
		}
		comment := buildCommentedExampleFieldComment(path, fields[path], field.Value)
		anchorKey.HeadComment = mergeNodeHeadComment(comment, anchorKey.HeadComment)
	}
}

func buildCommentedExampleFieldComment(path, fieldComment, value string) string {
	var lines []string
	if fieldComment = strings.TrimSpace(fieldComment); fieldComment != "" {
		lines = append(lines, fieldComment)
	}
	key := path
	if parts := strings.Split(path, "."); len(parts) > 0 {
		key = parts[len(parts)-1]
	}
	value = strings.TrimSpace(value)
	if value == "" {
		lines = append(lines, key+":")
	} else {
		lines = append(lines, key+": "+value)
	}
	return strings.Join(lines, "\n")
}

func findExampleCommentAnchor(mapping *yaml.Node, path, beforePath string) *yaml.Node {
	if beforePath != "" {
		if keyNode := findExampleKeyNodeByPath(mapping, beforePath); keyNode != nil {
			return keyNode
		}
	}
	parentPath, _ := splitExampleFieldPath(path)
	parent := findExampleMappingByPath(mapping, parentPath)
	if parent == nil || parent.Kind != yaml.MappingNode || len(parent.Content) == 0 {
		return nil
	}
	return parent.Content[0]
}

func findExampleKeyNodeByPath(mapping *yaml.Node, path string) *yaml.Node {
	parentPath, key := splitExampleFieldPath(path)
	parent := findExampleMappingByPath(mapping, parentPath)
	if parent == nil || parent.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(parent.Content); i += 2 {
		if parent.Content[i].Value == key {
			return parent.Content[i]
		}
	}
	return nil
}

func findExampleMappingByPath(mapping *yaml.Node, path []string) *yaml.Node {
	current := mapping
	for _, part := range path {
		if current == nil || current.Kind != yaml.MappingNode {
			return nil
		}
		var next *yaml.Node
		for i := 0; i+1 < len(current.Content); i += 2 {
			if current.Content[i].Value == part {
				next = current.Content[i+1]
				break
			}
		}
		current = next
	}
	return current
}

func splitExampleFieldPath(path string) ([]string, string) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, ""
	}
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	if len(cleaned) == 0 {
		return nil, ""
	}
	return cleaned[:len(cleaned)-1], cleaned[len(cleaned)-1]
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
