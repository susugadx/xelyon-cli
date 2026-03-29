package prompt

import (
	"os"
	"strings"
)

// EditToolMode は編集ツール露出と編集ガイドの切り替えモードです。
type EditToolMode string

const (
	// EditToolModeApplyPatch は apply_patch を使用するモードです。
	EditToolModeApplyPatch EditToolMode = "apply_patch"
	// EditToolModeLegacy は str_replace / write_file / delete_file を使用するモードです。
	EditToolModeLegacy EditToolMode = "str_replace"
)

// NormalizeEditToolMode は文字列を既知の編集ツールモードへ正規化します。
func NormalizeEditToolMode(editTool string) EditToolMode {
	switch strings.ToLower(strings.TrimSpace(editTool)) {
	case "str_replace", "legacy":
		return EditToolModeLegacy
	case "", "apply_patch":
		return EditToolModeApplyPatch
	default:
		return EditToolModeApplyPatch
	}
}

// ResolveEditToolMode は provider/model と環境変数上書きに基づいて編集ツールモードを解決します。
func ResolveEditToolMode(providerName string, modelName string) EditToolMode {
	if env := strings.TrimSpace(os.Getenv("XELYON_EDIT_TOOL")); env != "" {
		return NormalizeEditToolMode(env)
	}

	provider := strings.ToLower(strings.TrimSpace(providerName))
	model := strings.ToLower(strings.TrimSpace(modelName))

	if provider == "openrouter" {
		switch {
		case strings.HasPrefix(model, "anthropic/"),
			strings.HasPrefix(model, "deepseek/"):
			return EditToolModeLegacy
		case strings.HasPrefix(model, "openai/"),
			strings.HasPrefix(model, "google/"),
			strings.HasPrefix(model, "gemini/"):
			return EditToolModeApplyPatch
		default:
			return EditToolModeApplyPatch
		}
	}

	if provider == "bedrock" {
		if strings.Contains(model, "anthropic") || strings.Contains(model, "claude") {
			return EditToolModeLegacy
		}
		return EditToolModeApplyPatch
	}

	switch provider {
	case "openai", "gemini", "google":
		return EditToolModeApplyPatch
	case "claude", "anthropic", "deepseek":
		return EditToolModeLegacy
	default:
		return EditToolModeApplyPatch
	}
}
