package schema

// ReadFileParameters は read_file の provider-facing parameter schema を返す。
func ReadFileParameters(maxItems int) map[string]interface{} {
	return schemaObject(map[string]interface{}{
		"paths":   schemaStringArrayProperty("Files to read (1-10). Supports range syntax like file.go:10-20.", maxItems),
		"targets": schemaStringProperty("Locator IDs to read (e.g. '[L1,L5]'). Alternative to paths."),
		"detail":  schemaStringEnumProperty("Optional detail: auto (default); compact for locator targets or explicit path ranges; full; or outline.", "auto", "compact", "full", "outline"),
	})
}

// WriteFileParameters は write_file の provider-facing parameter schema を返す。
func WriteFileParameters() map[string]interface{} {
	return schemaObject(map[string]interface{}{
		"path":    schemaSinglePathProperty("File path to write to"),
		"content": schemaStringProperty("Content to write to the file"),
	}, "path", "content")
}

// DeleteFileParameters は delete_file の provider-facing parameter schema を返す。
func DeleteFileParameters() map[string]interface{} {
	return schemaObject(map[string]interface{}{
		"path": schemaSinglePathProperty("File path to delete"),
	}, "path")
}

// StrReplaceParameters は str_replace の provider-facing parameter schema を返す。
func StrReplaceParameters() map[string]interface{} {
	return schemaObject(map[string]interface{}{
		"path":       schemaSinglePathProperty("File path to edit"),
		"old_str":    schemaStringProperty("Exact string to find and replace"),
		"new_str":    schemaStringProperty("New string to replace with"),
		"start_line": schemaStringProperty("Start line number to limit search scope (optional)"),
		"end_line":   schemaStringProperty("End line number to limit search scope (optional)"),
		"edits":      schemaEditsArrayProperty("Batch edits: array of {old_str, new_str} pairs applied sequentially"),
	}, "path")
}

// ListDirParameters は list_dir の provider-facing parameter schema を返す。
func ListDirParameters() map[string]interface{} {
	return schemaObject(map[string]interface{}{
		"path":  schemaSinglePathProperty("Directory path to list"),
		"depth": schemaIntegerProperty("Recursion depth (default: 1, max: 3)"),
	}, "path")
}

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

func schemaSinglePathProperty(description string) map[string]interface{} {
	return schemaStringProperty(description)
}
