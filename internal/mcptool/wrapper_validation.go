package mcptool

import (
	"encoding/json"
	"fmt"
	"io"
)

// validateArgs は引数を検証する（簡易版）
func (w *Wrapper) validateArgs(out io.Writer, args map[string]string) error {
	if out == nil {
		out = io.Discard
	}

	// 空のスキーマの場合はスキップ
	if len(w.inputSchema) == 0 || string(w.inputSchema) == "null" {
		return nil
	}

	// JSONスキーマをパース（簡易実装）
	var schema map[string]any
	if err := json.Unmarshal(w.inputSchema, &schema); err != nil {
		// パースエラーは警告のみ
		fmt.Fprintf(out, "⚠️  Failed to parse input schema for tool %s: %v\n", w.toolName, err)
		return nil
	}

	if err := validateTopLevelRequiredArgs(out, w.toolName, schema, args); err != nil {
		return err
	}

	if properties, ok := schema["properties"].(map[string]any); ok && properties != nil {
		warnInvalidPropertySchemas(out, w.toolName, properties)

		if err := validateStructuredArgs(properties, args); err != nil {
			return err
		}
	}

	return nil
}

func warnInvalidPropertySchemas(out io.Writer, toolName string, properties map[string]any) {
	for propName, propInfo := range properties {
		propMap, ok := propInfo.(map[string]any)
		if !ok || propMap == nil {
			fmt.Fprintf(out, "⚠️  Warning: Invalid property schema for %s in tool %s\n", propName, toolName)
		}
	}
}

func validateTopLevelRequiredArgs(out io.Writer, toolName string, schema map[string]any, args map[string]string) error {
	requiredRaw, ok := schema["required"]
	if !ok {
		return nil
	}

	requiredList, ok := requiredRaw.([]any)
	if !ok {
		fmt.Fprintf(out, "⚠️  Warning: Invalid required schema for tool %s\n", toolName)
		return nil
	}

	for _, requiredArg := range requiredList {
		argName, ok := requiredArg.(string)
		if !ok || argName == "" {
			fmt.Fprintf(out, "⚠️  Warning: Invalid required argument entry for tool %s\n", toolName)
			continue
		}
		if _, hasArg := args[argName]; !hasArg {
			return fmt.Errorf("required argument '%s' is missing", argName)
		}
	}

	return nil
}

func validateStructuredArgs(properties map[string]any, args map[string]string) error {
	for argName, rawValue := range args {
		propMap, ok := properties[argName].(map[string]any)
		if !ok || propMap == nil {
			continue
		}
		propType, ok := propMap["type"].(string)
		if !ok || (propType != "array" && propType != "object") {
			continue
		}
		if _, err := parseStructuredArg(propType, argName, rawValue); err != nil {
			return err
		}
	}
	return nil
}

// ValidateArgs は MCP tool 実行前の簡易 argument validation を行う。
func (w *Wrapper) ValidateArgs(out io.Writer, args map[string]string) error {
	return w.validateArgs(out, args)
}
