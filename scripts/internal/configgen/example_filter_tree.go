package configgen

import (
	"strings"

	"gopkg.in/yaml.v3"
)

type fieldPathNode struct {
	allowLeaf bool
	children  map[string]*fieldPathNode
}

func buildFieldPathTree(fields map[string]string) *fieldPathNode {
	root := &fieldPathNode{children: map[string]*fieldPathNode{}}
	for path := range fields {
		insertFieldPath(root, path)
	}
	return root
}

func insertFieldPath(root *fieldPathNode, path string) {
	parts := strings.Split(path, ".")
	current := root
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return
		}
		if current.children == nil {
			current.children = map[string]*fieldPathNode{}
		}
		child, ok := current.children[part]
		if !ok {
			child = &fieldPathNode{children: map[string]*fieldPathNode{}}
			current.children[part] = child
		}
		if i == len(parts)-1 {
			child.allowLeaf = true
		}
		current = child
	}
}

type fieldTreeFilterContext struct {
	tree            *fieldPathNode
	topLevelAllowed map[string]bool
	omittedFields   map[string]bool
}

func (ctx fieldTreeFilterContext) childContext(childTree *fieldPathNode) fieldTreeFilterContext {
	return fieldTreeFilterContext{tree: childTree}
}

func (ctx fieldTreeFilterContext) isAllowedFallbackField(key string) bool {
	if ctx.topLevelAllowed == nil {
		return false
	}
	return ctx.topLevelAllowed[key]
}

func (ctx fieldTreeFilterContext) isOmitted(key string) bool {
	if ctx.omittedFields == nil {
		return false
	}
	return ctx.omittedFields[key]
}

func filterExampleRootMapping(mapping *yaml.Node, spec compiledExampleFilterSpec) {
	var filtered []*yaml.Node
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		valueNode := mapping.Content[i+1]
		sectionKey := keyNode.Value
		if !spec.allowedSections[sectionKey] {
			continue
		}

		sectionSpec, hasFilter := spec.sectionFilters[sectionKey]
		if hasFilter && valueNode.Kind == yaml.MappingNode {
			switch sectionSpec.mode {
			case ExampleFilterModeKeepAll:
				filterOmittedTopLevelFields(valueNode, sectionSpec.omittedFields)
			case ExampleFilterModeFields:
				ctx := fieldTreeFilterContext{
					tree:            sectionSpec.fieldTree,
					topLevelAllowed: sectionSpec.topLevelFields,
					omittedFields:   sectionSpec.omittedFields,
				}
				filterMappingByFieldTree(valueNode, ctx)
			}
		}

		filtered = append(filtered, keyNode, valueNode)
	}
	mapping.Content = filtered
}

func filterOmittedTopLevelFields(mapping *yaml.Node, omittedFields map[string]bool) {
	if mapping == nil || mapping.Kind != yaml.MappingNode || len(omittedFields) == 0 {
		return
	}
	var filtered []*yaml.Node
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		valueNode := mapping.Content[i+1]
		if omittedFields[keyNode.Value] {
			continue
		}
		filtered = append(filtered, keyNode, valueNode)
	}
	mapping.Content = filtered
}

func filterMappingByFieldTree(mapping *yaml.Node, ctx fieldTreeFilterContext) {
	if mapping == nil || mapping.Kind != yaml.MappingNode || ctx.tree == nil {
		return
	}

	var filtered []*yaml.Node
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keyNode := mapping.Content[i]
		valueNode := mapping.Content[i+1]
		key := keyNode.Value

		child, ok := ctx.tree.children[key]
		if !ok {
			if ctx.isAllowedFallbackField(key) && !ctx.isOmitted(key) {
				filtered = append(filtered, keyNode, valueNode)
			}
			continue
		}

		if child.allowLeaf && !ctx.isOmitted(key) {
			filtered = append(filtered, keyNode, valueNode)
			continue
		}

		if len(child.children) == 0 || valueNode.Kind != yaml.MappingNode {
			continue
		}

		filterMappingByFieldTree(valueNode, ctx.childContext(child))
		if len(valueNode.Content) == 0 {
			continue
		}
		filtered = append(filtered, keyNode, valueNode)
	}
	mapping.Content = filtered
}
