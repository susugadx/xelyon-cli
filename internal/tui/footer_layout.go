package tui

const minChatViewportHeight = 1

// maxFooterExpansionRows は固定 footer 以外に追加表示できる行数を返す。
// chat viewport は最低 1 行残す。
func (m Model) maxFooterExpansionRows() int {
	if m.height <= 0 {
		return 0
	}
	return max(0, m.height-statusBarHeight-inputHeight-minChatViewportHeight)
}
