package api

import "context"

type compactedInputItemsContextKey struct{}

// WithCompactedInputItems は Compact API の圧縮済み input items を request context に注入する。
func WithCompactedInputItems(ctx context.Context, items []InputItem) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(items) == 0 {
		return ctx
	}
	return context.WithValue(ctx, compactedInputItemsContextKey{}, CloneInputItems(items))
}

// CompactedInputItemsFromContext は request context に注入された圧縮済み input items を返す。
func CompactedInputItemsFromContext(ctx context.Context) []InputItem {
	if ctx == nil {
		return nil
	}
	items, ok := ctx.Value(compactedInputItemsContextKey{}).([]InputItem)
	if !ok {
		return nil
	}
	return CloneInputItems(items)
}

// CloneInputItems は Responses/Compact API input item 群を defensive copy する。
func CloneInputItems(items []InputItem) []InputItem {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]InputItem, len(items))
	for i, item := range items {
		cloned[i] = cloneInputItem(item)
	}
	return cloned
}

func cloneInputItem(item InputItem) InputItem {
	cloned := item
	cloned.Content = cloneInputItemContent(item.Content)
	cloned.ThoughtParts = cloneInputItemThoughtParts(item.ThoughtParts)
	cloned.Summary = cloneInputItemSummary(item.Summary)
	return cloned
}

func cloneInputItemContent(content interface{}) interface{} {
	switch typed := content.(type) {
	case []InputContentPart:
		return append([]InputContentPart(nil), typed...)
	default:
		return cloneAnyValue(typed)
	}
}

func cloneInputItemThoughtParts(parts []map[string]any) []map[string]any {
	if len(parts) == 0 {
		return nil
	}
	cloned := make([]map[string]any, len(parts))
	for i, part := range parts {
		if part == nil {
			continue
		}
		cloned[i] = cloneAnyMap(part)
	}
	return cloned
}

func cloneInputItemSummary(summary []map[string]any) []map[string]any {
	if len(summary) == 0 {
		return nil
	}
	cloned := make([]map[string]any, len(summary))
	for i, item := range summary {
		if item == nil {
			continue
		}
		cloned[i] = cloneAnyMap(item)
	}
	return cloned
}
