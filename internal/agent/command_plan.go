package agent

import "fmt"

// handlePlanCommand は Plan Mode の切り替え
func handlePlanCommand(agent *Agent, args []string) bool {
	out := agent.output()

	if len(args) > 0 {
		switch args[0] {
		case "on":
			agent.setPlanModeEnabled(true)
			green.Fprintln(out, "✅ Plan Mode ON - 調査→計画→承認（planning only）")
			return true
		case "off":
			agent.setPlanModeEnabled(false)
			green.Fprintln(out, "✅ Plan Mode OFF - 通常モード")
			return true
		case "status":
			if agent.PlanModeEnabled {
				cyan.Fprintln(out, "📋 Plan Mode: ON")
				_, _ = fmt.Fprintln(out, "   調査 → 計画 → 承認。実装は次の通常ターンで行う")
			} else {
				cyan.Fprintln(out, "📋 Plan Mode: OFF")
				_, _ = fmt.Fprintln(out, "   通常モード（ツール個別確認）")
			}
			return true
		}
	}

	// 引数なし：現在のステータスを表示
	if agent.PlanModeEnabled {
		cyan.Fprintln(out, "📋 Plan Mode: ON")
		_, _ = fmt.Fprintln(out, "   調査 → 計画 → 承認。実装は次の通常ターンで行う")
	} else {
		cyan.Fprintln(out, "📋 Plan Mode: OFF")
		_, _ = fmt.Fprintln(out, "   通常モード（ツール個別確認）")
	}
	return true
}
