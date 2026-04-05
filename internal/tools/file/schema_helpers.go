package file

func schemaObject(properties map[string]interface{}, required ...string) map[string]interface{} {
	schema := map[string]interface{}{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func schemaStringProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "string",
		"description": description,
	}
}

func schemaStringEnumProperty(description string, values ...string) map[string]interface{} {
	property := schemaStringProperty(description)
	if len(values) > 0 {
		property["enum"] = values
	}
	return property
}

func schemaIntegerProperty(description string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "integer",
		"description": description,
	}
}

func schemaStringArrayProperty(description string, maxItems int) map[string]interface{} {
	property := map[string]interface{}{
		"type":        "array",
		"items":       map[string]interface{}{"type": "string"},
		"description": description,
	}
	if maxItems > 0 {
		property["maxItems"] = maxItems
	}
	return property
}

func schemaObjectArrayProperty(description string, properties map[string]interface{}, required ...string) map[string]interface{} {
	return map[string]interface{}{
		"type":        "array",
		"items":       schemaObject(properties, required...),
		"description": description,
	}
}

func schemaEditsArrayProperty(description string) map[string]interface{} {
	return schemaObjectArrayProperty(description, map[string]interface{}{
		"old_str": map[string]interface{}{"type": "string"},
		"new_str": map[string]interface{}{"type": "string"},
	}, "old_str", "new_str")
}

func schemaReadFileParameters(maxItems int) map[string]interface{} {
	return schemaObject(map[string]interface{}{
		"paths":   schemaStringArrayProperty("Files to read (1-10). Supports range syntax like file.go:10-20.", maxItems),
		"targets": schemaStringProperty("Locator IDs to read (e.g. '[L1,L5]'). Alternative to paths."),
		"detail":  schemaStringEnumProperty("Optional detail: auto (default); compact for locator targets or explicit path ranges; full; or outline.", "auto", "compact", "full", "outline"),
	})
}

func schemaSinglePathProperty(description string) map[string]interface{} {
	return schemaStringProperty(description)
}
