package agent

import (
	"fmt"
	"io"

	"github.com/susugadx/xelyon-cli/internal/token"
)

// checkTokenWarning はトークン使用率をチェックして警告を表示
// 自動圧縮が有効な場合は表示しない（自動圧縮が処理するため）
func (a *Agent) checkTokenWarning() {
	cfg := a.cfg()

	// 自動圧縮が有効な場合は警告をスキップ（自動圧縮が処理する）
	if cfg.Compression.Enabled {
		return
	}

	percentage := a.GetTokenUsagePercentage()

	if percentage > 90 {
		yellow.Fprintln(a.output(), "⚠️ Token usage is at 90%. Consider using /compress")
	} else if percentage > 80 {
		_, _ = fmt.Fprintln(a.output(), "💡 Token usage is high (80%). /compress available if needed")
	}
}

// handleTokenLimitErrorWithWriter はトークン上限エラー時にユーザーへ案内を表示する。
// out は非nil前提（caller が a.output() を渡す）。
func handleTokenLimitErrorWithWriter(out io.Writer, err error) {
	if !token.IsTokenLimitError(err) {
		return
	}

	red.Fprintln(out, "❌ Token limit exceeded")
	yellow.Fprintln(out, "💡 Try: /compress to reduce history")
	yellow.Fprintln(out, "💡 Or:  /clear to start fresh")
}
