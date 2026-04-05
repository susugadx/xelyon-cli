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

func schemaRequiredPathsParameters(maxItems int) map[string]interface{} {
	return schemaObject(map[string]interface{}{
		"paths":   schemaStringArrayProperty("Files to read (1-10). Returns full content. Do not re-read files already returned.", maxItems),
		"targets": schemaStringProperty("Locator IDs to read (e.g. '[L1,L5]'). Alternative to paths."),
	})
}

func schemaSinglePathProperty(description string) map[string]interface{} {
	return schemaStringProperty(description)
}
