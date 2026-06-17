package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// handleProviderCommand はプロバイダーを切り替える。
func handleProviderCommand(agent *Agent, args []string) bool {
	return handleProviderSwitchCommand(agent, args, "/provider")
}

// handleUseCommand は後方互換のプロバイダー切り替えコマンドを処理する。
func handleUseCommand(agent *Agent, args []string) bool {
	return handleProviderSwitchCommand(agent, args, "/use")
}

func handleProviderSwitchCommand(agent *Agent, args []string, commandName string) bool {
	providerList := strings.Join(api.ListProviders(), ", ")
	out := agent.output()

	if len(args) == 0 {
		yellow.Fprintf(out, "Usage: %s <provider> [model]\n", commandName)
		yellow.Fprintf(out, "Available providers: %s\n", providerList)
		yellow.Fprintf(out, "Example: %s gemini gemini-3.5-flash\n", commandName)
		return true
	}

	providerName := args[0]
	requestedModel := ""
	if len(args) >= 2 {
		requestedModel = args[1]
	}

	// レジストリで登録済みかチェック
	if !api.IsRegisteredProvider(providerName) {
		red.Fprintf(out, "Unknown provider: %s\n", providerName)
		yellow.Fprintf(out, "Available providers: %s\n", providerList)
		return true
	}

	// 既に同じプロバイダーの場合でも、モデルが指定されていれば切り替え
	state := agent.CurrentProviderModelState()
	requestedProviderConfigKey := config.ActiveProviderConfigKey(providerName)
	if len(args) < 2 && requestedProviderConfigKey != "" && requestedProviderConfigKey == state.ProviderConfigKey {
		yellow.Fprintf(out, "Already using %s (model: %s)\n", providerName, state.CurrentModel)
		yellow.Fprintf(out, "Hint: Use '%s <provider> <model>' to change model\n", commandName)
		return true
	}

	_ = switchProviderModelWithOutput(agent, providerName, requestedModel)
	return true
}

func switchProviderModelWithOutput(agent *Agent, providerName, requestedModel string) error {
	providerList := strings.Join(api.ListProviders(), ", ")
	out := agent.output()

	if !api.IsRegisteredProvider(providerName) {
		red.Fprintf(out, "Unknown provider: %s\n", providerName)
		yellow.Fprintf(out, "Available providers: %s\n", providerList)
		return fmt.Errorf("unknown provider: %s", providerName)
	}

	outcome, err := agent.SwitchProviderModel(providerName, requestedModel)
	if err != nil {
		red.Fprintf(out, "❌ %v\n", err)

		// API キー設定方法を表示
		if instructions := config.ProviderSetupInstructions(providerName); len(instructions) > 0 {
			yellow.Fprintln(out, "\n設定方法:")
			for _, line := range instructions {
				yellow.Fprintf(out, "  %s\n", line)
			}
		}
		return err
	}

	printProviderSwitchOutcome(agent, outcome)
	if agent.CurrentProvider != nil {
		printContextSize(agent)
	}

	return nil
}
