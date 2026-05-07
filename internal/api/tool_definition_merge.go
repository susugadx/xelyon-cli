package api

import "context"

type additionalToolDefinitionsDisabledContextKey struct{}

// WithAdditionalToolDefinitionsDisabled は provider state などの追加 tool 定義を request 単位で無効化する。
func WithAdditionalToolDefinitionsDisabled(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, additionalToolDefinitionsDisabledContextKey{}, true)
}

// ToolDefinitionsWithAdditional は context の tool surface に追加 tool 定義を重複なしで結合する。
func ToolDefinitionsWithAdditional(ctx context.Context, additional []ToolDefinition) []ToolDefinition {
	base := ToolDefinitionsFromContext(ctx)
	if additionalToolDefinitionsDisabled(ctx) {
		return base
	}
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

func additionalToolDefinitionsDisabled(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	disabled, _ := ctx.Value(additionalToolDefinitionsDisabledContextKey{}).(bool)
	return disabled
}

func cloneToolDefinition(def ToolDefinition) ToolDefinition {
	cloned := def
	cloned.Parameters = cloneToolDefinitionMap(def.Parameters)
	return cloned
}
