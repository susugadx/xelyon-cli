package agent

import (
	"github.com/susugadx/xelyon-cli/internal/prompt"
)

const (
	// EditToolModeApplyPatch は apply_patch を使用するモード
	EditToolModeApplyPatch = string(prompt.EditToolModeApplyPatch)
	// EditToolModeLegacy は str_replace / write_file / delete_file を使用するモード
	EditToolModeLegacy = string(prompt.EditToolModeLegacy)
)

// ResolveEditToolMode はプロバイダーとモデルに応じて編集ツールのモードを解決します。
func ResolveEditToolMode(providerName string, modelName string) string {
	return string(prompt.ResolveEditToolMode(providerName, modelName))
}

func appendUniqueStrings(values []string, extras ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(extras))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, extra := range extras {
		if _, ok := seen[extra]; ok {
			continue
		}
		values = append(values, extra)
		seen[extra] = struct{}{}
	}
	return values
}

func filterStrings(values []string, excluded ...string) []string {
	blocked := make(map[string]struct{}, len(excluded))
	for _, value := range excluded {
		blocked[value] = struct{}{}
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := blocked[value]; ok {
			continue
		}
		result = append(result, value)
	}
	return result
}
