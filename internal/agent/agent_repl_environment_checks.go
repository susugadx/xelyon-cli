package agent

import "github.com/susugadx/xelyon-cli/internal/tools/common"

// checkRipgrepAvailability は ripgrep の有無をチェックし、未インストール時に案内を表示する。
func checkRipgrepAvailability(agent *Agent) {
	if agent == nil || common.IsRipgrepAvailable() {
		return
	}

	out := agent.output()
	yellow.Fprintln(out, "⚠️  ripgrep (rg) not found — Project Map disabled, search_code using grep fallback")
	dim.Fprintln(out, "   Install for better performance:")
	dim.Fprintln(out, "     Ubuntu/Debian : sudo apt install ripgrep")
	dim.Fprintln(out, "     macOS         : brew install ripgrep")
	dim.Fprintln(out, "     Windows       : winget install BurntSushi.ripgrep")
	dim.Fprintln(out, "     Other         : https://github.com/BurntSushi/ripgrep#installation")
}
