package agent

import (
	"strings"
)

// isCodexModel は Codex モデルかどうかを判定
// Codex モデルは reasoning が必須（"none" 非サポート）
func isCodexModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "codex")
}

func isAgentCodexModel(agent *Agent) bool {
	if agent == nil {
		return false
	}
	cfg := agent.cfg()
	model := agent.CurrentModel
	if cfg != nil {
		model = cfg.ModelCatalogName(agent.activeModelProviderConfigKey(cfg), model)
	}
	return isCodexModel(model)
}

// handleThinkingCommand は Extended Thinking モードの切り替え
func handleThinkingCommand(agent *Agent, args []string) bool {
	cfg := agent.cfg()
	isCodex := isAgentCodexModel(agent)
	out := agent.output()

	if len(args) == 0 {
		yellow.Fprintln(out, "Usage: /thinking <on|off|low|medium|high|xhigh>")
		return true
	}

	isDeepSeek := agent != nil && strings.ToLower(agent.ProviderName) == "deepseek"

	switch args[0] {
	case "on":
		cfg.Thinking.Enabled = true
		green.Fprintf(out, "🧠 Thinking Mode: ON (level: %s)\n", cfg.Thinking.Level)
		if isDeepSeek {
			green.Fprintf(out, "   Model: %s\n", agent.CurrentModel)
		}
	case "off":
		if isCodex {
			// Codexモデルは reasoning 必須のため "low" にフォールバック
			cfg.Thinking.Enabled = false
			cfg.Thinking.Level = "low"
			yellow.Fprintln(out, "⚠️  Codexモデルは reasoning 必須のため low に設定しました")
			green.Fprintln(out, "🧠 Thinking Mode: low (Codex minimum)")
		} else {
			cfg.Thinking.Enabled = false
			green.Fprintln(out, "🧠 Thinking Mode: OFF")
			if isDeepSeek {
				green.Fprintf(out, "   Model: %s\n", agent.CurrentModel)
			}
		}
	case "low", "medium", "high", "xhigh":
		cfg.Thinking.Enabled = true
		cfg.Thinking.Level = args[0]
		green.Fprintf(out, "🧠 Thinking Mode: ON (level: %s)\n", args[0])
		if isDeepSeek {
			green.Fprintf(out, "   Model: %s\n", agent.CurrentModel)
		}
	default:
		yellow.Fprintln(out, "Usage: /thinking <on|off|low|medium|high|xhigh>")
	}
	return true
}
