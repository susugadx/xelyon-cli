package prompt

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
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
	return ResolveEditToolModeWithConfig(providerName, modelName, nil)
}

// ResolveEditToolModeWithConfig は provider/model と環境変数に基づいて編集ツールモードを解決します。
func ResolveEditToolModeWithConfig(providerName string, modelName string, cfg *config.Config) EditToolMode {
	if env := strings.TrimSpace(os.Getenv("XELYON_EDIT_TOOL")); env != "" {
		return NormalizeEditToolMode(env)
	}

	catalogModel := ""
	if cfg != nil {
		catalogModel = cfg.ModelCatalogName(providerName, modelName)
	}
	route := llmcatalog.ResolveProviderRoute(providerName, modelName, catalogModel)
	if route.EditToolFamily == "apply_patch" {
		return EditToolModeApplyPatch
	}
	return EditToolModeLegacy
}
