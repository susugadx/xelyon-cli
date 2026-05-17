package agent

import (
	"fmt"
	"io"
)

// handlePlanCommand は Plan Mode の切り替え
func handlePlanCommand(agent *Agent, args []string) bool {
	out := agent.output()

	if len(args) > 0 {
		switch args[0] {
		case "on":
			agent.setPlanModeEnabled(true)
			green.Fprintln(out, "✅ Plan Mode ON - 調査→計画→承認→実装")
			return true
		case "off":
			agent.setPlanModeEnabled(false)
			green.Fprintln(out, "✅ Plan Mode OFF - 通常モード")
			return true
		case "toggle":
			if agent.PlanModeEnabled {
				agent.setPlanModeEnabled(false)
				green.Fprintln(out, "✅ Plan Mode OFF - 通常モード")
			} else {
				agent.setPlanModeEnabled(true)
				green.Fprintln(out, "✅ Plan Mode ON - 調査→計画→承認→実装")
			}
			return true
		case "status":
			printPlanModeStatus(out, agent.PlanModeEnabled)
			return true
		}
	}

	// 引数なし：現在のステータスを表示
	printPlanModeStatus(out, agent.PlanModeEnabled)
	return true
}

func printPlanModeStatus(out io.Writer, enabled bool) {
	if enabled {
		cyan.Fprintln(out, "📋 Plan Mode: ON")
		_, _ = fmt.Fprintln(out, "   調査 → 計画 → 承認後、同じターンで通常モード実装へ進む")
	} else {
		cyan.Fprintln(out, "📋 Plan Mode: OFF")
		_, _ = fmt.Fprintln(out, "   通常モード（ツール個別確認）")
	}
	_, _ = fmt.Fprintln(out, "   切替: /plan toggle（明示指定: /plan on / /plan off）")
}
