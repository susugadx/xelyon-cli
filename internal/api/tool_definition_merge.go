package api

import "context"

// ToolDefinitionsWithAdditional は context の tool surface に追加 tool 定義を重複なしで結合する。
func ToolDefinitionsWithAdditional(ctx context.Context, additional []ToolDefinition) []ToolDefinition {
	base := ToolDefinitionsFromContext(ctx)
	result := make([]ToolDefinition, 0, len(base)+len(additional))
	seen := make(map[string]bool, len(base)+len(additional))

	for _, def := range base {
		if seen[def.Name] {
			continue
		}
		seen[def.Name] = true
		result = append(result, cloneToolDefinition(def))
	}
	for _, def := range additional {
		if seen[def.Name] {
			continue
		}
		seen[def.Name] = true
		result = append(result, cloneToolDefinition(def))
	}
	return result
}

func cloneToolDefinition(def ToolDefinition) ToolDefinition {
	cloned := def
	cloned.Parameters = cloneToolDefinitionMap(def.Parameters)
	return cloned
}
