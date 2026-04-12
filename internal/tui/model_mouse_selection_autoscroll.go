package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// mouseAutoScrollMsg はマウスドラッグ中の自動スクロールティック。
type mouseAutoScrollMsg struct{}

const (
	autoScrollEdgeZone = 3                     // viewport 端からの行数
	autoScrollInterval = 50 * time.Millisecond // 自動スクロールの間隔
	autoScrollMaxSpeed = 5                     // 最大スクロール速度（行/tick）
)

// autoScrollAmount は現在のマウス位置からのスクロール量を返す。
// 正の値は下方向、負の値は上方向。端への近さに応じて加速する。
func (m Model) autoScrollAmount() int {
	y := m.mouseLastScreenY
	h := m.vp.height
	if h <= 0 {
		return 0
	}

	if y < autoScrollEdgeZone {
		speed := autoScrollEdgeZone - y
		if speed > autoScrollMaxSpeed {
			speed = autoScrollMaxSpeed
		}
		return -speed
	}
	if y >= h-autoScrollEdgeZone {
		speed := y - (h - autoScrollEdgeZone) + 1
		if speed > autoScrollMaxSpeed {
			speed = autoScrollMaxSpeed
		}
		return speed
	}
	return 0
}

// handleAutoScroll はドラッグ中の自動スクロールを処理する。
// viewport をスクロールし、選択端を更新して次のティックを発行する。
func (m *Model) handleAutoScroll() tea.Cmd {
	if !m.mouseDragging {
		m.mouseAutoScrolling = false
		return nil
	}

	amount := m.autoScrollAmount()
	if amount == 0 {
		m.mouseAutoScrolling = false
		return nil
	}

	if amount < 0 {
		m.vp.scrollUp(-amount)
	} else {
		m.vp.scrollDown(amount)
	}

	m.updateMouseSelectionEnd(m.mouseLastScreenX, m.mouseLastScreenY)
	m.afterViewportScroll()
	return mouseAutoScrollTick()
}

func mouseAutoScrollTick() tea.Cmd {
	return tea.Tick(autoScrollInterval, func(t time.Time) tea.Msg {
		return mouseAutoScrollMsg{}
	})
}
