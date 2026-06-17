package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// handleModelCommand はモデルの表示・切り替えを処理
func handleModelCommand(agent *Agent, args []string) bool {
	out := agent.output()

	// 引数なし → 現在のモデルとプロバイダーを表示
	if len(args) == 0 {
		state := agent.CurrentProviderModelState()
		_, _ = fmt.Fprintf(out, "🤖 Current model: %s\n", state.CurrentModel)
		_, _ = fmt.Fprintf(out, "🌐 Provider: %s\n", state.CurrentProvider)
		yellow.Fprintln(out, "\nUsage: /model <model-name>")
		yellow.Fprintln(out, "Enter any model name supported by your provider.")

		// ModelLister対応プロバイダーの場合、インストール済みモデルを表示
		if modelLister, ok := agent.CurrentProvider.(api.ModelLister); ok {
			models, err := modelLister.ListModels()
			if err != nil {
				yellow.Fprintf(out, "\nWarning: Could not list models: %v\n", err)
			} else if len(models) > 0 {
				yellow.Fprintln(out, "\nInstalled models:")
				for _, model := range models {
					_, _ = fmt.Fprintf(out, "  - %s\n", model)
				}
			}
		}
		return true
	}

	// /model <model-name> → モデル切り替え
	_ = switchModelForCurrentProviderWithOutput(agent, args[0])
	return true
}

func switchModelForCurrentProviderWithOutput(agent *Agent, newModel string) error {
	out := agent.output()
	outcome := agent.SwitchModelForCurrentProvider(newModel)
	if outcome.ValidationErr != nil {
		red.Fprintf(out, "Error: %v\n", outcome.ValidationErr)
		return outcome.ValidationErr
	}

	green.Fprintf(out, "✅ Model switched: %s → %s\n", outcome.OldModel, outcome.NewModel)
	printRuntimeSwitchContextNotice(agent, outcome.ContextNotice)
	if agent.CurrentProvider != nil {
		printContextSize(agent)
	}

	if outcome.LoadConfigErr != nil {
		yellow.Fprintf(out, "Warning: Failed to load config: %v\n", outcome.LoadConfigErr)
		return outcome.LoadConfigErr
	}

	if outcome.SaveConfigErr != nil {
		yellow.Fprintf(out, "Warning: Failed to save config: %v\n", outcome.SaveConfigErr)
		yellow.Fprintln(out, "Model switched for this session only")
		return outcome.SaveConfigErr
	}

	green.Fprintln(out, "💾 Default model saved to config")
	return nil
}
