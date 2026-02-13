package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/file"
)

// shouldThrottleWrite は書き込み制限の対象かどうかを判定する。
// Gemini プロバイダーのみ適用。
func (a *Agent) shouldThrottleWrite(tc *tools.ToolCall) bool {
	if strings.ToLower(a.ProviderName) != "gemini" {
		return false
	}
	return tools.IsWriteTool(tc.Tool)
}

// autoReadBack は書き込み成功後にファイルを自動で読み返す。
// ツール結果の History エントリに直接追記し、AIが変更後の状態を確認できるようにする。
func (a *Agent) autoReadBack(tc *tools.ToolCall) {
	path := extractPathFromToolCall(tc)
	if path == "" {
		return
	}

	// delete_file, move_file: ファイルが存在しないため読み返し不要
	// grep_replace: 複数ファイル対象で結果にサマリーが含まれるため不要
	switch tc.Tool {
	case "delete_file", "move_file", "grep_replace":
		return
	}

	content := file.ExecuteReadFile(path, 0, 0)

	// 最後の History エントリ（ツール結果）に追記
	if len(a.History) > 0 {
		last := &a.History[len(a.History)-1]
		last.Content += fmt.Sprintf("\n\n[Current file content after %s]\n%s", tc.Tool, content)
	}
}

// injectWriteThrottleMessage はスキップされた書き込みについて通知する。
// 最後の History エントリに追記する（FC モードの role 交互パターンを崩さないため）。
func (a *Agent) injectWriteThrottleMessage(skippedCount int) {
	msg := fmt.Sprintf(
		"\n\n[System] %d write operation(s) were queued but not executed. "+
			"Only 1 write per turn is allowed. "+
			"Review the file content above, then execute the next edit.",
		skippedCount,
	)

	if len(a.History) > 0 {
		last := &a.History[len(a.History)-1]
		last.Content += msg
	}
}

// extractPathFromToolCall はツール呼び出しからファイルパスを取得する。
func extractPathFromToolCall(tc *tools.ToolCall) string {
	if p := tc.Args["path"]; p != "" {
		return p
	}
	if p := tc.Args["dest"]; p != "" {
		return p
	}
	return ""
}
