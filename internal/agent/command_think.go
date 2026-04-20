package agent

import (
	"fmt"
	"strings"
)

// isCodexModel は Codex モデルかどうかを判定
// Codex モデルは reasoning が必須（"none" 非サポート）
func isCodexModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "codex")
}

// handleThinkCommand は Extended Thinking モードの切り替え
func handleThinkCommand(agent *Agent, args []string) bool {
	cfg := agent.cfg()
	isCodex := agent != nil && isCodexModel(agent.CurrentModel)
	out := agent.output()

	if len(args) == 0 {
		// 現在の状態を表示
		status := "OFF"
		if cfg.Thinking.Enabled {
			status = fmt.Sprintf("ON (level: %s)", cfg.Thinking.Level)
		} else if isCodex {
			status = "low (Codex minimum)"
		}
		_, _ = fmt.Fprintf(out, "🧠 Thinking Mode: %s\n", status)
		return true
	}

	isDeepSeek := agent != nil && strings.ToLower(agent.ProviderName) == "deepseek"

	switch args[0] {
	case "on":
		cfg.Thinking.Enabled = true
		// DeepSeek: モデル名で思考が決まるため reasoner に切り替え
		if isDeepSeek && agent != nil {
			agent.setCurrentModelAndSync("deepseek-reasoner")
		}
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
			// DeepSeek: reasoner → chat にフォールバック
			if isDeepSeek && agent != nil && agent.CurrentModel == "deepseek-reasoner" {
				agent.setCurrentModelAndSync("deepseek-chat")
			}
			green.Fprintln(out, "🧠 Thinking Mode: OFF")
			if isDeepSeek {
				green.Fprintf(out, "   Model: %s\n", agent.CurrentModel)
			}
		}
	case "low", "medium", "high", "xhigh":
		cfg.Thinking.Enabled = true
		cfg.Thinking.Level = args[0]
		// DeepSeek: モデル名で思考が決まるため reasoner に切り替え
		if isDeepSeek && agent != nil {
			agent.setCurrentModelAndSync("deepseek-reasoner")
		}
		green.Fprintf(out, "🧠 Thinking Mode: ON (level: %s)\n", args[0])
		if isDeepSeek {
			green.Fprintf(out, "   Model: %s\n", agent.CurrentModel)
		}
	default:
		yellow.Fprintln(out, "Usage: /think [on|off|low|medium|high|xhigh]")
	}
	return true
}
