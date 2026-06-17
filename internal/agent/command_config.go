package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/commandruntime"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// handleConfigCommand は設定の表示・変更を処理
func handleConfigCommand(agent *Agent, args []string) bool {
	out := agent.output()

	cfg, err := loadConfigForCommand()
	if err != nil {
		red.Fprintf(out, "Failed to load config: %v\n", err)
		return true
	}

	// /config show → 全設定をデフォルトとの差分付きで表示
	if len(args) > 0 && args[0] == "show" {
		_, _ = fmt.Fprint(out, showConfigForCommand(cfg))
		return true
	}

	// /config model <model-name> → モデル変更
	if len(args) >= 2 && args[0] == "model" {
		newModel := args[1]
		if err := validateConfigModelChange(agent, cfg, newModel); err != nil {
			red.Fprintf(out, "Error: %v\n", err)
			return true
		}

		// 設定更新
		cfg.DefaultModel = newModel

		// プロバイダー別の設定がある場合はそちらも更新
		if agent != nil {
			agent.SyncDefaultModelToProvider(cfg)
		}

		if err := saveConfigForCommand(cfg); err != nil {
			red.Fprintf(out, "Failed to save config: %v\n", err)
			return true
		}

		if agent != nil {
			agent.setRuntimeConfig(cfg)
			agent.SyncWithRuntimeConfig()
		}

		green.Fprintf(out, "✅ Default model updated to: %s\n", newModel)
		return true
	}

	// 引数なし → 対話式メニュー
	runInteractiveConfig(agent, cfg)
	return true
}

// isNonInteractiveConfigSubcommand は stdin を読まずに処理できる /config サブコマンドかを返す。
func isNonInteractiveConfigSubcommand(args []string) bool {
	return commandruntime.IsNonInteractiveConfigSubcommand(args)
}

var (
	loadConfigForCommand    = config.LoadConfig
	saveConfigForCommand    = config.SaveConfig
	showConfigForCommand    = config.ShowConfig
	setFieldValueForCommand = config.SetFieldValue
)
