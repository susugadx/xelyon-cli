package agent

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/refactor"
)

// checkCodeHealthOnChange はファイル変更時にコード健全性をチェック
func (a *Agent) checkCodeHealthOnChange(filePath string) {
	cfg := config.GetGlobalConfig()

	// 健全性チェックが無効なら何もしない
	if !cfg.CodeHealth.Enabled || !cfg.CodeHealth.AutoSuggest {
		return
	}

	// ソースファイルのみチェック
	if !refactor.ShouldCheckHealth(filePath) {
		return
	}

	// 設定からチェック項目を決定
	healthCfg := refactor.HealthCheckConfig{
		Enabled:          true,
		MaxFileLines:     cfg.CodeHealth.MaxFileLines,
		MaxFunctionLines: cfg.CodeHealth.MaxFunctionLines,
		CheckFileSize:    containsString(cfg.CodeHealth.OnChange, "check_file_size"),
		CheckFuncSize:    containsString(cfg.CodeHealth.OnChange, "check_function_size"),
		CheckDuplication: containsString(cfg.CodeHealth.OnChange, "check_duplication"),
	}

	// デフォルト値の適用
	if healthCfg.MaxFileLines == 0 {
		healthCfg.MaxFileLines = 300
	}
	if healthCfg.MaxFunctionLines == 0 {
		healthCfg.MaxFunctionLines = 50
	}

	// 健全性チェック実行
	result := refactor.CheckFileHealth(filePath, healthCfg)
	if result == nil || !result.HasWarning {
		return
	}

	// 警告表示
	warning := refactor.FormatHealthWarning(result)
	yellow.Print(warning)
}

// containsString はスライスに文字列が含まれるかチェック
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
